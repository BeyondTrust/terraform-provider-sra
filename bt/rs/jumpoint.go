package rs

import (
	"context"
	"strconv"
	"sync"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &jumpointResource{}
	_ resource.ResourceWithConfigure   = &jumpointResource{}
	_ resource.ResourceWithImportState = &jumpointResource{}
	_ resource.ResourceWithModifyPlan  = &jumpointResource{}

	// Because of the way the PHP code handles changing memberships, those
	// operations cannot be done in parallel. We use this mutex to ensure
	// we deal with membership updates one at a time
	jpMembershipMutex sync.Mutex
)

func newJumpointResource() resource.Resource {
	return &jumpointResource{}
}

type jumpointResource struct {
	apiResource[api.Jumpoint, models.Jumpoint]
}

func (r *jumpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Jump Group.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
			"code_name": schema.StringAttribute{
				Required: true,
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"platform": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"windows-x86", "linux-x86"}...),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"connected": schema.BoolAttribute{
				Computed: true,
			},
			"clustered": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"shell_jump_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"external_jump_item_network_id": schema.StringAttribute{
				Optional: true,
			},
			"protocol_tunnel_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"rdp_service_account_id": schema.Int64Attribute{
				Optional:    true,
				Description: "This field only applies to PRA",
			},

			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Jumpoint is a member of",
						},
					},
				},
			},
		},
	}
}

func (r *jumpointResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}
	var plan models.Jumpoint
	diags := req.Plan.Get(ctx, &plan)
	tflog.Debug(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Error reading plan")
		return
	}

	if r.ApiClient.IsRS() {
		plan.ProtocolTunnelEnabled = types.BoolNull()
	} else if r.ApiClient.IsPRA() && plan.ProtocolTunnelEnabled.IsUnknown() {
		plan.ProtocolTunnelEnabled = types.BoolValue(true)
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification")
}

func (r *jumpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", "Could not parse jumpoint ID: "+err.Error())
		return
	}

	CreateGPMemberships[api.GroupPolicyJumpoint](ctx, r.ApiClient, req.Plan, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyJumpoint, entityID int) { m.JumpointID = &entityID },
		func(m *api.GroupPolicyJumpoint) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpoint, gpID *string) { m.GroupPolicyID = gpID },
		&jpMembershipMutex,
	)
}

func (r *jumpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	// If the generic Read removed the resource from state (deleted out-of-band),
	// stop — there is nothing left to refresh memberships for.
	if resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
		return
	}

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", "Could not parse jumpoint ID: "+err.Error())
		return
	}

	ReadGPMemberships[api.GroupPolicyJumpoint](ctx, r.ApiClient, req.State, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyJumpoint) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpoint, gpID *string) { m.GroupPolicyID = gpID },
	)
}

func (r *jumpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", "Could not parse jumpoint ID: "+err.Error())
		return
	}

	UpdateGPMemberships[api.GroupPolicyJumpoint](ctx, r.ApiClient, req.Plan, req.State, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyJumpoint, entityID int) { m.JumpointID = &entityID },
		func(m *api.GroupPolicyJumpoint) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpoint, gpID *string) { m.GroupPolicyID = gpID },
		api.DiffGPJumpointLists,
		&jpMembershipMutex,
	)
}
