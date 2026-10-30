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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GPMembership is a group policy membership row: one member (EntityID) of one
// group policy. The With methods return a copy, so T stays a plain value type.
type GPMembership[T any] interface {
	comparable
	api.APIResource
	EntityID() int
	WithEntityID(int) T
	GroupPolicyIDValue() string
	WithGroupPolicyID(string) T
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
func CreateGPMemberships[T GPMembership[T]](
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
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
		m = m.WithEntityID(entityID)
		item, err := api.CreateItem(client, m)

		if err != nil {
			diags.AddError(
				"Error adding item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}

		results = append(results, createdOrSent(item, m).WithGroupPolicyID(m.GroupPolicyIDValue()))
		needsProvision.Add(m.GroupPolicyIDValue())
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
func ReadGPMemberships[T GPMembership[T]](
	ctx context.Context,
	client *api.APIClient,
	state tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
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
		item, found, err := fetchGPMembership(client, m, entityID)
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
func fetchGPMembership[T GPMembership[T]](client *api.APIClient, m T, entityID int) (item T, found bool, err error) {
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

	return items[0].WithGroupPolicyID(m.GroupPolicyIDValue()), true, nil
}

// perClient memoises one value per API client, error included. A provider
// process has one client; the tests build many.
type perClient[V any] struct {
	mu sync.Mutex
	m  map[*api.APIClient]func() (V, error)
}

func (p *perClient[V]) get(client *api.APIClient, build func() (V, error)) (V, error) {
	p.mu.Lock()
	f, ok := p.m[client]
	if !ok {
		f = sync.OnceValues(build)
		if p.m == nil {
			p.m = map[*api.APIClient]func() (V, error){}
		}
		p.m[client] = f
	}
	p.mu.Unlock()
	return f()
}

// gpMembershipIndex maps each member to its memberships, for one membership
// type, across every group policy. The API has no "memberships of object X"
// endpoint, so import discovery lists them all instead of probing every group
// policy per imported object: ⌈G/100⌉ + Σ max(1, ⌈mᵢ/100⌉) calls per type, once
// per provider process, rather than R×G. The group policy list itself is read
// once and shared by all four types.
//
// The result, error included, is kept for the life of the provider process: one
// terraform plan with many import blocks shares it, and a failed build is not
// repeated for every import in the run. Only ImportState reads it. Refresh never
// does, so a stale index cannot affect drift detection.
type gpMembershipIndex[T GPMembership[T]] struct {
	perClient[map[int][]T]
}

var (
	groupPolicyLists perClient[[]api.GroupPolicy]

	jumpGroupMemberIndex    gpMembershipIndex[api.GroupPolicyJumpGroup]
	jumpointMemberIndex     gpMembershipIndex[api.GroupPolicyJumpoint]
	accountGroupMemberIndex gpMembershipIndex[api.GroupPolicyVaultAccountGroup]
	accountMemberIndex      gpMembershipIndex[api.GroupPolicyVaultAccount]
)

// membershipsOf returns entityID's memberships of every group policy.
func (idx *gpMembershipIndex[T]) membershipsOf(client *api.APIClient, entityID int) ([]T, error) {
	byEntity, err := idx.get(client, func() (map[int][]T, error) { return buildGPMembershipIndex[T](client) })
	return byEntity[entityID], err
}

func buildGPMembershipIndex[T GPMembership[T]](client *api.APIClient) (map[int][]T, error) {
	gps, err := groupPolicyLists.get(client, func() ([]api.GroupPolicy, error) {
		return api.ListAllItems[api.GroupPolicy](client, nil)
	})
	if err != nil {
		return nil, err
	}

	var zero T
	byEntity := map[int][]T{}
	key := func(m *T) string { return strconv.Itoa((*m).EntityID()) }
	for _, gp := range gps {
		if gp.ID == nil {
			continue
		}
		gpID := strconv.Itoa(*gp.ID)
		rows, err := api.ListAllItemsEndpoint(client, zero.WithGroupPolicyID(gpID).Endpoint(), nil, key)
		if api.IsNotFound(err) {
			// Deleted between the list and this fetch: it has no members.
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			byEntity[row.EntityID()] = append(byEntity[row.EntityID()], row.WithGroupPolicyID(gpID))
		}
	}
	return byEntity, nil
}

// importWithGPMemberships imports by ID, then writes the imported object's group
// policy memberships into state. Read refreshes only the memberships already in
// state, so without this an import leaves the attribute null and the next apply
// fails creating memberships that already exist.
//
// Nothing found leaves the attribute null, which is what a configuration that
// declares no memberships plans. An API account that cannot list group policies
// gets a warning and a null attribute. Any other failure, including a list that
// kept changing while it was read, fails the import rather than recording a
// partial set that the next apply would act on.
func importWithGPMemberships[TApi api.APIResource, TTf any, T GPMembership[T]](
	ctx context.Context,
	r *apiResource[TApi, TTf],
	index *gpMembershipIndex[T],
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	entityID, err := strconv.Atoi(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Could not parse import ID [%s] as an integer: %s", req.ID, err.Error()))
		return
	}

	members, err := index.membershipsOf(r.ApiClient, entityID)
	if api.HasStatus(err, http.StatusForbidden) {
		resp.Diagnostics.AddWarning(
			"group_policy_memberships were not read",
			"This API account can't list group policies, so the import leaves group_policy_memberships unset.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading group policy memberships", "Could not discover the imported object's group policy memberships: "+err.Error())
		return
	}
	if len(members) == 0 {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), members)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.AddWarning(
		fmt.Sprintf("Imported %d group policy memberships", len(members)),
		"A configuration that omits group_policy_memberships will remove them, revoking that access. "+
			"Declare them, generate the configuration with -generate-config-out, "+
			"or add lifecycle { ignore_changes = [group_policy_memberships] }.",
	)
}

// UpdateGPMemberships diffs the plan and state group policy memberships,
// removes stale ones, creates new ones, provisions affected group policies,
// and writes the combined results to state.
func UpdateGPMemberships[T GPMembership[T]](
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
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
		m = m.WithEntityID(entityID)
		tflog.Trace(ctx, "🌈 Deleting item", map[string]interface{}{
			"item": m,
		})
		endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), entityID)
		err := api.DeleteItemEndpoint(client, endpoint)

		if err != nil {
			diags.AddError(
				"Error updating item's group policy memberships",
				"Unexpected deleting membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}
		needsProvision.Add(m.GroupPolicyIDValue())
	}

	results := noChange.ToSlice()
	for _, m := range toAdd.ToSlice() {
		m = m.WithEntityID(entityID)
		item, err := api.CreateItem(client, m)

		if err != nil {
			diags.AddError(
				"Error updating item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}

		results = append(results, createdOrSent(item, m).WithGroupPolicyID(m.GroupPolicyIDValue()))
		needsProvision.Add(m.GroupPolicyIDValue())
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
