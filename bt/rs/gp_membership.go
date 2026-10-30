package rs

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"terraform-provider-sra/api"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GPMembership is the constraint for GP types used with the membership helpers.
type GPMembership interface {
	comparable
	api.APIResource
}

// createdOrSent returns the created membership, or — when api.CreateItem
// answers (nil, nil) on a 204 No Content — the request echoed back. A 204 means
// the membership WAS created; dropping it would leave a live group-policy
// entitlement invisible to Terraform and produce an inconsistent-result error
// on the next plan.
func createdOrSent[T any](item *T, sent T) T {
	if item == nil {
		return sent
	}
	return *item
}

// provisionGroupPolicies provisions each unique group policy ID in the set.
func provisionGroupPolicies(
	client *api.APIClient,
	diags *diag.Diagnostics,
	needsProvision mapset.Set[string],
) {
	for _, id := range needsProvision.ToSlice() {
		p := api.GroupPolicyProvision{
			GroupPolicyID: &id,
		}
		_, err := api.CreateItem(client, p)

		if err != nil {
			diags.AddError(
				"Error provisioning item's group policy memberships",
				"Unexpected response provisioning membership of item ID ["+*p.GroupPolicyID+"]: "+err.Error(),
			)
			return
		}
	}
}

// CreateGPMemberships reads group policy memberships from the plan, creates
// each one via the API, provisions the affected group policies, and writes
// the results to state.
func CreateGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
	setEntityID func(*T, int),
	getGroupPolicyID func(*T) *string,
	setGroupPolicyID func(*T, *string),
	mu *sync.Mutex,
) {
	var tfGPList types.Set
	d := plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	if tfGPList.IsNull() {
		return
	}

	var gpList []T
	d = tfGPList.ElementsAs(ctx, &gpList, false)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	toAdd := mapset.NewSet(gpList...)

	tflog.Trace(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
		"add":  toAdd,
		"tf":   tfGPList,
		"list": gpList,
	})

	mu.Lock()
	defer mu.Unlock()

	results := []T{}
	needsProvision := mapset.NewSet[string]()
	for _, m := range toAdd.ToSlice() {
		setEntityID(&m, entityID)
		item, err := api.CreateItem(client, m)

		if err != nil {
			diags.AddError(
				"Error adding item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}

		result := createdOrSent(item, m)
		setGroupPolicyID(&result, getGroupPolicyID(&m))
		results = append(results, result)
		needsProvision.Add(*getGroupPolicyID(&m))
	}

	provisionGroupPolicies(client, diags, needsProvision)
	if diags.HasError() {
		return
	}

	d = state.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}

// ReadGPMemberships reads each group policy membership from state, refreshes
// it from the API, and writes the updated list back to state. A membership
// the API no longer reports (or 404s) is dropped from state as drift; any
// other API error aborts with a diagnostic.
func ReadGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	state tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
	getGroupPolicyID func(*T) *string,
	setGroupPolicyID func(*T, *string),
) {
	var tfGPList types.Set
	d := state.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	if tfGPList.IsNull() {
		return
	}

	var gpList []T
	d = tfGPList.ElementsAs(ctx, &gpList, false)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	// The spec documents a single JSON object for this GET, but the appliance
	// has been observed returning a JSON array scoped to this entity instead
	// (e.g. GET group-policy/<gp>/jump-group/<id> -> [{...}]); ListItemsEndpoint
	// tolerates either shape. Decode the array, take the entity's membership
	// (re-applying the group policy ID, which the response body omits), and
	// drop memberships the API no longer reports so an out-of-band removal
	// surfaces as drift instead of stale state.
	refreshed := make([]T, 0, len(gpList))
	for _, m := range gpList {
		item, found, err := fetchGPMembership(client, m, entityID, getGroupPolicyID, setGroupPolicyID)
		if err != nil {
			diags.AddError("Error reading group policy membership", "Unexpected error "+err.Error())
			return
		}
		if !found {
			// No longer a member of this group policy; drop it from state.
			tflog.Debug(ctx, "🌈 Membership no longer present, dropping from state", map[string]interface{}{"read": m})
			continue
		}
		tflog.Trace(ctx, "🌈 Read item", map[string]interface{}{"read": item})
		refreshed = append(refreshed, item)
	}

	d = respState.SetAttribute(ctx, path.Root("group_policy_memberships"), refreshed)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}

// fetchGPMembership reads entityID's membership of m's group policy. found is
// false when the entity is no longer a member.
func fetchGPMembership[T GPMembership](
	client *api.APIClient,
	m T,
	entityID int,
	getGroupPolicyID func(*T) *string,
	setGroupPolicyID func(*T, *string),
) (item T, found bool, err error) {
	gpId := *getGroupPolicyID(&m)

	endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), entityID)
	items, err := api.ListItemsEndpoint[T](client, endpoint)
	if err != nil {
		if api.IsNotFound(err) {
			// Object-returning endpoints 404 for a removed membership (the
			// array-returning ones return an empty list, handled below). Drop
			// it either way so the removal surfaces as drift.
			return item, false, nil
		}
		// A genuine error (5xx/auth/network). Now that both response shapes
		// decode, this is real — surface it rather than masking it.
		return item, false, fmt.Errorf("refreshing membership at [%s]: %w", endpoint, err)
	}

	if len(items) == 0 {
		return item, false, nil
	}

	item = items[0]
	setGroupPolicyID(&item, &gpId)
	return item, true, nil
}

// gpRelation describes one kind of group policy membership: how to address a
// group policy's member index for it, and which member a row names.
type gpRelation[T GPMembership] struct {
	name     string
	entityID func(*T) int
	setGP    func(*T, *string)
}

// endpoint is the group policy's member index for this relation, built by the
// model's own Endpoint() so the path (and its escaping) is stated once.
func (r gpRelation[T]) endpoint(gpID string) string {
	var m T
	r.setGP(&m, &gpID)
	return m.Endpoint()
}

func derefID(id *int) int {
	if id == nil {
		return 0
	}
	return *id
}

// relationFor is the table of membership kinds, keyed by type, so a caller
// names the relation by the type it already uses and cannot pass a mismatched
// endpoint.
func relationFor[T GPMembership]() (gpRelation[T], bool) {
	var rel any
	switch any(*new(T)).(type) {
	case api.GroupPolicyJumpGroup:
		rel = gpRelation[api.GroupPolicyJumpGroup]{"jump-group",
			func(m *api.GroupPolicyJumpGroup) int { return derefID(m.JumpGroupID) },
			func(m *api.GroupPolicyJumpGroup, gp *string) { m.GroupPolicyID = gp }}
	case api.GroupPolicyJumpoint:
		rel = gpRelation[api.GroupPolicyJumpoint]{"jumpoint",
			func(m *api.GroupPolicyJumpoint) int { return derefID(m.JumpointID) },
			func(m *api.GroupPolicyJumpoint, gp *string) { m.GroupPolicyID = gp }}
	case api.GroupPolicyVaultAccountGroup:
		rel = gpRelation[api.GroupPolicyVaultAccountGroup]{"vault-account-group",
			func(m *api.GroupPolicyVaultAccountGroup) int { return derefID(m.AccountGroupID) },
			func(m *api.GroupPolicyVaultAccountGroup, gp *string) { m.GroupPolicyID = gp }}
	case api.GroupPolicyVaultAccount:
		rel = gpRelation[api.GroupPolicyVaultAccount]{"vault-account",
			func(m *api.GroupPolicyVaultAccount) int { return derefID(m.AccountID) },
			func(m *api.GroupPolicyVaultAccount, gp *string) { m.GroupPolicyID = gp }}
	}
	r, ok := rel.(gpRelation[T])
	return r, ok
}

// gpMembershipIndex maps each member to its memberships, for one relation,
// across every group policy. The API has no "memberships of object X" endpoint,
// so import discovery lists them all instead of probing every group policy per
// imported object: ⌈G/100⌉ + Σ max(1, ⌈mᵢ/100⌉) calls per relation, once per
// provider process, rather than R×G.
//
// Only ImportState reads it. Refresh never does, so a stale index cannot affect
// drift detection.
type gpMembershipIndex struct {
	once     sync.Once
	byEntity any // map[int][]T for the relation's T
	err      error
}

type gpIndexKey struct {
	client   *api.APIClient
	relation string
}

var (
	gpIndexesMu sync.Mutex
	gpIndexes   = map[gpIndexKey]*gpMembershipIndex{}
)

// gpMembershipsOf returns entityID's memberships of every group policy. The
// index is built on first use and its result, error included, is kept for the
// life of the provider process: one terraform plan with many import blocks
// shares it, and a failed build is not repeated for every import in the run.
func gpMembershipsOf[T GPMembership](client *api.APIClient, rel gpRelation[T], entityID int) ([]T, error) {
	gpIndexesMu.Lock()
	key := gpIndexKey{client, rel.name}
	idx, ok := gpIndexes[key]
	if !ok {
		idx = &gpMembershipIndex{}
		gpIndexes[key] = idx
	}
	gpIndexesMu.Unlock()

	idx.once.Do(func() {
		idx.byEntity, idx.err = buildGPMembershipIndex(client, rel)
	})
	if idx.err != nil {
		return nil, idx.err
	}
	return idx.byEntity.(map[int][]T)[entityID], nil
}

func buildGPMembershipIndex[T GPMembership](client *api.APIClient, rel gpRelation[T]) (map[int][]T, error) {
	gps, err := api.ListAllItems[api.GroupPolicy](client, nil)
	if err != nil {
		return nil, err
	}

	byEntity := map[int][]T{}
	key := func(m *T) string { return strconv.Itoa(rel.entityID(m)) }
	for _, gp := range gps {
		if gp.ID == nil {
			continue
		}
		gpID := strconv.Itoa(*gp.ID)
		rows, err := api.ListAllItemsEndpoint(client, rel.endpoint(gpID), nil, key)
		if api.IsNotFound(err) {
			// Deleted between the list and this fetch: it has no members.
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			rel.setGP(&row, &gpID)
			byEntity[rel.entityID(&row)] = append(byEntity[rel.entityID(&row)], row)
		}
	}
	return byEntity, nil
}

// ImportGPMemberships writes the imported object's group policy memberships
// into state. Read refreshes only the memberships already in state, so without
// this an import leaves the attribute null and the next apply fails creating
// memberships that already exist.
//
// Nothing found leaves the attribute null, which is what a configuration that
// declares no memberships plans. An API account that cannot list group policies
// gets a warning and a null attribute. Any other failure, including a list that
// kept changing while it was read, fails the import rather than recording a
// partial set that the next apply would act on.
func ImportGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	importID string,
	state *tfsdk.State,
	diags *diag.Diagnostics,
) {
	rel, ok := relationFor[T]()
	if !ok {
		diags.AddError("Unsupported group policy membership type", fmt.Sprintf("No group policy relation is defined for %T", *new(T)))
		return
	}

	entityID, err := strconv.Atoi(importID)
	if err != nil {
		diags.AddError("Invalid import ID", fmt.Sprintf("Could not parse import ID [%s] as an integer: %s", importID, err.Error()))
		return
	}

	members, err := gpMembershipsOf(client, rel, entityID)
	if api.HasStatus(err, http.StatusForbidden) {
		diags.AddWarning(
			"group_policy_memberships were not read",
			"This API account can't list group policies, so the import leaves group_policy_memberships unset.",
		)
		return
	}
	if err != nil {
		diags.AddError("Error reading group policy memberships", "Could not discover the imported object's group policy memberships: "+err.Error())
		return
	}
	if len(members) == 0 {
		return
	}

	d := state.SetAttribute(ctx, path.Root("group_policy_memberships"), members)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	diags.AddWarning(
		fmt.Sprintf("Imported %d group policy memberships", len(members)),
		"A configuration that omits group_policy_memberships will remove them, revoking that access. "+
			"Declare them, generate the configuration with -generate-config-out, "+
			"or add lifecycle { ignore_changes = [group_policy_memberships] }.",
	)
}

// UpdateGPMemberships diffs the plan and state group policy memberships,
// removes stale ones, creates new ones, provisions affected group policies,
// and writes the combined results to state.
func UpdateGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
	setEntityID func(*T, int),
	getGroupPolicyID func(*T) *string,
	setGroupPolicyID func(*T, *string),
	diffFunc func([]T, []T) (mapset.Set[T], mapset.Set[T], mapset.Set[T]),
	mu *sync.Mutex,
) {
	var tfGPList types.Set
	d := plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	var gpList []T
	if !tfGPList.IsNull() {
		d = tfGPList.ElementsAs(ctx, &gpList, false)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
	}

	var tfGPStateList types.Set
	d = reqState.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPStateList)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	var stateGPList []T
	if !tfGPStateList.IsNull() {
		d = tfGPStateList.ElementsAs(ctx, &stateGPList, false)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
	}

	if tfGPList.IsNull() && tfGPStateList.IsNull() {
		return
	}

	toAdd, toRemove, noChange := diffFunc(gpList, stateGPList)

	tflog.Trace(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
		"add":    toAdd,
		"remove": toRemove,

		"tf":    tfGPList,
		"list":  gpList,
		"state": stateGPList,
	})

	mu.Lock()
	defer mu.Unlock()

	needsProvision := mapset.NewSet[string]()
	for _, m := range toRemove.ToSlice() {
		setEntityID(&m, entityID)
		tflog.Trace(ctx, "🌈 Deleting item", map[string]interface{}{
			"item": m,
		})
		endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), entityID)
		err := api.DeleteItemEndpoint[T](client, endpoint)

		if err != nil {
			diags.AddError(
				"Error updating item's group policy memberships",
				"Unexpected deleting membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}
		needsProvision.Add(*getGroupPolicyID(&m))
	}

	results := noChange.ToSlice()
	for _, m := range toAdd.ToSlice() {
		setEntityID(&m, entityID)
		item, err := api.CreateItem(client, m)

		if err != nil {
			diags.AddError(
				"Error updating item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}

		result := createdOrSent(item, m)
		setGroupPolicyID(&result, getGroupPolicyID(&m))
		results = append(results, result)
		needsProvision.Add(*getGroupPolicyID(&m))
	}

	provisionGroupPolicies(client, diags, needsProvision)
	if diags.HasError() {
		return
	}

	if tfGPList.IsNull() {
		// The plan removed the attribute entirely (it is Optional, non-Computed on
		// jump_group/jumpoint, so a removed block reads as null). The memberships
		// were deleted above; write a null set — not an empty one — so the applied
		// state matches the null plan and Terraform does not report a "provider
		// produced inconsistent result after apply". tfGPStateList is guaranteed
		// non-null here (we passed the both-null short-circuit above).
		d = respState.SetAttribute(ctx, path.Root("group_policy_memberships"), types.SetNull(tfGPStateList.ElementType(ctx)))
	} else {
		d = respState.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
	}
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}
