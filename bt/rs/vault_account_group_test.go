package rs

import (
	"context"
	"net/http"
	"strings"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// vagMockClient stands up an httptest server for the vault-account-group
// resource: the main GET returns a valid account group, and the jump-item-
// association sub-resource GET returns jiaStatus, letting a test simulate a
// backend error on that read without touching the primary resource read.
func vagMockClient(t *testing.T, jiaStatus int) *api.APIClient {
	t.Helper()
	return mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			w.WriteHeader(jiaStatus)
			return true
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":5,"name":"grp","description":""}`))
		return true
	})
}

// S1 regression: GetItemEndpoint returns (nil, err) on every error path
// (api/crud.go), so a hoisted error check must fire even though `item` is
// nil. Before the fix, the check was nested inside `if item != nil`, so a 500
// reading the account-group jump item association was silently dropped —
// no diagnostic, and state left exactly as it was going in.
func TestVaultAccountGroupRead_JIAErrorSurfaces(t *testing.T) {
	ctx := context.Background()

	r := &vaultAccountGroupResource{}
	r.ApiClient = vagMockClient(t, http.StatusInternalServerError)

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema

	jiaType, ok := sch.Attributes["jump_item_association"].GetType().(types.ObjectType)
	assert.True(t, ok)
	gpmType, ok := sch.Attributes["group_policy_memberships"].GetType().(types.SetType)
	assert.True(t, ok)

	model := models.VaultAccountGroup{
		ID:                     types.StringValue("5"),
		Name:                   types.StringValue("grp"),
		Description:            types.StringValue(""),
		AccountPolicy:          types.StringNull(),
		JumpItemAssociation:    types.ObjectNull(jiaType.AttrTypes),
		GroupPolicyMemberships: types.SetNull(gpmType.ElemType),
	}

	state := tfsdk.State{Schema: sch}
	diags := state.Set(ctx, &model)
	assert.False(t, diags.HasError(), "%v", diags)

	req := resource.ReadRequest{State: state}
	// Mirrors the framework: ReadResponse.State starts as a copy of the
	// incoming state and the resource updates it in place.
	resp := &resource.ReadResponse{State: state}

	r.Read(ctx, req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "a 500 reading the jump item association must surface a diagnostic")

	var after models.VaultAccountGroup
	getDiags := resp.State.Get(ctx, &after)
	assert.False(t, getDiags.HasError(), "%v", getDiags)
	assert.True(t, after.JumpItemAssociation.IsNull(), "state must be left untouched when the read errors")
}

// Finding 1 regression: the account-group jump-item-association endpoint
// documents only GET and PATCH (PRA openapi/bt-pra-configuration.openapi.yaml:
// 5286,5299; RS openapi/bt-rs-configuration.openapi.yaml:4201,4214) — never
// POST. Before the fix, Update chose POST whenever state's jump_item_association
// was null, which is exactly the state left by `terraform import` (only `id` is
// populated) once the attribute's static default fills a non-null plan value.
func TestVaultAccountGroupUpdate_JIAUsesPatchNeverPost(t *testing.T) {
	ctx := context.Background()

	var jiaMethod string
	var postToJIA bool
	r := &vaultAccountGroupResource{}
	r.ApiClient = mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			jiaMethod = r.Method
			if r.Method == http.MethodPost {
				postToJIA = true
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"filter_type":"any_jump_items","criteria":null,"jump_items":[]}`))
			return true
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":5,"name":"grp","description":""}`))
		return true
	})

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema

	jiaType, ok := sch.Attributes["jump_item_association"].GetType().(types.ObjectType)
	assert.True(t, ok)
	gpmType, ok := sch.Attributes["group_policy_memberships"].GetType().(types.SetType)
	assert.True(t, ok)
	criteriaType, ok := jiaType.AttrTypes["criteria"].(types.ObjectType)
	assert.True(t, ok)
	jumpItemsType, ok := jiaType.AttrTypes["jump_items"].(types.SetType)
	assert.True(t, ok)

	jia := types.ObjectValueMust(jiaType.AttrTypes, map[string]attr.Value{
		"filter_type": types.StringValue("any_jump_items"),
		"criteria":    types.ObjectNull(criteriaType.AttrTypes),
		"jump_items":  types.SetValueMust(jumpItemsType.ElemType, []attr.Value{}),
	})

	planModel := models.VaultAccountGroup{
		ID:                     types.StringValue("5"),
		Name:                   types.StringValue("grp"),
		Description:            types.StringValue(""),
		AccountPolicy:          types.StringNull(),
		JumpItemAssociation:    jia,
		GroupPolicyMemberships: types.SetNull(gpmType.ElemType),
	}
	// Mirrors a fresh `terraform import`: only `id` was populated by
	// ImportStatePassthroughID, so jump_item_association is null in state
	// while the attribute's static default fills a non-null plan value.
	stateModel := planModel
	stateModel.JumpItemAssociation = types.ObjectNull(jiaType.AttrTypes)

	plan := tfsdk.Plan{Schema: sch}
	diags := plan.Set(ctx, &planModel)
	assert.False(t, diags.HasError(), "%v", diags)

	state := tfsdk.State{Schema: sch}
	diags = state.Set(ctx, &stateModel)
	assert.False(t, diags.HasError(), "%v", diags)

	req := resource.UpdateRequest{Plan: plan, State: state}
	// Mirrors the framework: UpdateResponse.State starts as a copy of the
	// incoming state and the resource updates it in place.
	resp := &resource.UpdateResponse{State: state}

	r.Update(ctx, req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	assert.Equal(t, http.MethodPatch, jiaMethod, "Update must PATCH the account-group jump-item-association endpoint")
	assert.False(t, postToJIA, "Update must never POST to /jump-item-association — the endpoint documents no such verb")
}
