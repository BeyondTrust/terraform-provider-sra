package rs

import (
	"context"
	"strconv"
	"strings"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                   = &protocolTunnelJumpResource{}
	_ resource.ResourceWithConfigure      = &protocolTunnelJumpResource{}
	_ resource.ResourceWithImportState    = &protocolTunnelJumpResource{}
	_ resource.ResourceWithModifyPlan     = &protocolTunnelJumpResource{}
	_ resource.ResourceWithValidateConfig = &protocolTunnelJumpResource{}
)

func newProtocolTunnelJumpResource() resource.Resource {
	return &protocolTunnelJumpResource{}
}

type protocolTunnelJumpResource struct {
	apiResource[api.ProtocolTunnelJump, models.ProtocolTunnelJump]
}

func (r *protocolTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Protocol Tunnel Jump Item of tunnel_type tcp (the default), mssql or k8s.

NOTE: Protocol Tunnel Jumps are PRA only.

hostname is required unless tunnel_type is k8s. For k8s, omit it: the appliance sets it to the first 128 characters of url.
Changing tunnel_type replaces the item, which gives it a new ID.

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
				// Required unless tunnel_type is k8s, where the appliance derives it from
				// url. ValidateConfig enforces the first half, ModifyPlan plans the second.
				Optional: true,
				Computed: true,
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
					// The db tunnel types (mysql, psql, mongodb, oracle) and ip have their own
					// resources and endpoints; this endpoint's index does not filter by type.
					stringvalidator.OneOf([]string{"tcp", "mssql", "k8s"}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						tunnelTypeChangeRequiresReplace,
						"The appliance ignores tunnel_type on update, so changing it replaces the item.",
						"The appliance ignores `tunnel_type` on update, so changing it replaces the item.",
					),
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
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(""),
				Validators: []validator.String{untrimmedValidator{}},
			},
			"ca_certificates": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(""),
				Validators: []validator.String{untrimmedValidator{}},
			},
		},
	}
}

// tunnelTypeChangeRequiresReplace replaces the item only when a known
// tunnel_type changes to another known one. The appliance drops tunnel_type from
// a PATCH, so an in-place change would never land. An unknown plan value is left
// alone rather than planning a destroy on a guess: if it resolves to a different
// type, the apply fails with an inconsistent result instead.
func tunnelTypeChangeRequiresReplace(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = isKnown(req.StateValue) && isKnown(req.PlanValue)
}

// isMissing reports a required value that is absent: null, or known and empty.
// An unknown value is not missing; it is checked again once it is known.
func isMissing(v types.String) bool { return v.IsNull() || (!v.IsUnknown() && v.ValueString() == "") }

// k8sHostname is the hostname the appliance derives from a k8s url: PHP
// substr($url, 0, 128), which counts bytes.
func k8sHostname(url string) string {
	if len(url) > 128 {
		return url[:128]
	}
	return url
}

// ValidateConfig checks the fields each tunnel type requires or forbids. It
// reads configuration only, before defaults, and also runs under terraform
// validate, where variables are unknown. A null tunnel_type is "tcp", the schema
// default; only an unknown tunnel_type skips the type-dependent checks.
func (r *protocolTunnelJumpResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config models.ProtocolTunnelJump
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateProtocolTunnelConfig(config)...)
}

func validateProtocolTunnelConfig(c models.ProtocolTunnelJump) diag.Diagnostics {
	var diags diag.Diagnostics
	required := func(attr string, v types.String, why string) {
		if isMissing(v) {
			diags.AddAttributeError(path.Root(attr), "Missing "+attr, attr+" is required "+why+".")
		}
	}

	required("name", c.Name, "for every protocol tunnel")

	if c.TunnelType.IsUnknown() {
		return diags
	}
	ttype := "tcp"
	if !c.TunnelType.IsNull() {
		ttype = c.TunnelType.ValueString()
	}
	when := `when tunnel_type is "` + ttype + `"`

	switch ttype {
	case "tcp":
		required("tunnel_definitions", c.TunnelDefinitions, when)
		if isSet(c.TunnelDefinitions) {
			diags.Append(validateTunnelDefinitions(c.TunnelDefinitions.ValueString())...)
		}
		if isKnown(c.TunnelListenAddress) {
			if c.TunnelListenAddress.ValueString() == "" {
				diags.AddAttributeError(path.Root("tunnel_listen_address"), "Invalid tunnel_listen_address",
					"tunnel_listen_address can't be empty for a tcp tunnel; omit it to get 127.0.0.1.")
			} else if msg := tunnelListenAddressError(c.TunnelListenAddress.ValueString()); msg != "" {
				diags.AddAttributeError(path.Root("tunnel_listen_address"), "Invalid tunnel_listen_address", "tunnel_listen_address "+msg+".")
			}
		}
		required("hostname", c.Hostname, when)
	case "mssql":
		required("hostname", c.Hostname, when)
		required("username", c.Username, when)
	case "k8s":
		required("url", c.URL, when)
		required("ca_certificates", c.CACertificates, when)
		// Any configured hostname, "" included, must match what the appliance will
		// store, or the apply fails as an inconsistent result.
		if isKnown(c.Hostname) {
			h := c.Hostname.ValueString()
			urlMismatch := isKnown(c.URL) && h != k8sHostname(c.URL.ValueString())
			if h == "" || urlMismatch {
				diags.AddAttributeError(path.Root("hostname"), "Invalid hostname for a k8s tunnel",
					"hostname is derived from url for k8s; omit it.")
			}
		}
	default:
		// The schema's OneOf already reports an unsupported type.
		return diags
	}

	// Fields that belong to another tunnel type. Each of these already failed to
	// apply, with a 400 from the appliance or an inconsistent result. Only a set
	// value is forbidden, so "" passes: v1.4.0 state, generated configuration and
	// the module pattern `cond ? x : ""` all send it.
	forbid := func(attr string, v types.String, owner string) {
		if ttype != owner && isSet(v) {
			diags.AddAttributeError(path.Root(attr), "Unexpected "+attr,
				attr+` only applies when tunnel_type is "`+owner+`"; remove it for "`+ttype+`".`)
		}
	}
	forbid("tunnel_definitions", c.TunnelDefinitions, "tcp")
	forbid("tunnel_listen_address", c.TunnelListenAddress, "tcp")
	forbid("username", c.Username, "mssql")
	forbid("database", c.Database, "mssql")
	forbid("url", c.URL, "k8s")
	forbid("ca_certificates", c.CACertificates, "k8s")

	return diags
}

// validateTunnelDefinitions checks a tcp tunnel_definitions value: pairs of
// local (0-65535) and remote (1-65535) ports separated by ';'.
func validateTunnelDefinitions(defs string) diag.Diagnostics {
	var diags diag.Diagnostics
	at := path.Root("tunnel_definitions")
	parts := strings.Split(defs, ";")
	if len(parts)%2 != 0 {
		diags.AddAttributeError(at, "TunnelDefinitions invalid", "TunnelDefinitions must contain pairs of local and remote ports separated by ';'")
		return diags
	}
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			diags.AddAttributeError(at, "TunnelDefinitions invalid", "Empty port value in TunnelDefinitions")
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			diags.AddAttributeError(at, "TunnelDefinitions invalid", "All tunnel definition values must be integers")
			continue
		}
		if i%2 == 0 { // local port: 0..65535
			if v < 0 || v > 65535 {
				diags.AddAttributeError(at, "TunnelDefinitions port out of range", "Local ports must be between 0 and 65535")
			}
		} else { // remote port: 1..65535
			if v < 1 || v > 65535 {
				diags.AddAttributeError(at, "TunnelDefinitions port out of range", "Remote ports must be between 1 and 65535")
			}
		}
	}
	return diags
}

// ModifyPlan plans the values the appliance will return for the fields the
// configuration leaves out. Validation lives in ValidateConfig. Configuration
// is read to tell an omitted value (null) from one that is not known yet; the
// tunnel type is read from the plan, which already carries the default.
func (r *protocolTunnelJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config, state models.ProtocolTunnelJump
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	planProtocolTunnelDefaults(&plan, config, state)

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// planProtocolTunnelDefaults fills the computed values ModifyPlan plans. On
// create, state is the zero model, which is all null.
func planProtocolTunnelDefaults(plan *models.ProtocolTunnelJump, config, state models.ProtocolTunnelJump) {
	if plan.TunnelType.IsUnknown() {
		return
	}
	ttype := plan.TunnelType.ValueString()
	sameType := state.TunnelType.Equal(plan.TunnelType)

	// An omitted value keeps what the appliance holds for an item of the same
	// type, so an imported or console-set listen address such as 127.0.0.5 does
	// not plan a change. Otherwise it is what the appliance returns for this
	// type: 127.0.0.1 for a tcp listen address, "" for the rest.
	omitted := func(configured, prior types.String, fallback string) types.String {
		if !configured.IsNull() {
			return configured
		}
		if sameType && isKnown(prior) {
			return prior
		}
		return types.StringValue(fallback)
	}
	listenDefault := ""
	if ttype == "tcp" {
		listenDefault = "127.0.0.1"
	}
	plan.TunnelListenAddress = omitted(config.TunnelListenAddress, state.TunnelListenAddress, listenDefault)
	plan.TunnelDefinitions = omitted(config.TunnelDefinitions, state.TunnelDefinitions, "")

	if ttype == "k8s" && config.Hostname.IsNull() {
		if plan.URL.IsUnknown() {
			plan.Hostname = types.StringUnknown()
		} else {
			plan.Hostname = types.StringValue(k8sHostname(plan.URL.ValueString()))
		}
	}
}

// untrimmedValidator rejects leading or trailing whitespace. The appliance
// trims every string it is sent, so such a value would come back different from
// the configuration and fail the apply as an inconsistent result. A PEM read
// with file() usually ends in a newline.
type untrimmedValidator struct{}

func (v untrimmedValidator) Description(context.Context) string {
	return "must not have leading or trailing whitespace"
}

func (v untrimmedValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v untrimmedValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !isKnown(req.ConfigValue) {
		return
	}
	value := req.ConfigValue.ValueString()
	if strings.TrimFunc(value, isTrimmedByAppliance) != value {
		resp.Diagnostics.AddAttributeError(req.Path, "Value has surrounding whitespace",
			"The appliance trims this value; use trimspace(...).")
	}
}

// isTrimmedByAppliance matches what Laravel's TrimStrings middleware strips:
// whitespace plus the BOM, zero-width space and left-to-right mark.
func isTrimmedByAppliance(r rune) bool {
	return unicode.IsSpace(r) || r == '\uFEFF' || r == '\u200B' || r == '\u200E'
}
