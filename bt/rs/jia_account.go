package rs

import (
	"context"
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

	// CreateItem returns (nil, nil) on a 204 No Content: the association WAS
	// created, there is simply no body. Echo the accepted request rather than
	// writing a zero-value association — that would put filter_type: "" into
	// state, a value the attribute's own contract forbids (Required +
	// stringvalidator.OneOf at api_resource.go), and fail the apply with an
	// inconsistent-result error.
	result := createdOrSent(item, apiSub)
	logItem(ctx, "🙀 got item", result)
	d = state.SetAttribute(ctx, path.Root("jump_item_association"), result)
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

	if err != nil {
		if api.IsNotFound(err) {
			// No association: either it was deleted out of band, or the account
			// never had one (this GET 404s in both cases — measured).
			//
			// Write a null object, not a zero-value struct. A zero-value struct
			// serialises to filter_type: "", which is outside the set the
			// attribute declares, and which UpdateAccountJIA's stateIsGone check
			// (IsNull || IsUnknown) does not recognise as absence. Null is how the
			// create path already represents this, one function above.
			d = respState.SetAttribute(ctx, path.Root("jump_item_association"),
				types.ObjectNull(tfObj.AttributeTypes(ctx)))
			diags.Append(d...)
			return
		}
		if planIsGone || apiSub.FilterType == "" {
			// Documented tolerance, not an oversight: per the spec, this GET
			// "cannot be used if the Account or Secret is inheriting Jump Item
			// association criteria from its Account Group"
			// (openapi/bt-pra-configuration.openapi.yaml:4596-4610), and the
			// spec does not say what it returns in that case. planIsGone is
			// exactly the inherit case. FilterType == "" additionally covers
			// state already corrupted by the pre-fix bug (which wrote a
			// zero-value association on any error); tolerating it here lets
			// that state heal on the next successful read instead of hard-
			// failing every refresh for the users it already hurt.
			tflog.Debug(ctx, "🙀 Tolerating error reading account jump item association", map[string]interface{}{
				"planIsGone": planIsGone,
				"error":      err.Error(),
			})
			return
		}
		diags.AddError(
			"Error reading item",
			"Unexpected reading item ID ["+strconv.Itoa(accountID)+"]: "+err.Error(),
		)
		return
	}

	logItem(ctx, "🙀 got item", item)
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
	// Unknown counts as gone, and that is load-bearing rather than loose.
	// jump_item_association is Optional + Computed, so for a null config
	// Terraform copies the prior state value into the proposed new state and
	// then marks it unknown — which makes this unknown the only signal the
	// provider gets that the block was removed. Drop the IsUnknown() and the
	// unknown falls through to the As() below, where UnhandledUnknownAsEmpty
	// yields a zero struct and the provider PATCHes filter_type: "".
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
		logItem(ctx, "🦠 Deleting item", apiSub)
		err = api.DeleteItemEndpoint[api.AccountJumpItemAssociation](client, apiSub.Endpoint())
	} else if stateIsGone {
		logItem(ctx, "🦠 Creating item", apiSub)
		item, err = api.CreateItem(client, apiSub)
		// CreateItem returns (nil, nil) on a 204 No Content: the association
		// WAS created, there is simply no body. Resolve it to the echoed
		// request here so the item != nil branch below always has a populated
		// association to write — writing a zero value would put
		// filter_type: "" into state, which the attribute's own contract
		// forbids (Required + stringvalidator.OneOf, api_resource.go).
		if err == nil {
			resolved := createdOrSent(item, apiSub)
			item = &resolved
		}
	} else {
		logItem(ctx, "🦠 Updating item", apiSub)
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
		logItem(ctx, "🦠 Setting item in plan", item)
		logItem(ctx, "🙀 got item", item)
		d = respState.SetAttribute(ctx, path.Root("jump_item_association"), item)
	} else {
		// item is nil here only via the delete branch above (!stateIsGone &&
		// planIsGone) — the create branch now always resolves item to a
		// non-nil value before reaching this point. Do not "unify" the two:
		// the association was just deleted from the appliance, so state must
		// reflect its absence, not echo the request back into existence.
		//
		// Absence is a null object. A zero-value struct is not absence: it
		// serialises to filter_type: "", and the stateIsGone check above
		// (IsNull || IsUnknown) then reads it as "still present", so the next
		// apply re-enters this delete branch and DELETEs an association that is
		// already gone — a hard error on every subsequent apply.
		d = respState.SetAttribute(ctx, path.Root("jump_item_association"),
			types.ObjectNull(tfObj.AttributeTypes(ctx)))
	}
	diags.Append(d...)
	if diags.HasError() {
		return
	}
}
