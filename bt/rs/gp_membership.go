package rs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

// provisionGroupPolicies provisions each unique group policy ID in the set.
func provisionGroupPolicies(
	client *api.APIClient,
	diags *diag.Diagnostics,
	needsProvision mapset.Set[string],
) {
	for id := range needsProvision.Iter() {
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
	for m := range toAdd.Iterator().C {
		setEntityID(&m, entityID)
		item, err := api.CreateItem(client, m)

		if err != nil {
			diags.AddError(
				"Error adding item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}

		result := *item
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
// it from the API, and writes the updated list back to state. API errors are
// logged and skipped rather than treated as failures.
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

	// Each membership read returns a JSON array scoped to this entity under the
	// given group policy (e.g. GET group-policy/<gp>/jump-group/<id> -> [{...}]).
	// Decode the array, take the entity's membership (re-applying the group policy
	// ID, which the response body omits), and drop memberships the API no longer
	// reports so an out-of-band removal surfaces as drift instead of stale state.
	refreshed := make([]T, 0, len(gpList))
	for _, m := range gpList {
		gpId := *getGroupPolicyID(&m)

		endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), entityID)
		items, err := api.ListItemsEndpoint[T](client, endpoint)
		if err != nil {
			if strings.Contains(err.Error(), "status: 404") {
				// Object-returning endpoints 404 for a removed membership (the
				// array-returning ones return an empty list, handled below). Drop
				// it either way so the removal surfaces as drift.
				tflog.Debug(ctx, "🌈 Membership not found, dropping from state", map[string]interface{}{"read": m})
				continue
			}
			// A genuine error (5xx/auth/network). Now that both response shapes
			// decode, this is real — surface it rather than masking it.
			diags.AddError(
				"Error reading group policy membership",
				fmt.Sprintf("Unexpected error refreshing membership at [%s]: %s", endpoint, err.Error()),
			)
			return
		}

		if len(items) == 0 {
			// No longer a member of this group policy; drop it from state.
			tflog.Debug(ctx, "🌈 Membership no longer present, dropping from state", map[string]interface{}{"read": m})
			continue
		}

		item := items[0]
		setGroupPolicyID(&item, &gpId)
		tflog.Trace(ctx, "🌈 Read item", map[string]interface{}{"read": item})
		refreshed = append(refreshed, item)
	}

	d = respState.SetAttribute(ctx, path.Root("group_policy_memberships"), refreshed)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
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
	for m := range toRemove.Iterator().C {
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
	for m := range toAdd.Iterator().C {
		setEntityID(&m, entityID)
		item, err := api.CreateItem(client, m)

		if err != nil {
			diags.AddError(
				"Error updating item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(entityID)+"]: "+err.Error(),
			)
			return
		}

		result := *item
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
