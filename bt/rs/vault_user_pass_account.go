package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	_ resource.Resource                = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithConfigure   = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithImportState = &vaultUsernamePasswordAccountResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultUsernamePasswordAccountResource{}
)

func newVaultUsernamePasswordAccountResource() resource.Resource {
	return &vaultUsernamePasswordAccountResource{}
}

type vaultUsernamePasswordAccountResource struct {
	apiResource[api.VaultUsernamePasswordAccount, models.VaultUsernamePasswordAccount]
}

func (r *vaultUsernamePasswordAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Vault Username/Password Account.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString("username_password"),
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"personal": schema.BoolAttribute{
				Computed: true,
			},
			"owner_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"account_group_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
			},
			"account_policy": schema.StringAttribute{
				Optional: true,
			},
			"username": schema.StringAttribute{
				Required: true,
			},
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"last_checkout_timestamp": schema.StringAttribute{
				Computed: true,
			},

			"jump_item_association": accountJumpItemAssociationSchema(),
		},
	}
}

func (r *vaultUsernamePasswordAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH creating plan")

	var apiSub api.AccountJumpItemAssociation
	var tfObj types.Object
	diags := req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if tfObj.IsNull() {
		return
	}

	diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	req.Plan.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	apiSub.ID = &id
	tflog.Info(ctx, fmt.Sprintf("🙀 Creating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
		"data": apiSub,
	})

	item, err := api.CreateItem(r.ApiClient, apiSub)

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

func (r *vaultUsernamePasswordAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH reading state")

	var apiSub api.AccountJumpItemAssociation
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
		"data":   apiSub,
		"tfNULL": tfObj.IsNull(),
		"tfUNK":  tfObj.IsUnknown(),
	})

	item, err := api.GetItemEndpoint[api.AccountJumpItemAssociation](r.ApiClient, apiSub.Endpoint())

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

func (r *vaultUsernamePasswordAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH updating plan")

	var apiSub api.AccountJumpItemAssociation
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
	tflog.Info(ctx, "🙀 got state object", map[string]interface{}{
		"data": tfStateObj,
	})

	var item *api.AccountJumpItemAssociation
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
