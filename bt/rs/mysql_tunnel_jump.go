package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &mysqlTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &mysqlTunnelJumpResource{}
	_ resource.ResourceWithImportState = &mysqlTunnelJumpResource{}
)

func newMySQLTunnelJumpResource() resource.Resource { return &mysqlTunnelJumpResource{} }

type mysqlTunnelJumpResource struct {
	apiResource[api.MySQLTunnelJump, models.MySQLTunnelJump]
}

func (r *mysqlTunnelJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification (mysql)")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}
	var plan models.MySQLTunnelJump
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.TunnelListenAddress.IsNull() {
		plan.TunnelListenAddress = types.StringValue("127.0.0.1")
	}

	diags = applyMySQLDefaultsAndValidate(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification (mysql)")
}

func applyMySQLDefaultsAndValidate(plan *models.MySQLTunnelJump) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		diags.Append(diag.NewErrorDiagnostic("name is required", "The name field is required and must be 1..128 characters"))
		return diags
	}
	if l := len(plan.Name.ValueString()); l < 1 || l > 128 {
		diags.Append(diag.NewErrorDiagnostic("name length", "The name must be between 1 and 128 characters"))
	}

	if !plan.Hostname.IsNull() {
		if l := len(plan.Hostname.ValueString()); l < 1 || l > 128 {
			diags.Append(diag.NewErrorDiagnostic("hostname length", "The hostname must be between 1 and 128 characters"))
		}
	}

	// jumpoint_id and jump_group_id must be >= 1 when known. If the value is unknown
	// during planning (for example when coming from a module output), skip numeric
	// validation so Terraform can complete the plan phase.
	if !plan.JumpointID.IsNull() && !plan.JumpointID.IsUnknown() {
		if plan.JumpointID.ValueInt64() < 0 {
			diags.Append(diag.NewErrorDiagnostic("jumpoint_id invalid", "jumpoint_id must be >= 0"))
		}
	}
	if !plan.JumpGroupID.IsNull() && !plan.JumpGroupID.IsUnknown() {
		if plan.JumpGroupID.ValueInt64() < 0 {
			diags.Append(diag.NewErrorDiagnostic("jump_group_id invalid", "jump_group_id must be >= 0"))
		}
	}

	if plan.TunnelListenAddress.IsNull() || plan.TunnelListenAddress.ValueString() == "" {
		plan.TunnelListenAddress = types.StringValue("127.0.0.1")
	} else {
		addr := plan.TunnelListenAddress.ValueString()
		if len(addr) > 32 {
			diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address length", "tunnel_listen_address must be at most 32 characters"))
		}
		if !(len(addr) >= 4 && addr[:4] == "127.") {
			diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address subnet", "tunnel_listen_address must be in the 127.0.0.0/24 subnet"))
		}
	}
	return diags
}

func (r *mysqlTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a MySQL Tunnel Jump Item. NOTE: PRA only.",
		Attributes: map[string]schema.Attribute{
			"id":                    schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":                  schema.StringAttribute{Required: true},
			"jumpoint_id":           schema.Int64Attribute{Required: true},
			"hostname":              schema.StringAttribute{Required: true},
			"jump_group_id":         schema.Int64Attribute{Required: true},
			"jump_group_type":       schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("shared")},
			"tag":                   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"comments":              schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"jump_policy_id":        schema.Int64Attribute{Optional: true},
			"session_policy_id":     schema.Int64Attribute{Optional: true},
			"tunnel_listen_address": schema.StringAttribute{Optional: true, Computed: true},
			"username":              schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"database":              schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
		},
	}
}
