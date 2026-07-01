package rs

import (
	"context"
	"strconv"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Account is a member of",
						},
						"role": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf([]string{"inject", "inject_and_checkout"}...),
							},
						},
					},
				},
			},
		},
	}
}

func (r *vaultUsernamePasswordAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", "Could not parse resource ID: "+err.Error())
		return
	}

	CreateAccountJIA(ctx, r.ApiClient, req.Plan, &resp.State, &resp.Diagnostics, id)
	if resp.Diagnostics.HasError() {
		return
	}

	CreateGPMemberships[api.GroupPolicyVaultAccount](ctx, r.ApiClient, req.Plan, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyVaultAccount, entityID int) { m.AccountID = &entityID },
		func(m *api.GroupPolicyVaultAccount) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyVaultAccount, id *string) { m.GroupPolicyID = id },
		&accountMembershipMutex,
	)
}

func (r *vaultUsernamePasswordAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", "Could not parse resource ID: "+err.Error())
		return
	}

	ReadAccountJIA(ctx, r.ApiClient, req.State, &resp.State, &resp.Diagnostics, id)
	if resp.Diagnostics.HasError() {
		return
	}

	ReadGPMemberships[api.GroupPolicyVaultAccount](ctx, r.ApiClient, req.State, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyVaultAccount) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyVaultAccount, id *string) { m.GroupPolicyID = id },
	)
}

func (r *vaultUsernamePasswordAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	req.Plan.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", "Could not parse resource ID: "+err.Error())
		return
	}

	UpdateAccountJIA(ctx, r.ApiClient, req.Plan, req.State, &resp.State, &resp.Diagnostics, id)
	if resp.Diagnostics.HasError() {
		return
	}

	UpdateGPMemberships[api.GroupPolicyVaultAccount](ctx, r.ApiClient, req.Plan, req.State, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyVaultAccount, entityID int) { m.AccountID = &entityID },
		func(m *api.GroupPolicyVaultAccount) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyVaultAccount, id *string) { m.GroupPolicyID = id },
		api.DiffGPAccountLists,
		&accountMembershipMutex,
	)
}
