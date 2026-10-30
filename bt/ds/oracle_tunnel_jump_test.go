package ds

import (
	"testing"

	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// The Oracle list endpoint filters on gateway_id, asset_group_id and
// asset_group_type, and answers 400 to the standard names.
func TestOracleFilterNames(t *testing.T) {
	filter := api.MakeFilterMap(oracleTunnelJumpDataSourceModel{
		Name:          types.StringValue("ora"),
		JumpointID:    types.Int64Value(3),
		Hostname:      types.StringValue("h"),
		JumpGroupID:   types.Int64Value(4),
		JumpGroupType: types.StringValue("personal"),
		Tag:           types.StringValue("t"),
	})

	assert.Equal(t, map[string]string{
		"name":             "ora",
		"gateway_id":       "3",
		"hostname":         "h",
		"asset_group_id":   "4",
		"asset_group_type": "personal",
		"tag":              "t",
	}, filter)
}
