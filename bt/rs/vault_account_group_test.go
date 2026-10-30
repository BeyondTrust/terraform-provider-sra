package rs

import (
	"context"
	"net/http"
	"strings"
	"terraform-provider-sra/api"
	"terraform-provider-sra/api/apimock"
	"terraform-provider-sra/bt/models"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// readAccountGroup runs Read for account group 5 against srv, from a state whose
// jump_item_association is jia (nil for null, as an import leaves it).
func readAccountGroup(t *testing.T, srv *apimock.Server, jia *api.AccountGroupJumpItemAssociation) *resource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	srv.Rows("vault/account-group", apimock.Row{"id": 5, "name": "grp", "description": ""})

	r := &vaultAccountGroupResource{}
	r.ApiClient = apimockClient(srv)
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema
	jiaType := sch.Attributes["jump_item_association"].GetType().(types.ObjectType)
	gpmType := sch.Attributes["group_policy_memberships"].GetType().(types.SetType)

	state := tfsdk.State{Schema: sch}
	require.False(t, state.Set(ctx, &models.VaultAccountGroup{
		ID:                     types.StringValue("5"),
		Name:                   types.StringValue("grp"),
		Description:            types.StringValue(""),
		AccountPolicy:          types.StringNull(),
		JumpItemAssociation:    types.ObjectNull(jiaType.AttrTypes),
		GroupPolicyMemberships: types.SetNull(gpmType.ElemType),
	}).HasError())
	if jia != nil {
		require.False(t, state.SetAttribute(ctx, path.Root("jump_item_association"), jia).HasError())
	}

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	return resp
}

// After an import state holds only the id. Read used to skip writing the
// association whenever state held none, so an import never planned clean.
func TestVaultAccountGroupRead_NullJIAStateTakesTheAPIValue(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	srv.Handle(http.MethodGet, "vault/account-group/5/jump-item-association", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"filter_type":"criteria","criteria":{"shared_jump_groups":[3],"host":[],"name":[],"tag":["db"],"comment":[]},"jump_items":[{"id":7,"type":"shell_jump"}]}`))
	})

	resp := readAccountGroup(t, srv, nil)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var got api.AccountGroupJumpItemAssociation
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("jump_item_association"), &got).HasError())
	assert.Equal(t, "criteria", got.FilterType)
	require.NotNil(t, got.Criteria)
	assert.Equal(t, []int{3}, got.Criteria.SharedJumpGroups)
	assert.Equal(t, []string{"db"}, got.Criteria.Tag)
	assert.Equal(t, []api.InjectableJumpItem{{ID: 7, Type: "shell_jump"}}, got.JumpItems)
}

// An imported association narrower than any_jump_items is reset to it by a
// configuration that omits the block, widening where the credentials can be
// injected. The Read after an import (state null) says so; an any_jump_items
// association, or one already in state, gets no warning.
func TestVaultAccountGroupRead_WarnsWhenOmittingAnImportedJIAWouldWidenIt(t *testing.T) {
	t.Parallel()

	inState := &api.AccountGroupJumpItemAssociation{FilterType: "no_jump_items", JumpItems: []api.InjectableJumpItem{}}
	for name, tc := range map[string]struct {
		filterType string
		state      *api.AccountGroupJumpItemAssociation
		warned     bool
	}{
		"any_jump_items":   {"any_jump_items", nil, false},
		"no_jump_items":    {"no_jump_items", nil, true},
		"criteria":         {"criteria", nil, true},
		"already in state": {"no_jump_items", inState, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := apimock.New()
			defer srv.Close()
			srv.Handle(http.MethodGet, "vault/account-group/5/jump-item-association", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"filter_type":"` + tc.filterType + `","criteria":null,"jump_items":[]}`))
			})

			resp := readAccountGroup(t, srv, tc.state)
			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
			if tc.warned {
				require.Len(t, resp.Diagnostics.Warnings(), 1)
				assert.Contains(t, resp.Diagnostics.Warnings()[0].Detail(), "resets it to any_jump_items")
			} else {
				assert.Empty(t, resp.Diagnostics.Warnings())
			}
		})
	}
}
