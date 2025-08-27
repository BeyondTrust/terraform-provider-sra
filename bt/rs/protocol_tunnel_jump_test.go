package rs

import (
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyProtocolTunnelValidTCP(t *testing.T) {
	var plan models.ProtocolTunnelJump
	plan.Name = types.StringValue("example")
	plan.TunnelType = types.StringValue("tcp")
	plan.TunnelDefinitions = types.StringValue("22;24;26;28")
	plan.TunnelListenAddress = types.StringNull()

	diags := applyProtocolTunnelDefaultsAndValidate(&plan)
	assert.False(t, diags.HasError())
	assert.Equal(t, "127.0.0.1", plan.TunnelListenAddress.ValueString())
}

func TestApplyProtocolTunnelOddDefinitions(t *testing.T) {
	var plan models.ProtocolTunnelJump
	plan.Name = types.StringValue("example")
	plan.TunnelType = types.StringValue("tcp")
	plan.TunnelDefinitions = types.StringValue("22;24;26")

	diags := applyProtocolTunnelDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}

func TestApplyProtocolTunnelPortRange(t *testing.T) {
	var plan models.ProtocolTunnelJump
	plan.Name = types.StringValue("example")
	plan.TunnelType = types.StringValue("tcp")
	plan.TunnelDefinitions = types.StringValue("70000;24")

	diags := applyProtocolTunnelDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}

func TestApplyProtocolTunnelK8sMissingFields(t *testing.T) {
	var plan models.ProtocolTunnelJump
	plan.Name = types.StringValue("example")
	plan.TunnelType = types.StringValue("k8s")

	diags := applyProtocolTunnelDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}

func TestApplyProtocolTunnelListenAddressOutsideRange(t *testing.T) {
	var plan models.ProtocolTunnelJump
	plan.Name = types.StringValue("example")
	plan.TunnelType = types.StringValue("tcp")
	plan.TunnelDefinitions = types.StringValue("22;24")
	plan.TunnelListenAddress = types.StringValue("192.168.1.1")

	diags := applyProtocolTunnelDefaultsAndValidate(&plan)
	assert.True(t, diags.HasError())
}
