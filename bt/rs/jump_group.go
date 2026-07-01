package rs

import (
	"context"
	"strconv"
	"sync"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &jumpGroupResource{}
	_ resource.ResourceWithConfigure   = &jumpGroupResource{}
	_ resource.ResourceWithImportState = &jumpGroupResource{}
	_ resource.ResourceWithModifyPlan  = &jumpGroupResource{}

	// Because of the way the PHP code handles changing memberships, those
	// operations cannot be done in parallel. We use this mutex to ensure
	// we deal with membership updates one at a time
	jgMembershipMutex sync.Mutex
)

func newJumpGroupResource() resource.Resource {
	return &jumpGroupResource{}
}

type jumpGroupResource struct {
	apiResource[api.JumpGroup, models.JumpGroup]
}

func (r *jumpGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Default:  stringdefault.StaticString(""),
			},

			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Account is a member of",
						},
						"jump_item_role_id": schema.Int64Attribute{
							Description: `The ID of the Jump Item Role that applies to this membership. Omitting or 0 means "User's Default"`,
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(0),
						},
						"jump_policy_id": schema.Int64Attribute{
							Description: `The ID of the Jump Policy that applies to this membership. Omitting or 0 means "Set on Jump Items"

This field only applies to PRA`,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (r *jumpGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}

	/*
		Here we are setting some things that get defaults if they are not supplied.
	*/
	var tfGPList types.Set
	diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !tfGPList.IsNull() {
		var planList []models.GroupPolicyJumpGroup
		diags = tfGPList.ElementsAs(ctx, &planList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range planList {
			if r.ApiClient.IsPRA() {
				if m.JumpPolicyID.IsNull() || m.JumpPolicyID.IsUnknown() {
					m.JumpPolicyID = types.Int64Value(0)
				}
			} else {
				m.JumpPolicyID = types.Int64Null()
			}
			planList[i] = m
		}

		diags = resp.Plan.SetAttribute(ctx, path.Root("group_policy_memberships"), planList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Debug(ctx, "Finished modification")
}

func (r *jumpGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", "Could not parse jump group ID: "+err.Error())
		return
	}

	CreateGPMemberships[api.GroupPolicyJumpGroup](ctx, r.ApiClient, req.Plan, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyJumpGroup, entityID int) { m.JumpGroupID = &entityID },
		func(m *api.GroupPolicyJumpGroup) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpGroup, gpID *string) { m.GroupPolicyID = gpID },
		&jgMembershipMutex,
	)
}

func (r *jumpGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", "Could not parse jump group ID: "+err.Error())
		return
	}

	ReadGPMemberships[api.GroupPolicyJumpGroup](ctx, r.ApiClient, req.State, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyJumpGroup) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpGroup, gpID *string) { m.GroupPolicyID = gpID },
	)
}

func (r *jumpGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", "Could not parse jump group ID: "+err.Error())
		return
	}

	UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, r.ApiClient, req.Plan, req.State, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyJumpGroup, entityID int) { m.JumpGroupID = &entityID },
		func(m *api.GroupPolicyJumpGroup) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpGroup, gpID *string) { m.GroupPolicyID = gpID },
		api.DiffGPJumpItemLists,
		&jgMembershipMutex,
	)
}
