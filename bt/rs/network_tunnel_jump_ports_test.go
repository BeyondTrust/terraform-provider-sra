package rs

import (
	"context"
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyNetworkTunnelPortsList(t *testing.T) {
	var plan models.NetworkTunnelJump

	// rule with ip_addresses and ports.list = [22, 80]
	elemType := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
		"ip_addresses": types.ObjectType{AttrTypes: map[string]attr.Type{"list": types.ListType{ElemType: types.StringType}}},
		"ports":        types.ObjectType{AttrTypes: map[string]attr.Type{"list": types.ListType{ElemType: types.Int64Type}}},
	})

	portsObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.Int64Type}}, map[string]attr.Value{"list": types.ListValueMust(types.Int64Type, []attr.Value{types.Int64Value(22), types.Int64Value(80)})})

	ipList := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.1")})
	ipObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{"list": ipList})

	obj := types.ObjectValueMust(elemType.AttributeTypes(), map[string]attr.Value{
		"ip_addresses": ipObj,
		"ports":        portsObj,
	})

	plan.FilterRules = types.ListValueMust(elemType, []attr.Value{obj})

	diags := applyNetworkTunnelValidate(context.Background(), &plan)
	assert.False(t, diags.HasError())
}

func TestApplyNetworkTunnelPortsRange(t *testing.T) {
	var plan models.NetworkTunnelJump

	// rule with ip_addresses and ports.range = { start=1000, end=2000 }
	elemType := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
		"ip_addresses": types.ObjectType{AttrTypes: map[string]attr.Type{"list": types.ListType{ElemType: types.StringType}}},
		"ports":        types.ObjectType{AttrTypes: map[string]attr.Type{"range": types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}}}},
	})

	rangeObj := types.ObjectValueMust(map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}, map[string]attr.Value{"start": types.Int64Value(1000), "end": types.Int64Value(2000)})
	portsObj := types.ObjectValueMust(map[string]attr.Type{"range": types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}}}, map[string]attr.Value{"range": rangeObj})

	ipList := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.1")})
	ipObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{"list": ipList})

	obj := types.ObjectValueMust(elemType.AttributeTypes(), map[string]attr.Value{
		"ip_addresses": ipObj,
		"ports":        portsObj,
	})

	plan.FilterRules = types.ListValueMust(elemType, []attr.Value{obj})

	diags := applyNetworkTunnelValidate(context.Background(), &plan)
	assert.False(t, diags.HasError())
}
