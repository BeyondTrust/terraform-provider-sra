package rs

import (
	"context"
	"net/netip"

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
// allowEmpty accepts "": the mysql and postgresql APIs take an empty address
// (their rule is ipv4 without filled), and existing configurations set it.
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
// It is ModifyPlan, not a schema Default, because a Default replaces only a
// null configuration value and cannot see state.
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
