package rs

import (
	"context"
	"strconv"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                     = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithConfigure        = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithConfigValidators = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithImportState      = &vaultUsernamePasswordAccountResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultUsernamePasswordAccountResource{}
)

func newVaultUsernamePasswordAccountResource() resource.Resource {
	r := &vaultUsernamePasswordAccountResource{}
	r.apiResource.prepareCreate = r.prepareCreatePassword
	r.apiResource.prepareUpdate = r.prepareUpdatePassword
	return r
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
				Optional:    true,
				Sensitive:   true,
				Description: "Password stored in Terraform state. Use password_wo for ephemeral credentials.",
			},
			"password_wo": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Write-only password sent to BeyondTrust without being stored in Terraform plan or state.",
			},
			"password_wo_version": schema.Int64Attribute{
				Optional:    true,
				Description: "Version trigger for password_wo. Increment this value to update the password in BeyondTrust.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
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
							Validators:  groupPolicyIDValidators(),
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

func (r *vaultUsernamePasswordAccountResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("password"),
			path.MatchRoot("password_wo"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("password_wo"),
			path.MatchRoot("password_wo_version"),
		),
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
	// If the generic Read removed the resource from state (deleted out-of-band),
	// stop — there is nothing left to refresh associations/memberships for.
	if resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
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

func (r *vaultUsernamePasswordAccountResource) prepareCreatePassword(ctx context.Context, req resource.CreateRequest, item *api.VaultUsernamePasswordAccount, diagnostics *diag.Diagnostics) {
	var config, plan models.VaultUsernamePasswordAccount
	diagnostics.Append(req.Config.Get(ctx, &config)...)
	diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if diagnostics.HasError() {
		return
	}
	if password, ok := vaultAccountPassword(config, plan, true, diagnostics); ok {
		item.Password = password
	}
}

func (r *vaultUsernamePasswordAccountResource) prepareUpdatePassword(ctx context.Context, req resource.UpdateRequest, item *api.VaultUsernamePasswordAccount, diagnostics *diag.Diagnostics) {
	var config, plan, state models.VaultUsernamePasswordAccount
	diagnostics.Append(req.Config.Get(ctx, &config)...)
	diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	diagnostics.Append(req.State.Get(ctx, &state)...)
	if diagnostics.HasError() {
		return
	}
	if password, ok := vaultAccountPassword(config, plan, !plan.PasswordWOVersion.Equal(state.PasswordWOVersion), diagnostics); ok {
		item.Password = password
	}
}

func vaultAccountPassword(config, plan models.VaultUsernamePasswordAccount, useWriteOnly bool, diagnostics *diag.Diagnostics) (string, bool) {
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		return plan.Password.ValueString(), true
	}
	if !useWriteOnly {
		return "", true
	}
	if config.PasswordWO.IsNull() || config.PasswordWO.IsUnknown() {
		diagnostics.AddError(
			"Missing write-only Vault password",
			"password_wo must be known when creating the account or changing password_wo_version.",
		)
		return "", false
	}
	return config.PasswordWO.ValueString(), true
}
