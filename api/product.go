package api

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const ProductRS = "RS"
const ProductPRA = "PRA"

type RestrictsProducts interface {
	AllowRS() bool
	AllowPRA() bool
}

func IsProductAllowed(ctx context.Context, i interface{}, product string) bool {
	s, ok := i.(RestrictsProducts)

	if !ok {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not OK [%+v]\n", i))
		// Doesn't restrict products, so everything is allowed
		return true
	}

	if !s.AllowRS() && product == ProductRS {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not RS [%+v]\n", s))
		return false
	}
	if !s.AllowPRA() && product == ProductPRA {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not PRA [%+v]\n", s))
		return false
	}
	tflog.Trace(ctx, fmt.Sprintf("🌈 Yes, allowed [%+v]\n", s))
	return true
}
