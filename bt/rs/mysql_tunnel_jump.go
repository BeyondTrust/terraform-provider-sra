package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &mysqlTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &mysqlTunnelJumpResource{}
	_ resource.ResourceWithImportState = &mysqlTunnelJumpResource{}
	_ resource.ResourceWithModifyPlan  = &mysqlTunnelJumpResource{}
)

func newMySQLTunnelJumpResource() resource.Resource { return &mysqlTunnelJumpResource{} }

type mysqlTunnelJumpResource struct {
	apiResource[api.MySQLTunnelJump, models.MySQLTunnelJump]
}

// ModifyPlan validates the planned name, hostname and ids.
func (r *mysqlTunnelJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan models.MySQLTunnelJump
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateSQLTunnelPlan(plan.Name, plan.Hostname, plan.JumpointID, plan.JumpGroupID)...)
}

func (r *mysqlTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a MySQL Tunnel Jump Item. NOTE: PRA only.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":              schema.StringAttribute{Required: true},
			"jumpoint_id":       schema.Int64Attribute{Required: true},
			"hostname":          schema.StringAttribute{Required: true},
			"jump_group_id":     schema.Int64Attribute{Required: true},
			"jump_group_type":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("shared")},
			"tag":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"comments":          schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"jump_policy_id":    schema.Int64Attribute{Optional: true},
			"session_policy_id": schema.Int64Attribute{Optional: true},
			"tunnel_listen_address": schema.StringAttribute{
				Description:   tunnelListenAddressDescription,
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{tunnelListenAddressValidator{allowEmpty: true}},
				PlanModifiers: []planmodifier.String{tunnelListenAddressPlan{}},
			},
			"username": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"database": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
		},
	}
}
