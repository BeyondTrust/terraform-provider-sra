package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CreateAccountJIA reads the jump_item_association from the plan, creates
// it via the API, and writes the result to state.
func CreateAccountJIA(
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	accountID int,
) {
	var apiSub api.AccountJumpItemAssociation
	var tfObj types.Object
	d := plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	if tfObj.IsNull() {
		return
	}

	if tfObj.IsUnknown() {
		d = state.SetAttribute(ctx, path.Root("jump_item_association"), types.ObjectNull(tfObj.AttributeTypes(ctx)))
		diags.Append(d...)
		return
	}

	d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	apiSub.ID = &accountID
	tflog.Debug(ctx, fmt.Sprintf("🙀 Creating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
		"data": apiSub,
	})

	item, err := api.CreateItem(client, apiSub)

	if err != nil {
		diags.AddError(
			"Error Creating Account Jump Item Association",
			"Unexpected value for ID ["+strconv.Itoa(accountID)+"]: "+err.Error(),
		)
		return
	}

	// CreateItem returns (nil, nil) on a 204 No Content. Guard against writing a
	// nil association into state (which would produce an inconsistent-result
	// error) by falling back to an empty association, mirroring UpdateAccountJIA.
	if item != nil {
		rb, _ := json.Marshal(item)
		tflog.Debug(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})
		d = state.SetAttribute(ctx, path.Root("jump_item_association"), item)
	} else {
		var empty api.AccountJumpItemAssociation
		d = state.SetAttribute(ctx, path.Root("jump_item_association"), empty)
	}
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}

// ReadAccountJIA reads the jump_item_association from state, refreshes it
// from the API, and writes the result back to state.
func ReadAccountJIA(
	ctx context.Context,
	client *api.APIClient,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	accountID int,
) {
	var apiSub api.AccountJumpItemAssociation
	var tfObj types.Object
	d := reqState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	planIsGone := tfObj.IsNull() || tfObj.IsUnknown()

	if !planIsGone {
		d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		diags.Append(d...)
		if diags.HasError() {
			return
		}
	}

	apiSub.ID = &accountID
	tflog.Debug(ctx, fmt.Sprintf("🙀 Reading API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
		"data":          apiSub,
		"planIsNull":    tfObj.IsNull(),
		"planIsUnknown": tfObj.IsUnknown(),
	})

	item, err := api.GetItemEndpoint[api.AccountJumpItemAssociation](client, apiSub.Endpoint())

	var empty api.AccountJumpItemAssociation
	if item == nil && (planIsGone || apiSub.FilterType == "") {
		d = respState.SetAttribute(ctx, path.Root("jump_item_association"), empty)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		return
	}

	if err != nil {
		diags.AddError(
			"Error reading item",
			"Unexpected reading item ID ["+strconv.Itoa(accountID)+"]: "+err.Error(),
		)
		return
	}

	rb, _ := json.Marshal(item)
	tflog.Trace(ctx, "🙀 got item", map[string]interface{}{
		"data": string(rb),
	})
	d = respState.SetAttribute(ctx, path.Root("jump_item_association"), item)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}

// UpdateAccountJIA diffs the plan and state jump_item_association, then
// creates, updates, or deletes the association as needed and writes the
// result to state.
func UpdateAccountJIA(
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	accountID int,
) {
	var apiSub api.AccountJumpItemAssociation
	var tfObj types.Object
	d := plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	planIsGone := tfObj.IsNull() || tfObj.IsUnknown()

	if !planIsGone {
		d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		diags.Append(d...)
		if diags.HasError() {
			return
		}
	}

	var tfStateObj types.Object
	d = reqState.GetAttribute(ctx, path.Root("jump_item_association"), &tfStateObj)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	stateIsGone := tfStateObj.IsNull() || tfStateObj.IsUnknown()

	apiSub.ID = &accountID
	tflog.Debug(ctx, fmt.Sprintf("🤷🏻‍♂️ Updating Account Jump Associations with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
		"data":           apiSub,
		"planIsNull":     tfObj.IsNull(),
		"planIsUnknown":  tfObj.IsUnknown(),
		"stateIsNull":    tfStateObj.IsNull(),
		"stateIsUnknown": tfStateObj.IsUnknown(),
	})

	if planIsGone && stateIsGone {
		return
	}

	var item *api.AccountJumpItemAssociation
	var err error
	if !stateIsGone && planIsGone {
		tflog.Trace(ctx, fmt.Sprintf("🦠 Deleting item %+v", apiSub))
		err = api.DeleteItemEndpoint[api.AccountJumpItemAssociation](client, apiSub.Endpoint())
	} else if stateIsGone {
		tflog.Trace(ctx, fmt.Sprintf("🦠 Creating item %+v", apiSub))
		item, err = api.CreateItem(client, apiSub)
	} else {
		tflog.Trace(ctx, fmt.Sprintf("🦠 Updating item %+v", apiSub))
		item, err = api.UpdateItemEndpoint(client, apiSub, apiSub.Endpoint())
	}

	if err != nil {
		diags.AddError(
			"Error Updating Account Jump Item Association",
			"Unexpected value for ID ["+strconv.Itoa(accountID)+"]: "+err.Error(),
		)
		return
	}

	if item != nil {
		tflog.Trace(ctx, fmt.Sprintf("🦠 Setting item in plan %+v", item))
		rb, _ := json.Marshal(item)
		tflog.Trace(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})
		d = respState.SetAttribute(ctx, path.Root("jump_item_association"), item)
	} else {
		var empty api.AccountJumpItemAssociation
		tflog.Trace(ctx, fmt.Sprintf("🦠 Setting empty item in plan %+v", empty))
		d = respState.SetAttribute(ctx, path.Root("jump_item_association"), empty)
	}
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}
