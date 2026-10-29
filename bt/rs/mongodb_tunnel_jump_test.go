package rs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mongodbSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	(&mongodbTunnelJumpResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	return resp.Schema
}

// mongodbModel is a fully known item, as a plan or state holds it after the
// schema defaults are applied.
func mongodbModel(authDatabase types.String) models.MongodbTunnelJump {
	return models.MongodbTunnelJump{
		ID:                  types.StringValue("7"),
		Name:                types.StringValue("mongo"),
		JumpointID:          types.Int64Value(1),
		Hostname:            types.StringValue("mongo.example.com"),
		JumpGroupID:         types.Int64Value(2),
		JumpGroupType:       types.StringValue("shared"),
		Tag:                 types.StringValue(""),
		Comments:            types.StringValue(""),
		JumpPolicyID:        types.Int64Null(),
		SessionPolicyID:     types.Int64Null(),
		TunnelListenAddress: types.StringValue("127.0.0.1"),
		Username:            types.StringValue(""),
		Database:            types.StringValue(""),
		AuthDatabase:        authDatabase,
	}
}

// mongodbWire marshals the request body CopyTFtoAPI builds from m.
func mongodbWire(t *testing.T, m models.MongodbTunnelJump) map[string]any {
	t.Helper()
	var item api.MongodbTunnelJump
	api.CopyTFtoAPI(context.Background(), reflect.ValueOf(&m).Elem(), reflect.ValueOf(&item).Elem(), api.ProductPRA)
	b, err := json.Marshal(item)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(b, &body))
	return body
}

// mongodbResponseBody is the item as the appliance returns it. A nil
// authDatabase leaves the key out, as appliances before 26.3.1 do.
func mongodbResponseBody(authDatabase *string) string {
	body := map[string]any{
		"id": 7, "name": "mongo", "jumpoint_id": 1, "hostname": "mongo.example.com",
		"jump_group_id": 2, "jump_group_type": "shared", "tag": "", "comments": "",
		"local_address": "127.0.0.1", "username": "", "database": "",
	}
	if authDatabase != nil {
		body["auth_database"] = *authDatabase
	}
	b, _ := json.Marshal(body)
	return string(b)
}

func TestMongodbWireNamesOut(t *testing.T) {
	body := mongodbWire(t, mongodbModel(types.StringValue("")))

	assert.Equal(t, "127.0.0.1", body["local_address"])
	assert.NotContains(t, body, "tunnel_listen_address")
	assert.NotContains(t, body, "tunnel_type", "the MongoDB endpoint rejects unknown keys")
	// Plain strings, so an empty value is sent and clears the field.
	assert.Contains(t, body, "username")
	assert.Equal(t, "", body["username"])
	assert.Contains(t, body, "database")
	assert.Equal(t, "", body["database"])
}

// auth_database is rejected outright before PRA 26.3.1, so it may only appear
// on the wire when the configuration gives it a value.
func TestMongodbAuthDatabaseOnlySentWhenSet(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value types.String
		want  any
	}{
		{"null", types.StringNull(), nil},
		{"unknown", types.StringUnknown(), nil},
		{"empty", types.StringValue(""), nil},
		{"set", types.StringValue("admin"), "admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := mongodbWire(t, mongodbModel(tc.value))
			if tc.want == nil {
				assert.NotContains(t, body, "auth_database")
			} else {
				assert.Equal(t, tc.want, body["auth_database"])
			}
		})
	}
}

func TestMongodbWireNamesIn(t *testing.T) {
	var item api.MongodbTunnelJump
	require.NoError(t, json.Unmarshal([]byte(mongodbResponseBody(nil)), &item))

	var m models.MongodbTunnelJump
	require.NoError(t, api.CopyAPItoTF(context.Background(), reflect.ValueOf(&item).Elem(), reflect.ValueOf(&m).Elem(), reflect.TypeOf(item), api.ProductPRA))

	assert.Equal(t, "127.0.0.1", m.TunnelListenAddress.ValueString(), "local_address must land on tunnel_listen_address")
	assert.True(t, m.AuthDatabase.IsNull(), "a missing auth_database key reads back as null")
}

// mongodbClient answers GET, POST and PATCH on the MongoDB endpoint with
// respBody, recording the PATCH body.
func mongodbClient(t *testing.T, respBody string, patched *string) *api.APIClient {
	t.Helper()
	c := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPatch && patched != nil {
			b, _ := io.ReadAll(r.Body)
			*patched = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
		return true
	})
	c.Product = api.ProductPRA
	return c
}

func mongodbState(t *testing.T, sch schema.Schema, m models.MongodbTunnelJump) tfsdk.State {
	t.Helper()
	s := tfsdk.State{Schema: sch}
	require.False(t, s.Set(context.Background(), &m).HasError())
	return s
}

func mongodbPlan(t *testing.T, sch schema.Schema, m models.MongodbTunnelJump) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: sch}
	require.False(t, p.Set(context.Background(), &m).HasError())
	return p
}

// Driven through the resource's own Update, so this proves the wrapper is wired
// to the clearing adjuster and not just that the adjuster works.
func TestMongodbUpdateClearsAuthDatabaseInPlace(t *testing.T) {
	ctx := context.Background()
	sch := mongodbSchema(t)

	for _, tc := range []struct {
		name         string
		state, plan  types.String
		wantSent     bool
		wantSentWith string
	}{
		{"cleared", types.StringValue("x"), types.StringValue(""), true, ""},
		{"already empty", types.StringValue(""), types.StringValue(""), false, ""},
		{"unknown plan never clears", types.StringValue("x"), types.StringUnknown(), false, ""},
		{"changed", types.StringValue("x"), types.StringValue("y"), true, "y"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var patched string
			echo := tc.wantSentWith
			r := &mongodbTunnelJumpResource{}
			r.ApiClient = mongodbClient(t, mongodbResponseBody(&echo), &patched)

			req := resource.UpdateRequest{
				State: mongodbState(t, sch, mongodbModel(tc.state)),
				Plan:  mongodbPlan(t, sch, mongodbModel(tc.plan)),
			}
			resp := &resource.UpdateResponse{State: tfsdk.State{Schema: sch}}
			r.Update(ctx, req, resp)
			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(patched), &body))
			if tc.wantSent {
				assert.Equal(t, tc.wantSentWith, body["auth_database"])
			} else {
				assert.NotContains(t, body, "auth_database")
			}
		})
	}
}

func TestClearAuthDatabaseLeavesOtherTransitionsAlone(t *testing.T) {
	for _, tc := range []struct {
		name        string
		state, plan types.String
	}{
		{"null state", types.StringNull(), types.StringValue("")},
		{"unknown state", types.StringUnknown(), types.StringValue("")},
		{"null plan", types.StringValue("x"), types.StringNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var item api.MongodbTunnelJump
			clearAuthDatabase(mongodbModel(tc.state), mongodbModel(tc.plan), &item)
			assert.Nil(t, item.AuthDatabase)
		})
	}
}

// Before 26.3.1 the appliance never returns auth_database. Every operation must
// still record "" so it matches the schema default, and the plan never comes
// back inconsistent.
func TestMongodbAuthDatabaseNormalisedToEmpty(t *testing.T) {
	ctx := context.Background()
	sch := mongodbSchema(t)
	empty := ""

	for _, tc := range []struct {
		name     string
		response *string
	}{
		{"key absent (before 26.3.1)", nil},
		{"key empty", &empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := mongodbResponseBody(tc.response)
			check := func(t *testing.T, s tfsdk.State) {
				t.Helper()
				var got types.String
				require.False(t, s.GetAttribute(ctx, path.Root("auth_database"), &got).HasError())
				assert.False(t, got.IsNull())
				assert.Equal(t, "", got.ValueString())
			}

			t.Run("create", func(t *testing.T) {
				r := &mongodbTunnelJumpResource{}
				r.ApiClient = mongodbClient(t, body, nil)
				m := mongodbModel(types.StringValue(""))
				m.ID = types.StringUnknown()
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
				r.Create(ctx, resource.CreateRequest{Plan: mongodbPlan(t, sch, m)}, resp)
				require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
				check(t, resp.State)
			})
			t.Run("read", func(t *testing.T) {
				r := &mongodbTunnelJumpResource{}
				r.ApiClient = mongodbClient(t, body, nil)
				state := mongodbState(t, sch, mongodbModel(types.StringValue("")))
				resp := &resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, resp)
				require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
				check(t, resp.State)
			})
			t.Run("update", func(t *testing.T) {
				r := &mongodbTunnelJumpResource{}
				r.ApiClient = mongodbClient(t, body, nil)
				m := mongodbModel(types.StringValue(""))
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: sch}}
				r.Update(ctx, resource.UpdateRequest{State: mongodbState(t, sch, m), Plan: mongodbPlan(t, sch, m)}, resp)
				require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
				check(t, resp.State)
			})
		})
	}
}

// A Read that finds the item gone removes it from state; normalising must not
// write a half-populated object back.
func TestMongodbReadOfDeletedItemStaysRemoved(t *testing.T) {
	ctx := context.Background()
	sch := mongodbSchema(t)
	r := &mongodbTunnelJumpResource{}
	r.ApiClient = mockGPClient(t, func(w http.ResponseWriter, _ *http.Request) bool {
		w.WriteHeader(http.StatusNotFound)
		return true
	})
	state := mongodbState(t, sch, mongodbModel(types.StringValue("")))
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	assert.True(t, resp.State.Raw.IsNull())
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
	sch := mongodbSchema(t)
	for value, ok := range map[string]bool{
		"127.0.0.1":        true,
		"127.0.0.5":        true,
		"127.0.0.255":      true,
		"127.0.1.1":        false,
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
}

func TestMongodbJumpGroupTypeValidator(t *testing.T) {
	sch := mongodbSchema(t)
	assert.True(t, runStringValidators(t, sch, "jump_group_type", types.StringValue("shared")))
	assert.True(t, runStringValidators(t, sch, "jump_group_type", types.StringValue("personal")))
	assert.False(t, runStringValidators(t, sch, "jump_group_type", types.StringValue("public")))
}
