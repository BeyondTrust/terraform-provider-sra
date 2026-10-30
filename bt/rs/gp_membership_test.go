package rs

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"terraform-provider-sra/api"
	"terraform-provider-sra/api/apimock"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return gpMemberWithRole(gpID, 0)
}

// gpMemberWithRole is gpMember with an explicit jump_item_role_id, so a test
// can tell "the planned value was echoed" apart from "a zero-value fallback
// happened to also read 0".
func gpMemberWithRole(gpID string, roleID int64) tftypes.Value {
	return tftypes.NewValue(gpMemberObjType, map[string]tftypes.Value{
		"group_policy_id":   tftypes.NewValue(tftypes.String, gpID),
		"jump_item_role_id": tftypes.NewValue(tftypes.Number, big.NewFloat(float64(roleID))),
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

	UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, state, &respState, &diags, 42,
		api.DiffGPJumpItemLists, &sync.Mutex{})

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

	UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, state, &respState, &diags, 42,
		api.DiffGPJumpItemLists, &sync.Mutex{})

	assert.False(t, diags.HasError())
	assert.Equal(t, 0, deleted, "no membership should be deleted on a no-op update")
	assert.Equal(t, 0, created, "no membership should be created on a no-op update")

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())
	assert.Equal(t, 1, len(out.Elements()))
}

// UpdateGPMemberships writes the created memberships (with the plan's group
// policy ID re-applied) to state via the toAdd loop's with-body path.
//
// This must NOT be folded into TestUpdateGPMemberships_204NoContentTolerated:
// on a 204, createdOrSent(nil, m) returns the plan value m, which already
// carries GroupPolicyID, so re-applying it there is a no-op that cannot catch
// a broken WithGroupPolicyID. Here the mock returns a body, so
// api.GroupPolicyJumpGroup's `json:"-"` GroupPolicyID arrives nil from the
// API and WithGroupPolicyID is the only thing that puts it back.
func TestUpdateGPMemberships_WritesResults(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, nil)

	sch := gpTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7")})}
	state := tfsdk.State{Schema: sch, Raw: gpRaw(nil)}
	respState := tfsdk.State{Schema: sch, Raw: gpRaw(nil)}
	var diags diag.Diagnostics

	UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, state, &respState, &diags, 42,
		api.DiffGPJumpItemLists, &sync.Mutex{})

	assert.False(t, diags.HasError())

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())

	var stored []api.GroupPolicyJumpGroup
	assert.False(t, out.ElementsAs(ctx, &stored, false).HasError())
	if assert.Len(t, stored, 1) {
		if assert.NotNil(t, stored[0].GroupPolicyID, "the plan group policy ID must be re-applied; the response body omits it (json:\"-\")") {
			assert.Equal(t, "7", *stored[0].GroupPolicyID)
		}
	}
}

// ReadGPMemberships decodes the array the membership endpoint returns, keeps
// the memberships the API still reports, and drops ones it no longer does.
func TestReadGPMemberships_RefreshesAndDropsRemoved(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/jump-group/") {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "group-policy/7/") {
				// gp 7 still has the membership (scoped 1-element array)
				_, _ = w.Write([]byte(`[{"jump_group_id":42,"jump_item_role_id":3,"jump_policy_id":0}]`))
			} else {
				// gp 8: no longer a member
				_, _ = w.Write([]byte(`[]`))
			}
			return true
		}
		return false
	})

	sch := gpTestSchema()
	state := tfsdk.State{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7"), gpMember("8")})}
	respState := tfsdk.State{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7"), gpMember("8")})}
	var diags diag.Diagnostics

	ReadGPMemberships[api.GroupPolicyJumpGroup](ctx, client, state, &respState, &diags, 42)

	assert.False(t, diags.HasError())

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())
	assert.Equal(t, 1, len(out.Elements()), "gp 7 is kept; gp 8 (reported absent) is dropped")

	var refreshed []api.GroupPolicyJumpGroup
	assert.False(t, out.ElementsAs(ctx, &refreshed, false).HasError())
	if assert.Len(t, refreshed, 1) {
		assert.Equal(t, "7", *refreshed[0].GroupPolicyID)
		assert.Equal(t, 3, refreshed[0].JumpItemRoleID, "the refreshed jump_item_role_id (3) must land in state, not state's original (0)")
	}
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

	CreateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, &respState, &diags, 42, &sync.Mutex{})

	assert.False(t, diags.HasError())

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())

	var stored []api.GroupPolicyJumpGroup
	assert.False(t, out.ElementsAs(ctx, &stored, false).HasError())
	if assert.Len(t, stored, 1) {
		if assert.NotNil(t, stored[0].GroupPolicyID, "the plan group policy ID must be re-applied; the response body omits it (json:\"-\")") {
			assert.Equal(t, "7", *stored[0].GroupPolicyID)
		}
	}
}

// A1: CreateItem returns (nil, nil) on a 204 No Content. The membership was
// still created, so it must not be dropped from results (which would leave a
// live group-policy entitlement invisible to Terraform).
func TestCreateGPMemberships_204NoContentTolerated(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost && !strings.HasSuffix(r.URL.Path, "/provision") {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		return false
	})

	sch := gpTestSchema()
	// jump_item_role_id is planned as 5 (never 0) so this test can tell "the
	// planned value was echoed" apart from "a zero-value T fallback happened
	// to read back 0".
	plan := tfsdk.Plan{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMemberWithRole("7", 5)})}
	respState := tfsdk.State{Schema: sch, Raw: gpRaw(nil)}
	var diags diag.Diagnostics

	assert.NotPanics(t, func() {
		CreateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, &respState, &diags, 42, &sync.Mutex{})
	})

	assert.False(t, diags.HasError())

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())

	// JumpGroupID is `tfsdk:"-"` (the entity linkage, not part of Terraform
	// state) so ElementsAs never populates it — assert only the fields the
	// schema actually round-trips.
	var stored []api.GroupPolicyJumpGroup
	assert.False(t, out.ElementsAs(ctx, &stored, false).HasError())
	if assert.Len(t, stored, 1, "a 204 must still leave the membership present, not dropped") {
		assert.Equal(t, "7", *stored[0].GroupPolicyID, "the 204'd membership must keep the planned group policy ID")
		assert.Equal(t, 5, stored[0].JumpItemRoleID, "and the planned jump_item_role_id, not a zero-value fallback")
	}
}

// A1: same 204 tolerance, but through UpdateGPMemberships' toAdd loop.
func TestUpdateGPMemberships_204NoContentTolerated(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost && !strings.HasSuffix(r.URL.Path, "/provision") {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		return false
	})

	sch := gpTestSchema()
	// jump_item_role_id is planned as 5 (never 0) so this test can tell "the
	// planned value was echoed" apart from "a zero-value T fallback happened
	// to read back 0".
	plan := tfsdk.Plan{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMemberWithRole("7", 5)})}
	state := tfsdk.State{Schema: sch, Raw: gpRaw(nil)}
	respState := tfsdk.State{Schema: sch, Raw: gpRaw(nil)}
	var diags diag.Diagnostics

	assert.NotPanics(t, func() {
		UpdateGPMemberships[api.GroupPolicyJumpGroup](ctx, client, plan, state, &respState, &diags, 42,
			api.DiffGPJumpItemLists, &sync.Mutex{})
	})

	assert.False(t, diags.HasError())

	var out types.Set
	respState.GetAttribute(ctx, path.Root("group_policy_memberships"), &out)
	assert.False(t, out.IsNull())

	// JumpGroupID is `tfsdk:"-"` (the entity linkage, not part of Terraform
	// state) so ElementsAs never populates it — assert only the fields the
	// schema actually round-trips.
	var stored []api.GroupPolicyJumpGroup
	assert.False(t, out.ElementsAs(ctx, &stored, false).HasError())
	if assert.Len(t, stored, 1, "a 204 must still leave the membership present, not dropped") {
		assert.Equal(t, "7", *stored[0].GroupPolicyID, "the 204'd membership must keep the planned group policy ID")
		assert.Equal(t, 5, stored[0].JumpItemRoleID, "and the planned jump_item_role_id, not a zero-value fallback")
	}
}

// --- import discovery -------------------------------------------------------

func apimockClient(srv *apimock.Server) *api.APIClient {
	return &api.APIClient{RootURL: srv.URL, BaseURL: srv.URL + apimock.BasePath, HTTPClient: srv.Client(), ListRetryBackoff: []time.Duration{0, 0}}
}

// importResource runs r.ImportState for id the way the framework does: the
// response state starts as a null object of the resource's schema.
func importResource(t *testing.T, r resource.ResourceWithImportState, id string) (tfsdk.State, diag.Diagnostics) {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	resp := &resource.ImportStateResponse{State: tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
	return resp.State, resp.Diagnostics
}

func importJumpGroup(t *testing.T, client *api.APIClient, id string) (tfsdk.State, diag.Diagnostics) {
	t.Helper()
	r := &jumpGroupResource{}
	r.ApiClient = client
	return importResource(t, r, id)
}

func jumpGroupMemberships(t *testing.T, state tfsdk.State) (types.Set, []api.GroupPolicyJumpGroup) {
	t.Helper()
	var set types.Set
	require.False(t, state.GetAttribute(context.Background(), path.Root("group_policy_memberships"), &set).HasError())
	var members []api.GroupPolicyJumpGroup
	if !set.IsNull() {
		require.False(t, set.ElementsAs(context.Background(), &members, false).HasError())
	}
	return set, members
}

func gpRows(ids ...int) []apimock.Row {
	rows := make([]apimock.Row, len(ids))
	for i, id := range ids {
		rows[i] = apimock.Row{"id": id, "name": fmt.Sprint("gp ", id)}
	}
	return rows
}

func jgMember(jumpGroupID, roleID int) apimock.Row {
	return apimock.Row{"jump_group_id": jumpGroupID, "jump_item_role_id": roleID, "jump_policy_id": 0}
}

func warningSummaries(diags diag.Diagnostics) []string {
	out := []string{}
	for _, d := range diags.Warnings() {
		out = append(out, d.Summary())
	}
	return out
}

// Two group policies, the imported jump group a member of one: only that one is
// written, with the group policy ID the member index's rows don't carry.
func TestImportGPMemberships_WritesOnlyTheGroupPoliciesItIsIn(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("group-policy", gpRows(1, 2)...)
	srv.Rows("group-policy/1/jump-group", jgMember(9, 3))
	srv.Rows("group-policy/2/jump-group", jgMember(5, 4))

	state, diags := importJumpGroup(t, apimockClient(srv), "5")
	require.False(t, diags.HasError(), "%v", diags)

	_, members := jumpGroupMemberships(t, state)
	require.Len(t, members, 1)
	assert.Equal(t, "2", *members[0].GroupPolicyID)
	assert.Equal(t, 4, members[0].JumpItemRoleID)
	assert.Equal(t, []string{"Imported 1 group policy memberships"}, warningSummaries(diags),
		"the warning comes exactly when memberships are written")
}

// No membership found leaves the attribute null, which is what a configuration
// without the attribute plans. An empty set would plan "[] -> null" forever.
func TestImportGPMemberships_NoneFoundStaysNull(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("group-policy", gpRows(1)...)
	srv.Rows("group-policy/1/jump-group", jgMember(9, 3))

	state, diags := importJumpGroup(t, apimockClient(srv), "5")
	require.False(t, diags.HasError(), "%v", diags)

	set, _ := jumpGroupMemberships(t, state)
	assert.True(t, set.IsNull())
	assert.Empty(t, diags.Warnings(), "no memberships written, no warning")
}

// More than a page of group policies, and more than a page of members in the one
// that holds the imported object.
func TestImportGPMemberships_ReadsEveryPage(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	gps := make([]int, 150)
	for i := range gps {
		gps[i] = i + 1
	}
	srv.Rows("group-policy", gpRows(gps...)...)
	members := make([]apimock.Row, 250)
	for i := range members {
		members[i] = jgMember(1000+i, 1)
	}
	srv.Rows("group-policy/130/jump-group", members...)

	state, diags := importJumpGroup(t, apimockClient(srv), "1240")
	require.False(t, diags.HasError(), "%v", diags)

	_, found := jumpGroupMemberships(t, state)
	require.Len(t, found, 1)
	assert.Equal(t, "130", *found[0].GroupPolicyID)
	assert.Equal(t, 2, srv.Calls(http.MethodGet, "group-policy"))
	assert.Equal(t, 3, srv.Calls(http.MethodGet, "group-policy/130/jump-group"))
}

// A vault-only API account can't list group policies, and can't write
// memberships either: the import succeeds with them unset, and says so.
func TestImportGPMemberships_ForbiddenIsAWarning(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Set("group-policy", &apimock.Collection{Status: http.StatusForbidden})

	state, diags := importJumpGroup(t, apimockClient(srv), "5")
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []string{"group_policy_memberships were not read"}, warningSummaries(diags))
	set, _ := jumpGroupMemberships(t, state)
	assert.True(t, set.IsNull())
}

// Anything else fails the import: a half-discovered set would be acted on by the
// next apply as if it were complete.
func TestImportGPMemberships_OtherErrorsFailTheImport(t *testing.T) {
	t.Parallel()

	t.Run("member index error", func(t *testing.T) {
		t.Parallel()
		srv := apimock.New()
		defer srv.Close()
		srv.Rows("group-policy", gpRows(1, 2)...)
		srv.Rows("group-policy/1/jump-group", jgMember(5, 1))
		srv.Set("group-policy/2/jump-group", &apimock.Collection{Status: http.StatusInternalServerError})

		_, diags := importJumpGroup(t, apimockClient(srv), "5")
		require.True(t, diags.HasError())
		assert.Equal(t, "Error reading group policy memberships", diags.Errors()[0].Summary())
	})

	// Retried, without pausing (see apimockClient), before it gives up.
	t.Run("a list that keeps changing", func(t *testing.T) {
		t.Parallel()
		srv := apimock.New()
		defer srv.Close()
		gps := make([]int, 201)
		for i := range gps {
			gps[i] = i + 1
		}
		srv.Set("group-policy", &apimock.Collection{
			Rows: gpRows(gps...),
			OnPage: func(c *apimock.Collection, page int) {
				if page == 2 {
					c.Rows = c.Rows[1:]
				}
			},
		})

		_, diags := importJumpGroup(t, apimockClient(srv), "5")
		require.True(t, diags.HasError())
		assert.Contains(t, diags.Errors()[0].Detail(), "changed while it was being read")
	})
}

// The index is built once per client and relation: R imports over G group
// policies cost 1 + G list calls, not R×G probes. Every member is still found,
// so a key that collapsed rows could not pass.
func TestImportGPMemberships_IndexIsBuiltOnce(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("group-policy", gpRows(1, 2, 3, 4)...)
	srv.Rows("group-policy/1/jump-group", jgMember(10, 1), jgMember(11, 1))
	srv.Rows("group-policy/2/jump-group", jgMember(10, 2), jgMember(12, 2), jgMember(99, 2))
	srv.Rows("group-policy/3/jump-group", jgMember(11, 3), jgMember(12, 3))
	srv.Rows("group-policy/4/jump-group", jgMember(10, 4), jgMember(11, 4), jgMember(12, 4))
	client := apimockClient(srv)

	want := map[string][]string{"10": {"1", "2", "4"}, "11": {"1", "3", "4"}, "12": {"2", "3", "4"}}
	for id, gps := range want {
		state, diags := importJumpGroup(t, client, id)
		require.False(t, diags.HasError(), "%v", diags)
		_, members := jumpGroupMemberships(t, state)
		got := []string{}
		for _, m := range members {
			got = append(got, *m.GroupPolicyID)
		}
		assert.ElementsMatch(t, gps, got, "jump group %s", id)
	}

	assert.Equal(t, 5, srv.CallsWithMethod(http.MethodGet), "1 group policy list + 4 member lists")
}

// The four membership types share one group policy listing per client.
func TestImportGPMemberships_GroupPolicyListIsShared(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("group-policy", gpRows(1)...)
	srv.Rows("group-policy/1/jump-group", jgMember(5, 1))
	srv.Rows("group-policy/1/jumpoint", apimock.Row{"jumpoint_id": 5})
	client := apimockClient(srv)

	_, diags := importJumpGroup(t, client, "5")
	require.False(t, diags.HasError(), "%v", diags)
	jp := &jumpointResource{}
	jp.ApiClient = client
	_, diags = importResource(t, jp, "5")
	require.False(t, diags.HasError(), "%v", diags)

	assert.Equal(t, 1, srv.Calls(http.MethodGet, "group-policy"))
	assert.Equal(t, 1, srv.Calls(http.MethodGet, "group-policy/1/jumpoint"))
}

// A group policy deleted between the list and its member fetch has no members.
func TestImportGPMemberships_DeletedGroupPolicyCountsAsEmpty(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("group-policy", gpRows(1, 2)...)
	srv.Rows("group-policy/2/jump-group", jgMember(5, 1))
	// group-policy/1/jump-group is not registered: the mock 404s it.

	state, diags := importJumpGroup(t, apimockClient(srv), "5")
	require.False(t, diags.HasError(), "%v", diags)
	_, members := jumpGroupMemberships(t, state)
	require.Len(t, members, 1)
	assert.Equal(t, "2", *members[0].GroupPolicyID)
}

// A failed build is kept like a successful one, so a run with many import blocks
// doesn't repeat the failing listing for each.
func TestImportGPMemberships_FailedBuildIsCached(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Set("group-policy", &apimock.Collection{Status: http.StatusInternalServerError})
	client := apimockClient(srv)

	_, first := importJumpGroup(t, client, "5")
	require.True(t, first.HasError())
	calls := len(srv.Requests())

	_, second := importJumpGroup(t, client, "6")
	require.True(t, second.HasError())
	assert.Equal(t, first.Errors()[0].Detail(), second.Errors()[0].Detail())
	assert.Len(t, srv.Requests(), calls, "no new calls")
}

// Refresh is unchanged: state written by v1.4.0, with memberships null, reads
// back null and never lists group policies.
func TestReadGPMemberships_NullStateNeverListsGroupPolicies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("jump-group", apimock.Row{"id": 5, "name": "jg", "code_name": "jg", "comments": ""})

	r := &jumpGroupResource{}
	r.ApiClient = apimockClient(srv)
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	gpmType := schemaResp.Schema.Attributes["group_policy_memberships"].GetType().(types.SetType)

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(ctx, &models.JumpGroup{
		ID:                     types.StringValue("5"),
		Name:                   types.StringValue("jg"),
		CodeName:               types.StringValue("jg"),
		Comments:               types.StringValue(""),
		GroupPolicyMemberships: types.SetNull(gpmType.ElemType),
	}).HasError())

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	assert.True(t, resp.State.Raw.Equal(state.Raw), "state unchanged")
	assert.Zero(t, srv.Calls(http.MethodGet, "group-policy"))
}

// Every resource with group_policy_memberships discovers them on import through
// its own ImportState, not the generic passthrough, and keeps only the imported
// object's row: the policy also lists object 6.
func TestImportStateDiscoversMembershipsOnEveryResource(t *testing.T) {
	t.Parallel()

	vaultMember := func(field string) func(int, string) apimock.Row {
		return func(id int, role string) apimock.Row { return apimock.Row{field: id, "role": role} }
	}
	for name, tc := range map[string]struct {
		newResource func(*api.APIClient) resource.ResourceWithImportState
		relation    string
		// member is the policy's row for object id; role is used by vault types.
		member func(id int, role string) apimock.Row
		vault  bool
	}{
		"jump_group": {func(c *api.APIClient) resource.ResourceWithImportState {
			r := &jumpGroupResource{}
			r.ApiClient = c
			return r
		}, "jump-group", func(id int, _ string) apimock.Row { return jgMember(id, 1) }, false},
		"jumpoint": {func(c *api.APIClient) resource.ResourceWithImportState {
			r := &jumpointResource{}
			r.ApiClient = c
			return r
		}, "jumpoint", func(id int, _ string) apimock.Row { return apimock.Row{"jumpoint_id": id} }, false},
		"vault_account_group": {func(c *api.APIClient) resource.ResourceWithImportState {
			r := &vaultAccountGroupResource{}
			r.ApiClient = c
			return r
		}, "vault-account-group", vaultMember("account_group_id"), true},
		"vault_ssh_account": {func(c *api.APIClient) resource.ResourceWithImportState {
			r := &vaultSSHAccountResource{}
			r.ApiClient = c
			return r
		}, "vault-account", vaultMember("account_id"), true},
		"vault_token_account": {func(c *api.APIClient) resource.ResourceWithImportState {
			r := &vaultTokenAccountResource{}
			r.ApiClient = c
			return r
		}, "vault-account", vaultMember("account_id"), true},
		"vault_username_password_account": {func(c *api.APIClient) resource.ResourceWithImportState {
			r := &vaultUsernamePasswordAccountResource{}
			r.ApiClient = c
			return r
		}, "vault-account", vaultMember("account_id"), true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := apimock.New()
			defer srv.Close()
			srv.Rows("group-policy", gpRows(1)...)
			srv.Rows("group-policy/1/"+tc.relation, tc.member(6, "inject_and_checkout"), tc.member(5, "inject"))

			state, diags := importResource(t, tc.newResource(apimockClient(srv)), "5")
			require.False(t, diags.HasError(), "%v", diags)
			assert.Equal(t, 1, srv.Calls(http.MethodGet, "group-policy/1/"+tc.relation))

			var set types.Set
			require.False(t, state.GetAttribute(context.Background(), path.Root("group_policy_memberships"), &set).HasError())
			require.Len(t, set.Elements(), 1, "only object 5's membership")
			attrs := set.Elements()[0].(types.Object).Attributes()
			assert.Equal(t, types.StringValue("1"), attrs["group_policy_id"])
			if tc.vault {
				assert.Equal(t, types.StringValue("inject"), attrs["role"], "the row for object 5, not 6")
			}
		})
	}
}
