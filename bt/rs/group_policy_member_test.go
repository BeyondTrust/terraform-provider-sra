package rs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupPolicyMemberSchemaMatchesTerraformModel(t *testing.T) {
	managed := &groupPolicyMemberResource{}
	sch := groupPolicyMemberTestSchema(t, managed)

	expected := make(map[string]struct{})
	modelType := reflect.TypeOf(models.GroupPolicyMember{})
	for i := 0; i < modelType.NumField(); i++ {
		expected[modelType.Field(i).Tag.Get("tfsdk")] = struct{}{}
	}

	actual := make(map[string]struct{}, len(sch.Attributes))
	for name := range sch.Attributes {
		actual[name] = struct{}{}
	}
	assert.Equal(t, expected, actual)
	assert.Len(t, managed.ConfigValidators(context.Background()), 1)
}

func TestGroupPolicyMemberResourceIsRegistered(t *testing.T) {
	for _, factory := range ResourceList() {
		var metadata resource.MetadataResponse
		factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "sra"}, &metadata)
		if metadata.TypeName == "sra_group_policy_member" {
			return
		}
	}

	t.Fatal("sra_group_policy_member was not registered in ResourceList")
}

func TestGroupPolicyMemberResourceCRUDWithEmptyCreateResponse(t *testing.T) {
	ctx := context.Background()
	created := false
	provisionCount := 0
	requests := make([]string, 0, 7)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"test-token"}`))
			return
		}

		requests = append(requests, req.Method+" "+req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/api/config/v1/group-policy/9/member":
			assert.Equal(t, "100", req.URL.Query().Get("per_page"))
			assert.Equal(t, "1", req.URL.Query().Get("current_page"))
			if created {
				_, _ = w.Write([]byte(`[{"id":77,"security_provider_id":5,"group_name":"Suppliers"}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case req.Method == http.MethodPost && req.URL.Path == "/api/config/v1/group-policy/9/member":
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, float64(5), payload["security_provider_id"])
			assert.Equal(t, "Suppliers", payload["group_name"])
			assert.NotContains(t, payload, "group_policy_id")
			assert.NotContains(t, payload, "id")
			assert.NotContains(t, payload, "user_id")
			assert.NotContains(t, payload, "distinguished_name")
			created = true
			w.WriteHeader(http.StatusCreated)
		case req.Method == http.MethodPost && req.URL.Path == "/api/config/v1/group-policy/9/provision":
			provisionCount++
			w.WriteHeader(http.StatusNoContent)
		case req.Method == http.MethodGet && req.URL.Path == "/api/config/v1/group-policy/9/member/77":
			if !created {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"id":77,"security_provider_id":5,"group_name":"Suppliers"}`))
		case req.Method == http.MethodDelete && req.URL.Path == "/api/config/v1/group-policy/9/member/77":
			created = false
			w.WriteHeader(http.StatusNoContent)
		default:
			assert.Failf(t, "unexpected request", "%s %s", req.Method, req.URL.String())
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)

	clientID, clientSecret := "id", "secret"
	client, err := api.NewClient(server.URL, &clientID, &clientSecret)
	require.NoError(t, err)
	client.Product = api.ProductPRA

	managed := &groupPolicyMemberResource{}
	managed.ApiClient = client
	sch := groupPolicyMemberTestSchema(t, managed)
	plan := groupPolicyMemberTestPlan(t, sch, models.GroupPolicyMember{
		ID:                 types.StringUnknown(),
		GroupPolicyID:      types.StringValue("9"),
		SecurityProviderID: types.Int64Value(5),
		DistinguishedName:  types.StringNull(),
		GroupName:          types.StringValue("Suppliers"),
		UserID:             types.Int64Null(),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: plan.Raw}}
	managed.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "%v", createResp.Diagnostics)

	var createdState models.GroupPolicyMember
	require.False(t, createResp.State.Get(ctx, &createdState).HasError())
	assert.Equal(t, "77", createdState.ID.ValueString())
	assert.Equal(t, "Suppliers", createdState.GroupName.ValueString())
	assert.Equal(t, 1, provisionCount)

	readResp := resource.ReadResponse{State: createResp.State}
	managed.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)

	deleteResp := resource.DeleteResponse{}
	managed.Delete(ctx, resource.DeleteRequest{State: readResp.State}, &deleteResp)
	require.False(t, deleteResp.Diagnostics.HasError(), "%v", deleteResp.Diagnostics)
	assert.False(t, created)
	assert.Equal(t, 2, provisionCount)
	assert.Equal(t, []string{
		"GET /api/config/v1/group-policy/9/member",
		"POST /api/config/v1/group-policy/9/member",
		"GET /api/config/v1/group-policy/9/member",
		"POST /api/config/v1/group-policy/9/provision",
		"GET /api/config/v1/group-policy/9/member/77",
		"DELETE /api/config/v1/group-policy/9/member/77",
		"POST /api/config/v1/group-policy/9/provision",
	}, requests)
}

func TestGroupPolicyMemberImportState(t *testing.T) {
	managed := &groupPolicyMemberResource{}
	sch := groupPolicyMemberTestSchema(t, managed)
	initial := groupPolicyMemberTestPlan(t, sch, models.GroupPolicyMember{
		ID:                 types.StringUnknown(),
		GroupPolicyID:      types.StringValue("1"),
		SecurityProviderID: types.Int64Value(1),
		DistinguishedName:  types.StringNull(),
		GroupName:          types.StringNull(),
		UserID:             types.Int64Value(1),
	})
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: sch, Raw: initial.Raw}}

	managed.ImportState(context.Background(), resource.ImportStateRequest{ID: "9/77"}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state models.GroupPolicyMember
	require.False(t, resp.State.Get(context.Background(), &state).HasError())
	assert.Equal(t, "9", state.GroupPolicyID.ValueString())
	assert.Equal(t, "77", state.ID.ValueString())
}

func TestGroupPolicyMemberImportRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{"77", "9/0", "0/77", "9/name", "9/77/1", "2147483648/1"} {
		t.Run(id, func(t *testing.T) {
			managed := &groupPolicyMemberResource{}
			var resp resource.ImportStateResponse
			managed.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
			require.True(t, resp.Diagnostics.HasError())
		})
	}
}

func TestSetGroupPolicyMemberSelectorFromAPIPrefersStableUserID(t *testing.T) {
	userID := 42
	dn := "CN=Supplier,OU=Users,DC=example,DC=com"
	state := models.GroupPolicyMember{}

	ok := setGroupPolicyMemberSelectorFromAPI(&state, api.GroupPolicyMember{UserID: &userID, DistinguishedName: &dn})
	require.True(t, ok)
	assert.Equal(t, int64(42), state.UserID.ValueInt64())
	assert.True(t, state.DistinguishedName.IsNull())
	assert.True(t, state.GroupName.IsNull())
}

func groupPolicyMemberTestSchema(t *testing.T, managed *groupPolicyMemberResource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	managed.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

func groupPolicyMemberTestPlan(t *testing.T, sch schema.Schema, model models.GroupPolicyMember) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	var object types.Object
	diags := tfsdk.ValueFrom(ctx, model, sch.Type(), &object)
	require.False(t, diags.HasError(), "%v", diags)
	raw, err := object.ToTerraformValue(ctx)
	require.NoError(t, err)
	return tfsdk.Plan{Schema: sch, Raw: raw}
}
