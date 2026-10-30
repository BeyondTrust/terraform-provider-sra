package rs

import (
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyPostgresDefaultsAndValidate(t *testing.T) {
	// The listen address is ModifyPlan's (planTunnelListenAddress) and the
	// validator's; this leaves it as it found it, valid or not.
	for _, addr := range []types.String{types.StringNull(), types.StringValue(""), types.StringValue("::ffff:127.0.0.1")} {
		var plan models.PostgreSQLTunnelJump
		plan.Name = types.StringValue("example")
		plan.TunnelListenAddress = addr
		diags := applyPostgresDefaultsAndValidate(&plan)
		assert.False(t, diags.HasError())
		assert.Equal(t, addr, plan.TunnelListenAddress)
	}
}

func TestApplyPostgresInvalidName(t *testing.T) {
	var plan models.PostgreSQLTunnelJump
	plan.Name = types.StringValue("")
	diags := applyPostgresDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}
