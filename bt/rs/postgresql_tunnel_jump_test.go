package rs

import (
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyPostgresDefaultsAndValidate(t *testing.T) {
	var plan models.PostgreSQLTunnelJump

	// empty plan should get default listen address
	plan.Name = types.StringValue("example")
	plan.TunnelListenAddress = types.StringNull()
	diags := applyPostgresDefaultsAndValidate(&plan)
	assert.False(t, diags.HasError())
	assert.Equal(t, "127.0.0.1", plan.TunnelListenAddress.ValueString())
}

func TestApplyPostgresInvalidName(t *testing.T) {
	var plan models.PostgreSQLTunnelJump
	plan.Name = types.StringValue("")
	diags := applyPostgresDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}
