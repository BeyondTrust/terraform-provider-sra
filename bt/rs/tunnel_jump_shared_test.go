package rs

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
	for name, tc := range map[string]struct {
		r resource.Resource
		// The mysql and postgresql APIs accept "", and existing configurations
		// set it; mongodb and oracle never did.
		allowsEmpty bool
	}{
		"mongodb":    {&mongodbTunnelJumpResource{}, false},
		"oracle":     {&oracleTunnelJumpResource{}, false},
		"mysql":      {&mysqlTunnelJumpResource{}, true},
		"postgresql": {&postgresqlTunnelJumpResource{}, true},
	} {
		t.Run(name, func(t *testing.T) {
			sch := resourceSchema(t, tc.r)
			for value, ok := range map[string]bool{
				"127.0.0.1":   true,
				"127.0.0.5":   true,
				"127.0.0.255": true,
				"127.0.1.1":   false,
				"10.0.0.1":    false,
				"192.168.1.1": false,
				"::1":         false,
				// IPv4-mapped forms: net.ParseIP accepts them, the appliance's
				// ipv4 rule does not.
				"::ffff:127.0.0.1":         false,
				"::ffff:7f00:1":            false,
				"0:0:0:0:0:ffff:127.0.0.1": false,
				"127.0.0.01":               false,
				"localhost":                false,
				"":                         tc.allowsEmpty,
				" 127.0.0.1":               false,
			} {
				assert.Equal(t, ok, runStringValidators(t, sch, "tunnel_listen_address", types.StringValue(value)), "%q", value)
			}
			assert.True(t, runStringValidators(t, sch, "tunnel_listen_address", types.StringNull()))
			assert.True(t, runStringValidators(t, sch, "tunnel_listen_address", types.StringUnknown()))
		})
	}
}

func TestPlanTunnelListenAddress(t *testing.T) {
	for name, tc := range map[string]struct {
		config, prior, want types.String
	}{
		"create, omitted":             {types.StringNull(), types.StringNull(), types.StringValue("127.0.0.1")},
		"omitted keeps the appliance": {types.StringNull(), types.StringValue("127.0.0.5"), types.StringValue("127.0.0.5")},
		"empty keeps the appliance":   {types.StringValue(""), types.StringValue("127.0.0.5"), types.StringValue("127.0.0.5")},
		"omitted, prior empty":        {types.StringNull(), types.StringValue(""), types.StringValue("127.0.0.1")},
		"unknown stays unknown":       {types.StringUnknown(), types.StringValue("127.0.0.5"), types.StringUnknown()},
		"configured wins":             {types.StringValue("127.0.0.9"), types.StringValue("127.0.0.5"), types.StringValue("127.0.0.9")},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, planTunnelListenAddress(tc.config, tc.prior))
		})
	}
}

func TestValidateSQLTunnelPlan(t *testing.T) {
	name, host := types.StringValue("example"), types.StringValue("db.example")
	id := types.Int64Value(1)
	for label, tc := range map[string]struct {
		name, hostname      types.String
		jumpoint, jumpGroup types.Int64
		ok                  bool
	}{
		"valid":                   {name, host, id, id, true},
		"zero ids pass":           {name, host, types.Int64Value(0), types.Int64Value(0), true},
		"unknown ids pass":        {name, host, types.Int64Unknown(), types.Int64Unknown(), true},
		"unknown name passes":     {types.StringUnknown(), host, id, id, true},
		"unknown hostname passes": {name, types.StringUnknown(), id, id, true},
		"empty name":              {types.StringValue(""), host, id, id, false},
		"null name":               {types.StringNull(), host, id, id, false},
		"name over 128":           {types.StringValue(strings.Repeat("n", 129)), host, id, id, false},
		"empty hostname":          {name, types.StringValue(""), id, id, false},
		"negative jumpoint_id":    {name, host, types.Int64Value(-1), id, false},
		"negative jump_group_id":  {name, host, id, types.Int64Value(-1), false},
	} {
		diags := validateSQLTunnelPlan(tc.name, tc.hostname, tc.jumpoint, tc.jumpGroup)
		assert.Equal(t, tc.ok, !diags.HasError(), "%s: %v", label, diags)
	}
}

// A name or hostname taken from another resource's output plans.
func TestSQLTunnelUnknownNameAndHostnamePlan(t *testing.T) {
	for _, name := range []string{"mysql", "postgresql"} {
		t.Run(name, func(t *testing.T) {
			r := tunnelHarnesses[name].plan(t, tunnelConfig(map[string]any{"name": unknown, "hostname": unknown}), nil)
			require.Empty(t, r.diags)
			assert.False(t, r.planned["name"].IsKnown())
		})
	}
}

// tunnelHarnesses are the tunnel types whose listen address tunnelListenAddressPlan plans.
var tunnelHarnesses = map[string]planHarness{
	"mysql":      {newMySQLTunnelJumpResource},
	"postgresql": {newPostgreSQLTunnelJumpResource},
	"mongodb":    {newMongodbTunnelJumpResource},
	"oracle":     {newOracleTunnelJumpResource},
}

func tunnelConfig(extra map[string]any) map[string]any {
	c := map[string]any{"name": "db", "jumpoint_id": 1, "hostname": "db.example", "jump_group_id": 2}
	for k, v := range extra {
		c[k] = v
	}
	return c
}

// tunnelState is the state an apply of config would leave, taken from the
// create plan with the id filled in.
func tunnelState(t *testing.T, h planHarness, config map[string]any) map[string]tftypes.Value {
	t.Helper()
	r := h.plan(t, config, nil)
	require.Empty(t, r.diags)
	state := map[string]tftypes.Value{}
	for k, v := range r.planned {
		state[k] = v
	}
	state["id"] = tftypes.NewValue(tftypes.String, "1")
	return state
}

func TestTunnelListenAddressPlan(t *testing.T) {
	for name, h := range tunnelHarnesses {
		t.Run(name, func(t *testing.T) {
			// Proves the plan modifier is wired: without it, a Computed attribute
			// with no default plans as unknown on create.
			t.Run("create with the address omitted plans 127.0.0.1", func(t *testing.T) {
				r := h.plan(t, tunnelConfig(nil), nil)
				require.Empty(t, r.diags)
				assert.Equal(t, "127.0.0.1", str(t, r.planned["tunnel_listen_address"]))
			})

			// The framework marks an omitted Computed attribute unknown only when
			// something else changes, which is when the old code reset it.
			t.Run("an imported address survives a change to another attribute", func(t *testing.T) {
				state := tunnelState(t, h, tunnelConfig(map[string]any{"tunnel_listen_address": "127.0.0.5"}))
				r := h.plan(t, tunnelConfig(map[string]any{"tag": "changed"}), state)
				require.Empty(t, r.diags)
				assert.Equal(t, "127.0.0.5", str(t, r.planned["tunnel_listen_address"]))
				assert.Equal(t, "changed", str(t, r.planned["tag"]))
			})

			t.Run("an unknown address stays unknown", func(t *testing.T) {
				r := h.plan(t, tunnelConfig(map[string]any{"tunnel_listen_address": unknown}), nil)
				require.Empty(t, r.diags)
				assert.False(t, r.planned["tunnel_listen_address"].IsKnown())
			})

			t.Run("an out-of-range state value is not validated", func(t *testing.T) {
				state := tunnelState(t, h, tunnelConfig(nil))
				state["tunnel_listen_address"] = tftypes.NewValue(tftypes.String, "127.1.0.1")
				r := h.plan(t, tunnelConfig(map[string]any{"tag": "changed"}), state)
				require.Empty(t, r.diags)
				assert.Equal(t, "127.1.0.1", str(t, r.planned["tunnel_listen_address"]))
			})
		})
	}
}

// v140SQLState is a mysql or postgresql item as provider v1.4.0 writes it to
// state for tunnelConfig(nil). v1.4.0 planned an omitted or empty listen address
// as 127.0.0.1, and left the two policy ids null when unset.
func v140SQLState(t *testing.T, h planHarness) map[string]tftypes.Value {
	return h.values(t, map[string]any{
		"id": "13", "name": "db", "jumpoint_id": 1, "hostname": "db.example", "jump_group_id": 2,
		"jump_group_type": "shared", "tag": "", "comments": "",
		"tunnel_listen_address": "127.0.0.1", "username": "", "database": "",
	})
}

// State written by v1.4.0 replans clean, whether the configuration omits the
// address, sets it to "" or sets 127.0.0.1.
func TestTunnelListenAddressV140StateReplansClean(t *testing.T) {
	for _, name := range []string{"mysql", "postgresql"} {
		h := tunnelHarnesses[name]
		for label, configured := range map[string]any{"omitted": nil, "empty": "", "explicit": "127.0.0.1"} {
			t.Run(name+"/"+label, func(t *testing.T) {
				config := tunnelConfig(nil)
				if configured != nil {
					config["tunnel_listen_address"] = configured
				}
				state := v140SQLState(t, h)

				r := h.plan(t, config, state)
				require.Empty(t, r.diags)
				assert.True(t, r.plannedRaw.Equal(tftypes.NewValue(h.typ(t), state)), "plan differs from state")
			})
		}
	}
}

// An empty address has nothing to keep on create, or when state holds "" too.
// Planning 127.0.0.1 there would not match the configuration, which Terraform
// reports as "Provider produced invalid plan"; it is an attribute error instead.
func TestTunnelListenAddressEmptyWithNothingToKeep(t *testing.T) {
	for _, name := range []string{"mysql", "postgresql"} {
		h := tunnelHarnesses[name]
		config := tunnelConfig(map[string]any{"tunnel_listen_address": ""})
		priorEmpty := v140SQLState(t, h)
		priorEmpty["tunnel_listen_address"] = tftypes.NewValue(tftypes.String, "")
		for label, prior := range map[string]map[string]tftypes.Value{"create": nil, "prior empty": priorEmpty} {
			t.Run(name+"/"+label, func(t *testing.T) {
				r := h.plan(t, config, prior)
				require.Len(t, r.diags, 1)
				assert.Equal(t, "Invalid tunnel_listen_address", r.diags[0].Summary)
				assert.Contains(t, r.diags[0].Detail, "omit it to get 127.0.0.1")
				assert.True(t, r.diags[0].Attribute.Equal(tftypes.NewAttributePath().WithAttributeName("tunnel_listen_address")))
			})
		}
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
