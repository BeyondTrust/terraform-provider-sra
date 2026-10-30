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
	_ resource.Resource                = &oracleTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &oracleTunnelJumpResource{}
	_ resource.ResourceWithImportState = &oracleTunnelJumpResource{}
	_ resource.ResourceWithModifyPlan  = &oracleTunnelJumpResource{}
)

func newOracleTunnelJumpResource() resource.Resource { return &oracleTunnelJumpResource{} }

type oracleTunnelJumpResource struct {
	apiResource[api.OracleTunnelJump, models.OracleTunnelJump]
}

func (r *oracleTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Oracle Tunnel Jump Item.\n\nNOTE: PRA only. Requires PRA 26.3.1 or later.\n\n" + tunnelJumpNotAssociableNote,
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{Required: true},
			"jumpoint_id": schema.Int64Attribute{
				Description: "Sent to the API as `gateway_id`; API errors use that name.",
				Required:    true,
			},
			"hostname": schema.StringAttribute{Required: true},
			"jump_group_id": schema.Int64Attribute{
				Description: "Sent to the API as `asset_group_id`; API errors use that name.",
				Required:    true,
			},
			"jump_group_type": schema.StringAttribute{
				Description: "`shared` or `personal`. Sent to the API as `asset_group_type`; API errors use that name.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("shared"),
				Validators:  jumpGroupTypeValidator(),
			},
			"tag":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"comments":          schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"jump_policy_id":    schema.Int64Attribute{Optional: true},
			"session_policy_id": schema.Int64Attribute{Optional: true},
			"tunnel_listen_address": schema.StringAttribute{
				Description: tunnelListenAddressDescription + tunnelListenAddressLocalAddressNote,
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{tunnelListenAddressValidator{}},
			},
			"username": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"database": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
		},
	}
}

// ModifyPlan plans an omitted tunnel_listen_address as the value the appliance
// holds (127.0.0.1 on create); see planTunnelListenAddress.
func (r *oracleTunnelJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config, state models.OracleTunnelJump
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.TunnelListenAddress = planTunnelListenAddress(config.TunnelListenAddress, state.TunnelListenAddress)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}
