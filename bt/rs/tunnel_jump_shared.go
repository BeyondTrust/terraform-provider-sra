package rs

import (
	"context"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tunnelListenAddressDescription is the listen address documentation shared by
// the mysql, postgresql, mongodb and oracle tunnel types.
const tunnelListenAddressDescription = "Local address the tunnel listens on: a plain IPv4 address within 127.0.0.0/24. " +
	"When omitted, it keeps the value the appliance holds, and defaults to 127.0.0.1 on create. " +
	"To reset an item to 127.0.0.1, set it explicitly. Omit it rather than setting it to an empty string."

// tunnelListenAddressLocalAddressNote documents the rename for the tunnel types
// whose API calls the field local_address.
const tunnelListenAddressLocalAddressNote = " Sent to the API as `local_address`; API errors use that name."

// tunnelListenAddressDefault is what the appliance binds a new tunnel to when
// it is given no address.
const tunnelListenAddressDefault = "127.0.0.1"

const tunnelJumpNotAssociableNote = "Vault account jump item associations accept only MSSQL protocol tunnels on the appliance, " +
	"so this type can't be associated with a vault account."

func isKnown(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }

// isSet reports a known, non-empty value. Null, unknown and "" are not set.
func isSet(v types.String) bool { return isKnown(v) && v.ValueString() != "" }

// tunnelListenLoopback is the range the appliance binds tunnel listeners in.
var tunnelListenLoopback = netip.MustParsePrefix("127.0.0.0/24")

// tunnelListenAddressError returns why addr is not a usable tunnel listen
// address, or "" when it is. The appliance's rule is Laravel `ipv4`, so an
// IPv4-mapped IPv6 form such as "::ffff:127.0.0.1" is rejected here although
// net.ParseIP would accept it. The appliance accepts all of 127/8, not just the
// /24 its message names; this keeps to the /24 the provider has always
// enforced. A dotted quad is at most 15 characters, so the appliance's
// 32-character limit needs no separate check.
func tunnelListenAddressError(addr string) string {
	ip, err := netip.ParseAddr(addr)
	if err != nil || !ip.Is4() {
		return "must be an IPv4 address, such as 127.0.0.1"
	}
	if !tunnelListenLoopback.Contains(ip) {
		return "must be within the 127.0.0.0/24 subnet"
	}
	return ""
}

// tunnelListenAddressValidator applies tunnelListenAddressError to a known
// value. Null and unknown values pass: planTunnelListenAddress fills a null,
// and an unknown is checked once it is known. It sees only configuration, never
// state, so an imported item outside the range does not fail every plan.
//
// allowEmpty accepts "" for mysql and postgresql, whose existing configurations
// set it. It is valid only on an existing item, where it keeps the current
// address; tunnelListenAddressPlan rejects it on create.
type tunnelListenAddressValidator struct {
	allowEmpty bool
}

func (v tunnelListenAddressValidator) Description(context.Context) string {
	return "must be an IPv4 address within 127.0.0.0/24"
}

func (v tunnelListenAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v tunnelListenAddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !isKnown(req.ConfigValue) {
		return
	}
	if v.allowEmpty && req.ConfigValue.ValueString() == "" {
		return
	}
	if msg := tunnelListenAddressError(req.ConfigValue.ValueString()); msg != "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid tunnel_listen_address", "tunnel_listen_address "+msg+".")
	}
}

// planTunnelListenAddress is the planned listen address for a configured value
// and the prior state's. An omitted (or empty) address keeps what the appliance
// holds, so an imported item listening on 127.0.0.5 does not plan a change back
// to the default; on create it is the default. An unknown stays unknown, to be
// planned once it is known.
//
// It is a plan modifier, not a schema Default, because a Default replaces only
// a null configuration value and cannot see state.
func planTunnelListenAddress(config, prior types.String) types.String {
	switch {
	case config.IsUnknown():
		return types.StringUnknown()
	case isSet(config):
		return config
	case isSet(prior):
		return prior
	default:
		return types.StringValue(tunnelListenAddressDefault)
	}
}

// tunnelListenAddressPlan applies planTunnelListenAddress as the attribute's
// plan modifier.
type tunnelListenAddressPlan struct{}

func (tunnelListenAddressPlan) Description(context.Context) string {
	return "an omitted address keeps the prior value, or 127.0.0.1 on create"
}

func (m tunnelListenAddressPlan) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (tunnelListenAddressPlan) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if isKnown(req.ConfigValue) && req.ConfigValue.ValueString() == "" && !isSet(req.StateValue) {
		// No address to keep. Planning the default would not match the
		// configured "", which Terraform rejects as an invalid plan.
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid tunnel_listen_address",
			"tunnel_listen_address can't be empty here; omit it to get 127.0.0.1 or set an address.")
		return
	}
	resp.PlanValue = planTunnelListenAddress(req.ConfigValue, req.StateValue)
}

// validateSQLTunnelPlan checks the planned name, hostname and ids of a mysql or
// postgresql tunnel. A negative id is invalid; 0 passes, since an id that is not
// known yet can plan as 0, and the appliance validates it on apply. Unknown
// values pass.
func validateSQLTunnelPlan(name, hostname types.String, jumpointID, jumpGroupID types.Int64) diag.Diagnostics {
	var diags diag.Diagnostics
	// An unknown name or hostname (another resource's output) is checked once
	// it is known; its ValueString() is "".
	if !name.IsUnknown() {
		if name.IsNull() || name.ValueString() == "" {
			diags.AddError("name is required", "The name field is required and must be 1..128 characters")
			return diags
		}
		if l := len(name.ValueString()); l > 128 {
			diags.AddError("name length", "The name must be between 1 and 128 characters")
		}
	}
	if isKnown(hostname) {
		if l := len(hostname.ValueString()); l < 1 || l > 128 {
			diags.AddError("hostname length", "The hostname must be between 1 and 128 characters")
		}
	}
	if !jumpointID.IsNull() && !jumpointID.IsUnknown() && jumpointID.ValueInt64() < 0 {
		diags.AddError("jumpoint_id invalid", "jumpoint_id must be >= 0")
	}
	if !jumpGroupID.IsNull() && !jumpGroupID.IsUnknown() && jumpGroupID.ValueInt64() < 0 {
		diags.AddError("jump_group_id invalid", "jump_group_id must be >= 0")
	}
	return diags
}
