package rs

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Oracle endpoint names three fields differently from every other jump
// item, and rejects unknown keys. The Terraform attributes keep the standard
// names; these pin that the wire carries the Oracle ones, in both directions.
func TestOracleWireNamesOut(t *testing.T) {
	m := models.OracleTunnelJump{
		ID:                  types.StringNull(),
		Name:                types.StringValue("ora"),
		JumpointID:          types.Int64Value(3),
		Hostname:            types.StringValue("ora.example.com"),
		JumpGroupID:         types.Int64Value(4),
		JumpGroupType:       types.StringValue("personal"),
		Tag:                 types.StringValue(""),
		Comments:            types.StringValue(""),
		JumpPolicyID:        types.Int64Null(),
		SessionPolicyID:     types.Int64Null(),
		TunnelListenAddress: types.StringValue("127.0.0.4"),
		Username:            types.StringValue("scott"),
		Database:            types.StringValue(""),
	}
	body := wireBody[api.OracleTunnelJump](t, m)

	assert.Equal(t, float64(3), body["gateway_id"])
	assert.Equal(t, float64(4), body["asset_group_id"])
	assert.Equal(t, "personal", body["asset_group_type"])
	assert.Equal(t, "127.0.0.4", body["local_address"])
	assert.Equal(t, "scott", body["username"])
	assert.Contains(t, body, "database", `database is always sent, so "" clears it`)
	for _, key := range []string{"jumpoint_id", "jump_group_id", "jump_group_type", "tunnel_listen_address", "tunnel_type"} {
		assert.NotContains(t, body, key)
	}
}

func TestOracleWireNamesIn(t *testing.T) {
	var item api.OracleTunnelJump
	require.NoError(t, json.Unmarshal([]byte(`{"id":9,"name":"ora","gateway_id":3,"hostname":"h","asset_group_id":4,
		"asset_group_type":"personal","tag":"","comments":"","local_address":"127.0.0.4","username":"scott","database":"orcl"}`), &item))

	var m models.OracleTunnelJump
	require.NoError(t, api.CopyAPItoTF(context.Background(), reflect.ValueOf(&item).Elem(), reflect.ValueOf(&m).Elem(), reflect.TypeOf(item), api.ProductPRA))

	assert.Equal(t, "9", m.ID.ValueString())
	assert.Equal(t, int64(3), m.JumpointID.ValueInt64())
	assert.Equal(t, int64(4), m.JumpGroupID.ValueInt64())
	assert.Equal(t, "personal", m.JumpGroupType.ValueString())
	assert.Equal(t, "127.0.0.4", m.TunnelListenAddress.ValueString())
	assert.Equal(t, "orcl", m.Database.ValueString())
}
