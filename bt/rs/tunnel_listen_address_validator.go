package rs

import (
	"context"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// tunnelListenLoopback is the range the appliance binds tunnel listeners in.
var tunnelListenLoopback = netip.MustParsePrefix("127.0.0.0/24")

// tunnelListenAddressError returns why addr is not a usable tunnel listen
// address, or "" when it is. This is the same range check as
// applyMySQLDefaultsAndValidate, made stricter where the product is: the
// appliance's rule is Laravel `ipv4`, so an IPv4-mapped IPv6 form such as
// "::ffff:127.0.0.1" is rejected here although net.ParseIP would accept it. A
// dotted quad is at most 15 characters, so the appliance's 32-character limit
// needs no separate check.
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
// value. Null and unknown values pass: the schema default fills a null, and an
// unknown is checked once it is known.
type tunnelListenAddressValidator struct{}

func (v tunnelListenAddressValidator) Description(context.Context) string {
	return "must be an IPv4 address within 127.0.0.0/24"
}

func (v tunnelListenAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v tunnelListenAddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if msg := tunnelListenAddressError(req.ConfigValue.ValueString()); msg != "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid tunnel_listen_address", "tunnel_listen_address "+msg+".")
	}
}
