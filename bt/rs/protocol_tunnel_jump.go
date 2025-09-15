package rs

import (
	"context"
	"net"
	"strconv"
	"strings"
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

	// Apply more thorough validation and defaults
	diags = applyProtocolTunnelDefaultsAndValidate(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification")
}

// applyProtocolTunnelDefaultsAndValidate enforces OpenAPI constraints for ProtocolTunnelJump
func applyProtocolTunnelDefaultsAndValidate(plan *models.ProtocolTunnelJump) diag.Diagnostics {
	var diags diag.Diagnostics

	// name required
	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		diags.Append(diag.NewErrorDiagnostic("name is required", "The name field is required and must be 1..128 characters"))
		return diags
	}

	ttype := plan.TunnelType.ValueString()

	// TCP-specific: tunnel_definitions required and must be even number of semicolon-separated ints
	if ttype == "tcp" {
		if plan.TunnelDefinitions.IsNull() || plan.TunnelDefinitions.ValueString() == "" {
			diags.Append(diag.NewErrorDiagnostic("TunnelDefinitions is required", "You must supply TunnelDefinitions when TunnelType is \"tcp\"."))
			return diags
		}
		parts := strings.Split(plan.TunnelDefinitions.ValueString(), ";")
		if len(parts)%2 != 0 {
			diags.Append(diag.NewErrorDiagnostic("TunnelDefinitions invalid", "TunnelDefinitions must contain pairs of local and remote ports separated by ';'"))
			return diags
		}
		for i, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				diags.Append(diag.NewErrorDiagnostic("TunnelDefinitions invalid", "Empty port value in TunnelDefinitions"))
				continue
			}
			v, err := strconv.Atoi(p)
			if err != nil {
				diags.Append(diag.NewErrorDiagnostic("TunnelDefinitions invalid", "All tunnel definition values must be integers"))
				continue
			}
			if i%2 == 0 { // local port: 0..65535
				if v < 0 || v > 65535 {
					diags.Append(diag.NewErrorDiagnostic("TunnelDefinitions port out of range", "Local ports must be between 0 and 65535"))
				}
			} else { // remote port: 1..65535
				if v < 1 || v > 65535 {
					diags.Append(diag.NewErrorDiagnostic("TunnelDefinitions port out of range", "Remote ports must be between 1 and 65535"))
				}
			}
		}
	}

	// mssql-specific: username required
	if ttype == "mssql" {
		if plan.Username.IsNull() || plan.Username.ValueString() == "" {
			diags.Append(diag.NewErrorDiagnostic("Username is required", "You must supply a Username when TunnelType is \"mssql\"."))
		}
	}

	// k8s-specific: url and ca_certificates required
	if ttype == "k8s" {
		if plan.URL.IsNull() || plan.URL.ValueString() == "" {
			diags.Append(diag.NewErrorDiagnostic("url is required", "You must supply a url when TunnelType is \"k8s\"."))
		}
		if plan.CACertificates.IsNull() || plan.CACertificates.ValueString() == "" {
			diags.Append(diag.NewErrorDiagnostic("ca_certificates is required", "You must supply ca_certificates when TunnelType is \"k8s\"."))
		}
	}

	// tunnel_listen_address: only allowed for tcp. Default/validate for tcp, clear for others.
	if ttype == "tcp" {
		if plan.TunnelListenAddress.IsNull() || plan.TunnelListenAddress.ValueString() == "" {
			plan.TunnelListenAddress = types.StringValue("127.0.0.1")
		} else {
			ip := net.ParseIP(plan.TunnelListenAddress.ValueString())
			if ip == nil {
				diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address invalid", "tunnel_listen_address must be a valid IP address"))
			} else {
				_, cidr, _ := net.ParseCIDR("127.0.0.0/24")
				if !cidr.Contains(ip) {
					diags.Append(diag.NewErrorDiagnostic("tunnel_listen_address subnet", "tunnel_listen_address must be within the 127.0.0.0/24 subnet"))
				}
			}
		}
	} else {
		// Some provider implementations return an empty string for this field
		// when it is not applicable. Set it to empty string to avoid a
		// provider-inconsistent result after apply (null -> "").
		plan.TunnelListenAddress = types.StringValue("")
	}

	return diags
}
