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

	refreshed := make([]T, 0, len(gpList))
	for _, m := range gpList {
		tflog.Trace(ctx, "🌈 Reading item", map[string]interface{}{
			"read": m,
		})
		gpId := *getGroupPolicyID(&m)

		endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), entityID)
		item, err := api.GetItemEndpoint[T](client, endpoint)

		if err != nil {
			if strings.Contains(err.Error(), "status: 404") {
				// The membership was removed out-of-band; drop it from state so the
				// drift surfaces on the next plan rather than being masked.
				tflog.Debug(ctx, "🌈 Membership no longer exists, dropping from state", map[string]interface{}{
					"read": m,
				})
				continue
			}
			// A genuine API error (5xx, auth, network). Surface it instead of
			// silently keeping stale state and reporting a clean refresh.
			diags.AddError(
				"Error reading group policy membership",
				fmt.Sprintf("Unexpected error refreshing membership at [%s]: %s", endpoint, err.Error()),
			)
			return
		}

		if item != nil {
			tflog.Trace(ctx, "🌈 Read item", map[string]interface{}{
				"read": *item,
			})
			setGroupPolicyID(item, &gpId)
			refreshed = append(refreshed, *item)
		} else {
			// No content but no error; keep the value already in state.
			refreshed = append(refreshed, m)
		}
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
