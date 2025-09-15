package rs

import (
	"context"
	"net"
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
	_ resource.Resource                = &postgresqlTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &postgresqlTunnelJumpResource{}
	_ resource.ResourceWithImportState = &postgresqlTunnelJumpResource{}
)

func newPostgreSQLTunnelJumpResource() resource.Resource { return &postgresqlTunnelJumpResource{} }

type postgresqlTunnelJumpResource struct {
	apiResource[api.PostgreSQLTunnelJump, models.PostgreSQLTunnelJump]
}

func (r *postgresqlTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PostgreSQL Tunnel Jump Item. NOTE: PRA only.",
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

func (r *postgresqlTunnelJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification (postgresql)")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}
	var plan models.PostgreSQLTunnelJump
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// apply validation/defaulting logic
	diags = applyPostgresDefaultsAndValidate(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification (postgresql)")
}

func (r *postgresqlTunnelJumpResource) printableName() string {
	return "postgresql_tunnel_jump"
}

// applyPostgresDefaultsAndValidate applies defaults and returns diagnostics similar to
// the logic used during ModifyPlan. This is separated to make unit testing easier.
func applyPostgresDefaultsAndValidate(plan *models.PostgreSQLTunnelJump) diag.Diagnostics {
	var diags diag.Diagnostics
	// name: required, length 1..128
	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		diags.Append(diag.NewErrorDiagnostic("name is required", "The name field is required and must be 1..128 characters"))
		return diags
	}
	if l := len(plan.Name.ValueString()); l < 1 || l > 128 {
		diags.Append(diag.NewErrorDiagnostic("name length", "The name must be between 1 and 128 characters"))
	}

	// hostname length
	if !plan.Hostname.IsNull() {
		if l := len(plan.Hostname.ValueString()); l < 1 || l > 128 {
			diags.Append(diag.NewErrorDiagnostic("hostname length", "The hostname must be between 1 and 128 characters"))
		}
	}

	// jumpoint_id and jump_group_id must be >= 1
	// jumpoint_id and jump_group_id must be >= 1 when known and negative values are invalid.
	// Accept 0 during planning because module outputs or unknown values can appear as 0 in the
	// plan; the real server-side validation will run on apply.
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

	// tunnel_listen_address default and subnet constraint (127.0.0.0/24)
	if plan.TunnelListenAddress.IsNull() || plan.TunnelListenAddress.ValueString() == "" {
		plan.TunnelListenAddress = types.StringValue("127.0.0.1")
	} else {
		addr := plan.TunnelListenAddress.ValueString()
		if len(addr) > 32 {
			diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address length", "tunnel_listen_address must be at most 32 characters"))
		}
		// precise subnet check: must be valid IP and within 127.0.0.0/24
		ip := net.ParseIP(addr)
		if ip == nil {
			diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address invalid", "tunnel_listen_address must be a valid IP address"))
		} else {
			_, cidr, _ := net.ParseCIDR("127.0.0.0/24")
			if !cidr.Contains(ip) {
				diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address subnet", "tunnel_listen_address must be within the 127.0.0.0/24 subnet"))
			}
		}
	}
	return diags
}
