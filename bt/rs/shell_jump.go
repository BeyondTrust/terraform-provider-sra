package rs

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &shellJumpResource{}
	_ resource.ResourceWithConfigure   = &shellJumpResource{}
	_ resource.ResourceWithImportState = &shellJumpResource{}
	_ resource.ResourceWithModifyPlan  = &shellJumpResource{}
)

// Factory function to generate a new resource type. This must be in the
// main list of resource functions in api_resource.go
func newShellJumpResource() resource.Resource {
	return &shellJumpResource{}
}

// The main type for the resource. By convention this should be in the form
//     <resourceName>Resource
// This type name of the API model will be used to generate the public name
// for this resource that is used in the *.tf files. The
// public name will be converted like:
//     ResourceName -> bt_resource_name
type shellJumpResource struct {
	// Compose with the main apiResource struct to get all the boilerplate
	// implementations. Types are: this resource, api model, terraform model
	apiResource[api.ShellJump, models.ShellJumpModel]
}

// We must define the schema for each resource individually. Anything that can be supplied by the API response
// needs to be marked as "Computed", even if we translate from "null" on a POST to an empty string
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

// In order for Terraform to work as expected, the "plan" supplied by the user to the API must match the result. This
// function gives us an opportunity to modify the plan and tell Terraform any default values. Unfortunately, it seems
// like this must be done before sending to the appliance. If terraform complains that the plan isn't stable,
// then you need to do something here.
func (r *shellJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Info(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Info(ctx, "No plan to modify")
		return
	}
	var plan models.ShellJumpModel
	diags := req.Plan.Get(ctx, &plan)
	tflog.Info(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Info(ctx, "Error reading plan")
		return
	}
	/*
	Here we are setting some things that get defaults if they are not supplied.
	*/
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

	// Here we convert null values to the empty default representations for the
	// fields where that is ultimately what will happen. Effectively these
	// are fields that the user doesn't need to supply but won't be null
	// in responses from the API.
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
