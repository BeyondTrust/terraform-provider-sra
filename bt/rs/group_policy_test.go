package rs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupPolicySchemaMatchesTerraformModel(t *testing.T) {
	var resp resource.SchemaResponse
	(&groupPolicyResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())

	expected := make(map[string]struct{})
	modelType := reflect.TypeOf(models.GroupPolicy{})
	for i := 0; i < modelType.NumField(); i++ {
		expected[modelType.Field(i).Tag.Get("tfsdk")] = struct{}{}
	}

	actual := make(map[string]struct{}, len(resp.Schema.Attributes))
	for name := range resp.Schema.Attributes {
		actual[name] = struct{}{}
	}

	assert.Equal(t, expected, actual)
	assert.Contains(t, actual, "perm_sd_static_port_for_external_tools")
}

func TestGroupPolicyResourceIsRegistered(t *testing.T) {
	for _, factory := range ResourceList() {
		var metadata resource.MetadataResponse
		factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "sra"}, &metadata)
		if metadata.TypeName == "sra_group_policy" {
			return
		}
	}

	t.Fatal("sra_group_policy was not registered in ResourceList")
}

func TestConfiguredGroupPolicyFieldsForOtherProduct(t *testing.T) {
	config := models.GroupPolicy{
		PermAccessAllowed:     types.BoolValue(true),
		PermSupportAllowed:    types.StringValue("full_support"),
		PermInviteExternalRep: types.BoolNull(),
	}

	praFields := configuredGroupPolicyFieldsForOtherProduct(config, api.ProductPRA)
	require.Len(t, praFields, 1)
	assert.Equal(t, "perm_support_allowed", praFields[0].name)
	assert.Equal(t, "rs", praFields[0].product)

	rsFields := configuredGroupPolicyFieldsForOtherProduct(config, api.ProductRS)
	require.Len(t, rsFields, 1)
	assert.Equal(t, "perm_access_allowed", rsFields[0].name)
	assert.Equal(t, "pra", rsFields[0].product)
}

func TestGroupPolicyResourceCRUDWithPRAFields(t *testing.T) {
	ctx := context.Background()
	var stored map[string]any
	var methods []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"test-token"}`))
			return
		}

		methods = append(methods, req.Method)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case http.MethodPost, http.MethodPatch:
			body, err := io.ReadAll(req.Body)
			if !assert.NoError(t, err) {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !assert.NoError(t, json.Unmarshal(body, &stored)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored["id"] = float64(42)
			assert.True(t, strings.HasPrefix(req.URL.Path, "/api/config/v1/group-policy"))
			assert.NotContains(t, stored, "perm_support_allowed")
			assert.Equal(t, true, stored["perm_sd_static_port_for_external_tools"])
			assert.NoError(t, json.NewEncoder(w).Encode(stored))
		case http.MethodGet:
			assert.Equal(t, "/api/config/v1/group-policy/42", req.URL.Path)
			assert.NoError(t, json.NewEncoder(w).Encode(stored))
		case http.MethodDelete:
			assert.Equal(t, "/api/config/v1/group-policy/42", req.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			assert.Failf(t, "unexpected request", "%s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)

	clientID, clientSecret := "id", "secret"
	client, err := api.NewClient(server.URL, &clientID, &clientSecret)
	require.NoError(t, err)
	client.Product = api.ProductPRA

	managed := &groupPolicyResource{}
	managed.ApiClient = client
	sch := groupPolicyTestSchema(t, managed)

	createModel := praGroupPolicyTestModel(types.StringUnknown(), "Managed Group Policy")
	createPlan := groupPolicyTestPlan(t, sch, createModel)
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: createPlan.Raw}}
	managed.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "%v", createResp.Diagnostics)

	var created models.GroupPolicy
	require.False(t, createResp.State.Get(ctx, &created).HasError())
	assert.Equal(t, "42", created.ID.ValueString())
	assert.True(t, created.PermSdStaticPortForExternalTools.ValueBool())
	assert.True(t, created.PermSupportAllowed.IsNull())

	readResp := resource.ReadResponse{State: createResp.State}
	managed.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)

	updateModel := praGroupPolicyTestModel(types.StringValue("42"), "Updated Group Policy")
	updateModel.PermRemoteRdp = types.BoolValue(false)
	updatePlan := groupPolicyTestPlan(t, sch, updateModel)
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: sch, Raw: updatePlan.Raw}}
	managed.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, &updateResp)
	require.False(t, updateResp.Diagnostics.HasError(), "%v", updateResp.Diagnostics)

	var updated models.GroupPolicy
	require.False(t, updateResp.State.Get(ctx, &updated).HasError())
	assert.Equal(t, "Updated Group Policy", updated.Name.ValueString())
	assert.False(t, updated.PermRemoteRdp.ValueBool())

	deleteResp := resource.DeleteResponse{}
	managed.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	require.False(t, deleteResp.Diagnostics.HasError(), "%v", deleteResp.Diagnostics)

	assert.Equal(t, []string{http.MethodPost, http.MethodGet, http.MethodPatch, http.MethodDelete}, methods)
}

func TestGroupPolicyImportRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{"SMS_Group_Policy", "0", "-1", "2147483648"} {
		t.Run(id, func(t *testing.T) {
			managed := &groupPolicyResource{}
			var resp resource.ImportStateResponse
			managed.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)

			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "numeric ID from 1")
		})
	}
}

func groupPolicyTestSchema(t *testing.T, managed *groupPolicyResource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	managed.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

func groupPolicyTestPlan(t *testing.T, sch schema.Schema, model models.GroupPolicy) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	var object types.Object
	diags := tfsdk.ValueFrom(ctx, model, sch.Type(), &object)
	require.False(t, diags.HasError(), "%v", diags)
	raw, err := object.ToTerraformValue(ctx)
	require.NoError(t, err)
	return tfsdk.Plan{Schema: sch, Raw: raw}
}

func praGroupPolicyTestModel(id types.String, name string) models.GroupPolicy {
	return models.GroupPolicy{
		ID:                                  id,
		Name:                                types.StringValue(name),
		PermShareOtherTeam:                  types.BoolValue(false),
		PermSessionIdleTimeout:              types.Int64Value(-1),
		PermExtendedAvailabilityModeAllowed: types.BoolValue(false),
		PermEditExternalKey:                 types.BoolValue(false),
		PermCollaborate:                     types.BoolValue(false),
		PermCollaborateControl:              types.BoolValue(false),
		PermJumpClient:                      types.BoolValue(true),
		PermLocalJump:                       types.BoolValue(false),
		PermRemoteJump:                      types.BoolValue(false),
		PermRemoteVnc:                       types.BoolValue(false),
		PermRemoteRdp:                       types.BoolValue(true),
		PermShellJump:                       types.BoolValue(false),
		DefaultJumpItemRoleID:               types.Int64Value(1),
		PrivateJumpItemRoleID:               types.Int64Value(1),
		InferiorJumpItemRoleID:              types.Int64Value(1),
		UnassignedJumpItemRoleID:            types.Int64Value(1),
		PermAccessAllowed:                   types.BoolValue(true),
		AccessPermStatus:                    types.StringValue("defined"),
		PermInviteExternalUser:              types.BoolValue(false),
		PermWebJump:                         types.BoolValue(false),
		PermProtocolTunnel:                  types.BoolValue(false),
		PermSdStaticPortForExternalTools:    types.BoolValue(true),
		PermSupportAllowed:                  types.StringNull(),
		RepPermStatus:                       types.StringNull(),
		PermGenerateSessionKey:              types.BoolNull(),
		PermSendIosProfiles:                 types.BoolNull(),
		PermAcceptTeamSessions:              types.BoolNull(),
		PermTransferOtherTeam:               types.BoolNull(),
		PermInviteExternalRep:               types.BoolNull(),
		PermNextSessionButton:               types.BoolNull(),
		PermDisableAutoAssignment:           types.BoolNull(),
		PermRoutingIdleTimeout:              types.Int64Null(),
		AutoAssignmentMaxSessions:           types.Int64Null(),
		PermSupportButtonPersonalDeploy:     types.BoolNull(),
		PermSupportButtonTeamManage:         types.BoolNull(),
		PermSupportButtonChangePublicSites:  types.BoolNull(),
		PermSupportButtonTeamDeploy:         types.BoolNull(),
		PermLocalVNC:                        types.BoolNull(),
		PermLocalRDP:                        types.BoolNull(),
		PermVpro:                            types.BoolNull(),
		PermConsoleIdleTimeout:              types.Int64Null(),
	}
}

func TestGroupPolicyImportSetsNumericID(t *testing.T) {
	managed := &groupPolicyResource{}
	managed.ApiClient = &api.APIClient{Product: api.ProductPRA}
	sch := groupPolicyTestSchema(t, managed)
	initial := praGroupPolicyTestModel(types.StringNull(), "Imported Group Policy")
	plan := groupPolicyTestPlan(t, sch, initial)
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: sch, Raw: plan.Raw}}

	managed.ImportState(context.Background(), resource.ImportStateRequest{ID: "9"}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var id types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("id"), &id).HasError())
	assert.Equal(t, "9", id.ValueString())
}
