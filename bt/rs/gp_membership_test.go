package rs

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// --- test scaffolding for the group-policy membership helpers ---------------

// A minimal schema carrying just the group_policy_memberships attribute (the
// jump_group shape). It is enough for GetAttribute/SetAttribute on that path.
func gpTestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id":   schema.StringAttribute{Required: true},
						"jump_item_role_id": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(0)},
						"jump_policy_id":    schema.Int64Attribute{Optional: true, Computed: true},
					},
				},
			},
		},
	}
}

var (
	gpMemberObjType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"group_policy_id":   tftypes.String,
		"jump_item_role_id": tftypes.Number,
		"jump_policy_id":    tftypes.Number,
	}}
	gpSetType    = tftypes.Set{ElementType: gpMemberObjType}
	gpSchemaType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"group_policy_memberships": gpSetType}}
)

func gpMember(gpID string) tftypes.Value {
	return tftypes.NewValue(gpMemberObjType, map[string]tftypes.Value{
		"group_policy_id":   tftypes.NewValue(tftypes.String, gpID),
		"jump_item_role_id": tftypes.NewValue(tftypes.Number, big.NewFloat(0)),
		"jump_policy_id":    tftypes.NewValue(tftypes.Number, nil),
	})
}

// gpRaw builds an object value for the schema. Pass nil members for a null set.
func gpRaw(members []tftypes.Value) tftypes.Value {
	var set tftypes.Value
	if members == nil {
		set = tftypes.NewValue(gpSetType, nil)
	} else {
		set = tftypes.NewValue(gpSetType, members)
	}
	return tftypes.NewValue(gpSchemaType, map[string]tftypes.Value{"group_policy_memberships": set})
}

// gpJumpGroupCallbacks returns the same callbacks jump_group.Update passes.
func gpJumpGroupCallbacks() (func(*api.GroupPolicyJumpGroup, int), func(*api.GroupPolicyJumpGroup) *string, func(*api.GroupPolicyJumpGroup, *string)) {
	return func(m *api.GroupPolicyJumpGroup, entityID int) { m.JumpGroupID = &entityID },
		func(m *api.GroupPolicyJumpGroup) *string { return m.GroupPolicyID },
		func(m *api.GroupPolicyJumpGroup, gpID *string) { m.GroupPolicyID = gpID }
}

// mockGPClient stands up an httptest server that satisfies the OAuth handshake
// plus membership create/delete/provision, and returns a client pointed at it.
// The optional handler lets a test assert on or customize responses.
func mockGPClient(t *testing.T, handler func(w http.ResponseWriter, r *http.Request) bool) *api.APIClient {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"x"}`))
			return
		}
		if handler != nil && handler(w, r) {
			return
		}
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/provision"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jump_group_id":42,"jump_item_role_id":0}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(ts.Close)

	id, secret := "id", "secret"
	c, err := api.NewClient(ts.URL, &id, &secret)
	assert.NoError(t, err)
	c.SetTestLogger(t)
	return c
}

// --- tests ------------------------------------------------------------------

// Regression guard for I1: removing every membership (plan attribute null,
// state populated) must leave the attribute NULL in state, not an empty set —
// otherwise Terraform reports "inconsistent result after apply" on the
// Optional/non-Computed jump_group & jumpoint resources.
func TestUpdateGPMemberships_RemoveAllWritesNull(t *testing.T) {
	ctx := context.Background()

	var deleted int
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodDelete {
			deleted++
			w.WriteHeader(http.StatusOK)
			return true
		}
		return false
	})

	sch := gpTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: gpRaw(nil)}                              // membership block removed
	state := tfsdk.State{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7")})} // one existing membership
	respState := tfsdk.State{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7")})}
	var diags diag.Diagnostics

	setEntityID, getGP, setGP := gpJumpGroupCallbacks()
	UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, state, &respState, &diags, 42,
		setEntityID, getGP, setGP, api.DiffGPJumpItemLists, &sync.Mutex{})

	assert.False(t, diags.HasError(), "update should succeed")
	assert.Equal(t, 1, deleted, "the stale membership should be deleted")

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.True(t, out.IsNull(), "removing all memberships must write a null set, not an empty one")
}

// A no-op update (plan == state) must delete nothing and preserve the membership.
func TestUpdateGPMemberships_NoChangePreservesState(t *testing.T) {
	ctx := context.Background()

	var deleted, created int
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodDelete {
			deleted++
		}
		if r.Method == http.MethodPost && !strings.HasSuffix(r.URL.Path, "/provision") {
			created++
		}
		return false
	})

	sch := gpTestSchema()
	members := []tftypes.Value{gpMember("7")}
	plan := tfsdk.Plan{Schema: sch, Raw: gpRaw(members)}
	state := tfsdk.State{Schema: sch, Raw: gpRaw(members)}
	respState := tfsdk.State{Schema: sch, Raw: gpRaw(members)}
	var diags diag.Diagnostics

	setEntityID, getGP, setGP := gpJumpGroupCallbacks()
	UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, state, &respState, &diags, 42,
		setEntityID, getGP, setGP, api.DiffGPJumpItemLists, &sync.Mutex{})

	assert.False(t, diags.HasError())
	assert.Equal(t, 0, deleted, "no membership should be deleted on a no-op update")
	assert.Equal(t, 0, created, "no membership should be created on a no-op update")

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())
	assert.Equal(t, 1, len(out.Elements()))
}

// CreateGPMemberships writes the created memberships (with the plan's group
// policy ID re-applied) to state.
func TestCreateGPMemberships_WritesResults(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, nil)

	sch := gpTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7")})}
	respState := tfsdk.State{Schema: sch, Raw: gpRaw(nil)}
	var diags diag.Diagnostics

	setEntityID, getGP, setGP := gpJumpGroupCallbacks()
	CreateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, &respState, &diags, 42,
		setEntityID, getGP, setGP, &sync.Mutex{})

	assert.False(t, diags.HasError())

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())
	assert.Equal(t, 1, len(out.Elements()))
}
