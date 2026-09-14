package rs

import (
	"context"
	"net/http"
	"strings"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"
	"testing"

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
