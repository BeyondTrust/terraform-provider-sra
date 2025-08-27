package rs

import (
	"context"
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyNetworkTunnelValidate(t *testing.T) {
	var plan models.NetworkTunnelJump

	// missing filter_rules should return error
	plan.FilterRules = types.ListNull(types.ObjectType{})
	diags := applyNetworkTunnelValidate(context.Background(), &plan)
	assert.True(t, diags.HasError())

	// present filter_rules should pass (JSON array with one rule that includes ip_addresses)
	objType := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{"ip_addresses": types.ListType{ElemType: types.StringType}})
	obj := types.ObjectValueMust(objType.AttributeTypes(), map[string]attr.Value{"ip_addresses": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.168.1.1")})})
	lv := types.ListValueMust(objType, []attr.Value{obj})
	plan.FilterRules = lv
	diags = applyNetworkTunnelValidate(context.Background(), &plan)
	assert.False(t, diags.HasError())
}

func TestApplyNetworkTunnelInvalidJSON(t *testing.T) {
	var plan models.NetworkTunnelJump
	// invalid structured value - leave as raw single-object with bad data (simulate user error)
	emptyElemType := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{})
	plan.FilterRules = types.ListValueMust(emptyElemType, []attr.Value{})
	diags := applyNetworkTunnelValidate(context.Background(), &plan)
	assert.True(t, diags.HasError())
}

func TestApplyNetworkTunnelMissingIPAddresses(t *testing.T) {
	var plan models.NetworkTunnelJump
	// an object missing ip_addresses
	objType2 := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{"protocol": types.StringType})
	obj := types.ObjectValueMust(objType2.AttributeTypes(), map[string]attr.Value{"protocol": types.StringValue("TCP")})
	plan.FilterRules = types.ListValueMust(objType2, []attr.Value{obj})
	diags := applyNetworkTunnelValidate(context.Background(), &plan)
	assert.True(t, diags.HasError())
}

func TestApplyNetworkTunnelTooManyItems(t *testing.T) {
	var plan models.NetworkTunnelJump
	// create an array with 51 empty objects
	var elems []attr.Value
	for i := 0; i < 51; i++ {
		elems = append(elems, types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}))
	}
	emptyElemType2 := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{})
	plan.FilterRules = types.ListValueMust(emptyElemType2, elems)
	diags := applyNetworkTunnelValidate(context.Background(), &plan)
	assert.True(t, diags.HasError())
}
