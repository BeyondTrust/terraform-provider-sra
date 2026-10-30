package api

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestFilterMap(t *testing.T) {
	t.Parallel()

	type filterTestModel struct {
		Name         types.String  `filter:"name"`
		Tag          types.String  `filter:"tag"`
		JumpGroupID  types.Int64   `filter:"jump_group_id"`
		AnotherID    types.Int64   `filter:"another_id"`
		NotSupported types.Float64 `filter:"not_supported"`
		NotMapped    types.String
	}

	testModel := &filterTestModel{
		Name:         types.StringValue("filter_name"),
		Tag:          types.StringNull(),
		JumpGroupID:  types.Int64Value(24601),
		AnotherID:    types.Int64Null(),
		NotSupported: types.Float64Value(246.01),
		NotMapped:    types.StringValue("doesn't_really_matter"),
	}

	filter := MakeFilterMap(*testModel)

	assert.Equal(t, testModel.Name.ValueString(), filter["name"])
	assert.Equal(t, strconv.Itoa(int(testModel.JumpGroupID.ValueInt64())), filter["jump_group_id"])
	_, exists := filter["not_supported"]
	assert.False(t, exists)
	_, exists = filter["tag"]
	assert.False(t, exists)
	_, exists = filter["another_id"]
	assert.False(t, exists)
	_, exists = filter["NotMapped"]
	assert.False(t, exists)
}

// Before bools were handled, include_personal on the single SSH account data
// source was silently dropped.
func TestFilterMapBool(t *testing.T) {
	t.Parallel()

	type boolModel struct {
		On      types.Bool `filter:"on"`
		Off     types.Bool `filter:"off"`
		Null    types.Bool `filter:"null"`
		Unknown types.Bool `filter:"unknown"`
	}

	filter := MakeFilterMap(boolModel{
		On:      types.BoolValue(true),
		Off:     types.BoolValue(false),
		Null:    types.BoolNull(),
		Unknown: types.BoolUnknown(),
	})

	assert.Equal(t, map[string]string{"on": "true", "off": "false"}, filter)
}
