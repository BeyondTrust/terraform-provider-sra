package rs

import (
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyMySQLDefaultsAndValidate(t *testing.T) {
	// The listen address is ModifyPlan's (planTunnelListenAddress) and the
	// validator's; this leaves it as it found it, valid or not.
	for _, addr := range []types.String{types.StringNull(), types.StringValue(""), types.StringValue("::ffff:127.0.0.1")} {
		var plan models.MySQLTunnelJump
		plan.Name = types.StringValue("example")
		plan.TunnelListenAddress = addr
		diags := applyMySQLDefaultsAndValidate(&plan)
		assert.False(t, diags.HasError())
		assert.Equal(t, addr, plan.TunnelListenAddress)
	}
}

func TestApplyMySQLInvalidName(t *testing.T) {
	var plan models.MySQLTunnelJump
	plan.Name = types.StringValue("")
	diags := applyMySQLDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}
