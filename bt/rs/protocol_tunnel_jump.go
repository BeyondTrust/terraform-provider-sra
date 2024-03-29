package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &protocolTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &protocolTunnelJumpResource{}
	_ resource.ResourceWithImportState = &protocolTunnelJumpResource{}
	_ resource.ResourceWithModifyPlan  = &protocolTunnelJumpResource{}
)

func newProtocolTunnelJumpResource() resource.Resource {
	return &protocolTunnelJumpResource{}
}

type protocolTunnelJumpResource struct {
	apiResource[api.ProtocolTunnelJump, models.ProtocolTunnelJump]
}

func (r *protocolTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Protocol Tunnel Jump Item.

NOTE: Protocol Tunnel Jumps are PRA only.

For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance`,
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
			"jump_group_id": schema.Int64Attribute{
				Required: true,
			},
			"jump_group_type": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("shared"),
				Validators: jumpGroupTypeValidator(),
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
			"jump_policy_id": schema.Int64Attribute{
				Optional: true,
			},
			"session_policy_id": schema.Int64Attribute{
				Optional: true,
			},
			"tunnel_listen_address": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"tunnel_definitions": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"tunnel_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("tcp"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"tcp", "mssql"}...),
				},
			},
			"username": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"database": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"url": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"ca_certificates": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
		},
	}
}

func (r *protocolTunnelJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}
	var plan models.ProtocolTunnelJump
	diags := req.Plan.Get(ctx, &plan)
	tflog.Debug(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Error reading plan")
		return
	}

	if plan.TunnelDefinitions.IsNull() && plan.TunnelType.ValueString() == "tcp" {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("TunnelDefinitions is required", "You must supply TunnelDefinitions when TunnelType is \"tcp\"."))
		return
	}
	if plan.Username.IsNull() && plan.TunnelType.ValueString() == "mssql" {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Username is required", "You must supply a Username when TunnelType is \"mssql\"."))
		return
	}

	if plan.TunnelType.ValueString() == "tcp" && plan.TunnelListenAddress.IsNull() {
		plan.TunnelListenAddress = types.StringValue("127.0.0.1")
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification")
}
