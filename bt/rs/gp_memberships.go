package rs

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func updateGPMemberships(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Group Policy Memberships

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	var tfGPList types.Set
	diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var gpList []api.GroupPolicyVaultAccountGroup
	if !tfGPList.IsNull() {
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var tfGPStateList types.Set
	diags = req.State.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPStateList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateGPList []api.GroupPolicyVaultAccountGroup
	if !tfGPStateList.IsNull() {
		diags = tfGPStateList.ElementsAs(ctx, &stateGPList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	setGPList := mapset.NewSet(gpList...)
	setGPStateList := mapset.NewSet(stateGPList...)

	toAdd := setGPList.Difference(setGPStateList)
	toRemove := setGPStateList.Difference(setGPList)

	tflog.Info(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
		"add":    toAdd,
		"remove": toRemove,

		"tf":    tfGPList,
		"list":  gpList,
		"state": stateGPList,
	})

	for m := range toRemove.Iterator().C {
		m.AccountGroupID = &id
		tflog.Info(ctx, "🌈 Deleting item", map[string]interface{}{
			"add":     m,
			"gp":      m.GroupPolicyID,
			"account": m.AccountGroupID,
		})
		endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), *m.AccountGroupID)
		err := api.DeleteItemEndpoint[api.GroupPolicyVaultAccountGroup](r.ApiClient, endpoint)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating item's group policy memberships",
				"Unexpected deleting membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}
	}

	results := []api.GroupPolicyVaultAccountGroup{}
	for m := range toAdd.Iterator().C {
		m.AccountGroupID = &id
		item, err := api.CreateItem(r.ApiClient, m)
		item.GroupPolicyID = m.GroupPolicyID

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating item's group policy memberships",
				"Unexpected adding membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}
		results = append(results, *item)
	}

	diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
