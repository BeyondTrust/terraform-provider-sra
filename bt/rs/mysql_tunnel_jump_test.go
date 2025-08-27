package rs

import (
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyMySQLDefaultsAndValidate(t *testing.T) {
	var plan models.MySQLTunnelJump
	plan.Name = types.StringValue("example")
	plan.TunnelListenAddress = types.StringNull()
	diags := applyMySQLDefaultsAndValidate(&plan)
	assert.False(t, diags.HasError())
	assert.Equal(t, "127.0.0.1", plan.TunnelListenAddress.ValueString())
}

func TestApplyMySQLInvalidName(t *testing.T) {
	var plan models.MySQLTunnelJump
	plan.Name = types.StringValue("")
	diags := applyMySQLDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}
