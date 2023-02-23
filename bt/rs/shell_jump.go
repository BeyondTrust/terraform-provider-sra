package rs

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	. "terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &shellJumpResource{}
	_ resource.ResourceWithConfigure   = &shellJumpResource{}
	_ resource.ResourceWithImportState = &shellJumpResource{}
	_ resource.ResourceWithModifyPlan  = &shellJumpResource{}
)

func newShellJumpResource() resource.Resource {
	return &shellJumpResource{}
}

type shellJumpResource struct {
	apiResource[shellJumpResource, api.ShellJump, ShellJumpModel]
}

func (r *shellJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"jumpoint_id": schema.Int64Attribute{
				Required: true,
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"protocol": schema.StringAttribute{
				Required: true,
			},
			"port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"jump_group_id": schema.Int64Attribute{
				Required: true,
			},
			"jump_group_type": schema.StringAttribute{
				Required: true,
			},
			"terminal": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"keep_alive": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"jump_policy_id": schema.Int64Attribute{
				Optional: true,
			},
			"session_policy_id": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}

func (r *shellJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Info(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Info(ctx, "No plan to modify")
		return
	}
	var plan ShellJumpModel
	diags := req.Plan.Get(ctx, &plan)
	tflog.Info(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Info(ctx, "Error reading plan")
		return
	}
	if plan.Terminal.ValueString() == "" {
		tflog.Info(ctx, "plan.Terminal was null, setting default")
		plan.Terminal = types.StringValue("xterm")
	}
	if plan.Port.ValueInt64() == 0 {
		if plan.Protocol.ValueString() != "ssh" {
			tflog.Info(ctx, "plan.Port was null, setting default as 23")
			plan.Port = types.Int64Value(23)
		} else {
			tflog.Info(ctx, "plan.Port was null, setting default as 22")
			plan.Port = types.Int64Value(22)
		}
	}

	// Just convert null values to the empty default representations
	plan.Username = types.StringValue(plan.Username.ValueString())
	plan.KeepAlive = types.Int64Value(plan.KeepAlive.ValueInt64())
	plan.Tag = types.StringValue(plan.Tag.ValueString())
	plan.Comments = types.StringValue(plan.Comments.ValueString())

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Finished modification")
}
