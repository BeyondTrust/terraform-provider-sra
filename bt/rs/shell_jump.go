package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
//
//	<resourceName>Resource
//
// This type name of the API model will be used to generate the public name
// for this resource that is used in the *.tf files. The
// public name will be converted like:
//
//	ResourceName -> sra_resource_name
type shellJumpResource struct {
	// Compose with the main apiResource struct to get all the boilerplate
	// implementations. Types are: this resource, api model, terraform model
	apiResource[api.ShellJump, models.ShellJump]
}

// We must define the schema for each resource individually. Anything that can be supplied by the API response
// needs to be marked as "Computed", even if we translate from "null" on a POST to an empty string
func (r *shellJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Shell Jump Item.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("ssh"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"ssh", "telnet"}...),
				},
			},
			"port": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{},
			},
			"jump_group_id": schema.Int64Attribute{
				Required: true,
			},
			"jump_group_type": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("shared"),
				Validators: jumpGroupTypeValidator(),
			},
			"terminal": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("xterm"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"xterm", "VT100"}...),
				},
			},
			"keep_alive": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
			"tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
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
// then you need to do something here. Defaults that don't need any logic can be specified in the schema.
func (r *shellJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}
	var plan models.ShellJump
	diags := req.Plan.Get(ctx, &plan)
	tflog.Debug(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Error reading plan")
		return
	}
	/*
		Here we are setting some things that get defaults if they are not supplied.
	*/
	if plan.Port.ValueInt64() == 0 {
		if plan.Protocol.ValueString() != "ssh" {
			tflog.Debug(ctx, "plan.Port was null, setting default as 23")
			plan.Port = types.Int64Value(23)
		} else {
			tflog.Debug(ctx, "plan.Port was null, setting default as 22")
			plan.Port = types.Int64Value(22)
		}
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification")
}
