package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &vaultAccountGroupResource{}
	_ resource.ResourceWithConfigure   = &vaultAccountGroupResource{}
	_ resource.ResourceWithImportState = &vaultAccountGroupResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultAccountGroupResource{}
)

func newVaultAccountGroupResource() resource.Resource {
	return &vaultAccountGroupResource{}
}

type vaultAccountGroupResource struct {
	apiResource[api.VaultAccountGroup, models.VaultAccountGroup]
}

func (r *vaultAccountGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	jiaSchema := accountJumpItemAssociationSchema()

	var tfDefault types.Object
	tfDefault, _ = types.ObjectValue(
		map[string]attr.Type{"filter_type": types.StringType, "criteria": types.ObjectType{}},
		map[string]attr.Value{"filter_type": types.StringValue("any_jump_items"), "criteria": types.ObjectNull(map[string]attr.Type{})},
	)

	jiaSchema.Default = objectdefault.StaticValue(tfDefault)

	resp.Schema = schema.Schema{
		Description: "Manages a Vault Account Group.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"account_policy": schema.StringAttribute{
				Optional: true,
			},

			"jump_item_association": jiaSchema,
		},
	}
}

func (r *vaultAccountGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH reading state")

	var apiSub api.AccountGroupJumpItemAssociation
	var tfObj types.Object
	diags := req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !tfObj.IsNull() {
		diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	apiSub.ID = &id
	tflog.Info(ctx, fmt.Sprintf("🙀 Reading API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
		"data": apiSub,
	})

	item, err := api.GetItemEndpoint[api.AccountGroupJumpItemAssociation](r.ApiClient, apiSub.Endpoint())

	if item == nil && tfObj.IsNull() {
		return
	}

	rb, _ := json.Marshal(item)
	tflog.Info(ctx, "🙀 got item", map[string]interface{}{
		"data": string(rb),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading item",
			"Unexpected reading item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
		)
		return
	}
	diags = req.State.SetAttribute(ctx, path.Root("jump_item_association"), item)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vaultAccountGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH updating plan")

	var apiSub api.AccountGroupJumpItemAssociation
	var tfObj types.Object
	diags := req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !tfObj.IsNull() {
		diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var tfId types.String
	req.Plan.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	apiSub.ID = &id
	tflog.Info(ctx, fmt.Sprintf("🙀 Updating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
		"data": apiSub,
	})

	var tfStateObj types.Object
	diags = req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfStateObj)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var item *api.AccountGroupJumpItemAssociation
	var err error
	if tfStateObj.IsNull() {
		item, err = api.CreateItem(r.ApiClient, apiSub)
	} else {
		item, err = api.UpdateItemEndpoint(r.ApiClient, apiSub, apiSub.Endpoint())
	}

	rb, _ := json.Marshal(item)
	tflog.Info(ctx, "🙀 got item", map[string]interface{}{
		"data": string(rb),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading item",
			"Unexpected creating item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
		)
		return
	}
	diags = req.Plan.SetAttribute(ctx, path.Root("jump_item_association"), item)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
