package rs

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resourceSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	return resp.Schema
}

// wireBody is the PRA request body CopyTFtoAPI builds from m, decoded back into
// a map so a test can check which keys are sent.
func wireBody[TApi any, TTf any](t *testing.T, m TTf) map[string]any {
	t.Helper()
	var item TApi
	api.CopyTFtoAPI(context.Background(), reflect.ValueOf(&m).Elem(), reflect.ValueOf(&item).Elem(), api.ProductPRA)
	b, err := json.Marshal(item)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(b, &body))
	return body
}

// runStringValidators runs the schema's own validators for attr against value.
func runStringValidators(t *testing.T, sch schema.Schema, attr string, value types.String) bool {
	t.Helper()
	a, ok := sch.Attributes[attr].(schema.StringAttribute)
	require.True(t, ok, "%s is not a string attribute", attr)
	resp := &validator.StringResponse{}
	for _, v := range a.Validators {
		v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root(attr), ConfigValue: value}, resp)
	}
	return !resp.Diagnostics.HasError()
}

func TestTunnelListenAddressValidator(t *testing.T) {
	for name, r := range map[string]resource.Resource{
		"mongodb": &mongodbTunnelJumpResource{},
		"oracle":  &oracleTunnelJumpResource{},
	} {
		t.Run(name, func(t *testing.T) {
			sch := resourceSchema(t, r)
			for value, ok := range map[string]bool{
				"127.0.0.1":        true,
				"127.0.0.5":        true,
				"127.0.0.255":      true,
				"127.0.1.1":        false,
				"10.0.0.1":         false,
				"192.168.1.1":      false,
				"::1":              false,
				"::ffff:127.0.0.1": false,
				"127.0.0.01":       false,
				"localhost":        false,
				"":                 false,
				" 127.0.0.1":       false,
			} {
				assert.Equal(t, ok, runStringValidators(t, sch, "tunnel_listen_address", types.StringValue(value)), "%q", value)
			}
			assert.True(t, runStringValidators(t, sch, "tunnel_listen_address", types.StringNull()))
			assert.True(t, runStringValidators(t, sch, "tunnel_listen_address", types.StringUnknown()))
		})
	}
}

func TestTunnelJumpGroupTypeValidator(t *testing.T) {
	for name, r := range map[string]resource.Resource{
		"mongodb":  &mongodbTunnelJumpResource{},
		"oracle":   &oracleTunnelJumpResource{},
		"protocol": &protocolTunnelJumpResource{},
	} {
		t.Run(name, func(t *testing.T) {
			sch := resourceSchema(t, r)
			for value, ok := range map[string]bool{
				"shared":   true,
				"personal": true,
				"public":   false,
				"asset":    false,
			} {
				assert.Equal(t, ok, runStringValidators(t, sch, "jump_group_type", types.StringValue(value)), "%q", value)
			}
		})
	}
}
