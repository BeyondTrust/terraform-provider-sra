package rs

import (
	"context"
	"net/http"
	"strings"
	"terraform-provider-sra/api"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test scaffolding for the vault-account JIA helpers ---------------------

func jiaTestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jump_item_association": accountJumpItemAssociationSchema(),
		},
	}
}

var (
	jiaCriteriaType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"shared_jump_groups": tftypes.Set{ElementType: tftypes.Number},
		"host":               tftypes.Set{ElementType: tftypes.String},
		"name":               tftypes.Set{ElementType: tftypes.String},
		"tag":                tftypes.Set{ElementType: tftypes.String},
		"comment":            tftypes.Set{ElementType: tftypes.String},
	}}
	jiaJumpItemType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":   tftypes.Number,
		"type": tftypes.String,
	}}
	jiaInnerType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"filter_type": tftypes.String,
		"criteria":    jiaCriteriaType,
		"jump_items":  tftypes.Set{ElementType: jiaJumpItemType},
	}}
	jiaSchemaType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"jump_item_association": jiaInnerType}}
)

// jiaRaw builds the object value; present=false yields a null association.
func jiaRaw(present bool) tftypes.Value {
	var inner tftypes.Value
	if !present {
		inner = tftypes.NewValue(jiaInnerType, nil)
	} else {
		inner = tftypes.NewValue(jiaInnerType, map[string]tftypes.Value{
			"filter_type": tftypes.NewValue(tftypes.String, "any_jump_items"),
			"criteria":    tftypes.NewValue(jiaCriteriaType, nil),
			"jump_items":  tftypes.NewValue(tftypes.Set{ElementType: jiaJumpItemType}, nil),
		})
	}
	return tftypes.NewValue(jiaSchemaType, map[string]tftypes.Value{"jump_item_association": inner})
}

// --- tests ------------------------------------------------------------------

// The delete transition (association present in state, removed from plan) must
// issue a DELETE. This branch is never exercised by the E2E suite, which only
// re-applies the same config.
func TestUpdateAccountJIA_DeleteTransition(t *testing.T) {
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

	sch := jiaTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRaw(false)}  // association removed
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(true)} // association present
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	var diags diag.Diagnostics

	UpdateAccountJIA(ctx, client, plan, state, &respState, &diags, 99)

	assert.False(t, diags.HasError())
	assert.Equal(t, 1, deleted, "removing the association from the plan should issue a DELETE")

	// The delete branch must write the empty association, not echo anything
	// back (there is nothing to echo — the plan is gone). This guards against
	// a future refactor routing the delete branch through the same
	// createdOrSent resolution the create branch uses just below.
	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	// Null, not a zero-value struct. stateIsGone above is IsNull || IsUnknown, so
	// a zero-value struct reads as "still present" and the next apply re-enters
	// this same delete branch against an association that is already gone.
	assert.True(t, tfObj.IsNull(), "a deleted association is absent, and absence is null")
}

// The update-path CREATE branch (state absent, plan present) has the same
// 204-zero-value hazard as CreateAccountJIA: api.CreateItem returns (nil, nil)
// on a 204 No Content, and the association WAS created. This must resolve to
// the echoed plan value, not a zero-value association — distinct from the
// DELETE branch immediately above, which must NOT echo anything.
func TestUpdateAccountJIA_CreatePath204NoContentTolerated(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		return false
	})

	sch := jiaTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRaw(true)}    // association added
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(false)} // association absent
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	var diags diag.Diagnostics

	UpdateAccountJIA(ctx, client, plan, state, &respState, &diags, 99)

	assert.False(t, diags.HasError(), "%v", diags)

	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.False(t, tfObj.IsNull(), "a 204 on create must still leave the association present, not dropped")

	var apiSub api.AccountJumpItemAssociation
	d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	assert.False(t, d.HasError())
	assert.Equal(t, "any_jump_items", apiSub.FilterType, "a 204 on create must echo the planned filter_type, not a zero-value fallback")
}

// Both plan and state absent is a no-op: no API calls, no error.
func TestUpdateAccountJIA_NoOp(t *testing.T) {
	ctx := context.Background()

	var calls int
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodDelete || r.Method == http.MethodPost || r.Method == http.MethodPatch {
			calls++
		}
		return false
	})

	sch := jiaTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRaw(false)}
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	var diags diag.Diagnostics

	UpdateAccountJIA(ctx, client, plan, state, &respState, &diags, 99)

	assert.False(t, diags.HasError())
	assert.Equal(t, 0, calls, "no association in plan or state should make no API calls")
}

// mockJIAGetClient stands up an httptest server whose GET on the account JIA
// endpoint returns status. Used for the B1 ReadAccountJIA regression tests.
func mockJIAGetClient(t *testing.T, status int) *api.APIClient {
	t.Helper()
	return mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodGet {
			w.WriteHeader(status)
			return true
		}
		return false
	})
}

// B1: a genuine API error (not 404, not the documented inherit/corrupted-state
// tolerance) against a populated association must surface a diagnostic.
func TestReadAccountJIA_ErrorWithPopulatedStateSurfaces(t *testing.T) {
	ctx := context.Background()
	client := mockJIAGetClient(t, http.StatusInternalServerError)

	sch := jiaTestSchema()
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	var diags diag.Diagnostics

	ReadAccountJIA(ctx, client, state, &respState, &diags, 99)

	assert.True(t, diags.HasError(), "a populated association with a genuine API error must surface a diagnostic")
}

// B1: an association-less account (state absent) hitting the documented
// "cannot be used while inheriting from Account Group" error must not
// hard-fail every refresh, and must leave state untouched rather than
// fabricating an empty association.
func TestReadAccountJIA_ErrorWithNoStateTolerated(t *testing.T) {
	ctx := context.Background()
	client := mockJIAGetClient(t, http.StatusInternalServerError)

	sch := jiaTestSchema()
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	var diags diag.Diagnostics

	ReadAccountJIA(ctx, client, state, &respState, &diags, 99)

	assert.False(t, diags.HasError(), "the inherit case must not hard-fail")
	assert.True(t, respState.Raw.Equal(jiaRaw(false)), "state must be left untouched, not fabricated")
}

// A 404 means there is no association — either deleted out of band, or never
// created. It clears to NULL, not to a zero-value struct, and does not error.
//
// This test previously asserted the opposite ("a 404 should clear to an empty
// association, not remove it"). That was defending empty-vs-error, not
// empty-vs-null: null was never considered. It landed in 1e90278 alongside the
// CreateAccountJIA comment stating that a zero-value association puts
// filter_type: "" into state, "a value the attribute's own contract forbids" —
// the same commit thus fixed the create path and codified the bug in the read
// path. A zero-value struct is not absence; UpdateAccountJIA's stateIsGone check
// (IsNull || IsUnknown) does not recognise it as such.
func TestReadAccountJIA_NotFoundClearsToNull(t *testing.T) {
	ctx := context.Background()
	client := mockJIAGetClient(t, http.StatusNotFound)

	sch := jiaTestSchema()
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	var diags diag.Diagnostics

	ReadAccountJIA(ctx, client, state, &respState, &diags, 99)

	assert.False(t, diags.HasError())

	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.True(t, tfObj.IsNull(), "a 404 means no association, which is a null object")
}

// Finding 9: CreateItem returns (nil, nil) on a 204 No Content — the
// association WAS created, there is simply no body. The create path must
// echo the accepted request rather than writing a zero-value association:
// a zero value would put filter_type: "" into state, a value the attribute's
// own contract forbids (Required + stringvalidator.OneOf, api_resource.go),
// and fail the apply with an inconsistent-result error.
func TestCreateAccountJIA_204NoContentTolerated(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		return false
	})

	sch := jiaTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRaw(true)}
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	var diags diag.Diagnostics

	CreateAccountJIA(ctx, client, plan, &state, &diags, 99)

	assert.False(t, diags.HasError(), "%v", diags)

	var tfObj types.Object
	d := state.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.False(t, tfObj.IsNull(), "a 204 must still leave the association present, not dropped")

	var apiSub api.AccountJumpItemAssociation
	d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	assert.False(t, d.HasError())
	assert.Equal(t, "any_jump_items", apiSub.FilterType, "a 204 must echo the planned filter_type, not a zero-value fallback")
}

// The ordinary refresh for an association-less account: nothing in state, and
// the GET 404s because there is nothing to get. Every other read test here
// starts from a populated state, so this — the path most accounts take on every
// single refresh — went uncovered.
//
// It also pins the part of the fix that is not self-evident. The null object is
// built from tfObj.AttributeTypes(ctx), and tfObj is itself null here. A null
// types.Object still carries its attribute types, so what gets written is a
// typed null; if it did not, SetAttribute would reject the value outright.
func TestReadAccountJIA_NotFoundWithNoStateStaysNull(t *testing.T) {
	ctx := context.Background()
	client := mockJIAGetClient(t, http.StatusNotFound)

	sch := jiaTestSchema()
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(false)}
	var diags diag.Diagnostics

	ReadAccountJIA(ctx, client, state, &respState, &diags, 99)

	assert.False(t, diags.HasError(), "%v", diags)

	// Raw equality with jiaRaw(false) subsumes an IsNull check -- that fixture
	// builds the association as a null object -- and adds that nothing else moved.
	assert.True(t, respState.Raw.Equal(jiaRaw(false)),
		"absent before the refresh, absent and untouched after it")
}

// Out-of-band deletion, refresh through re-create. This is the two halves of the
// fix working as a pair, and the HTTP verb is what proves it.
//
// Because the refresh clears to null, UpdateAccountJIA sees stateIsGone and
// routes the re-create to POST, which is how an association is created.
// Clearing to a zero-value struct instead leaves stateIsGone false, so the same
// apply PATCHes an association that no longer exists. Measured against a live
// appliance 2026-09-15: that PATCH returns 400 "Account does not have an Asset
// association." both after an out-of-band delete and for an account that never
// had one, while the POST returns 200 in both cases.
func TestAccountJIA_OutOfBandDeletionRecreatesWithPOST(t *testing.T) {
	ctx := context.Background()

	var methods []string
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			return false
		}
		if r.Method == http.MethodGet {
			// Deleted on the appliance since the last apply.
			w.WriteHeader(http.StatusNotFound)
			return true
		}
		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusNoContent)
		return true
	})

	sch := jiaTestSchema()
	// State as the last apply left it: the association present.
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	refreshed := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	var diags diag.Diagnostics

	ReadAccountJIA(ctx, client, state, &refreshed, &diags, 99)
	require.False(t, diags.HasError(), "%v", diags)

	// The config still asks for the association, so the next apply re-creates it
	// against the state the refresh just wrote.
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRaw(true)}
	respState := tfsdk.State{Schema: sch, Raw: refreshed.Raw}
	UpdateAccountJIA(ctx, client, plan, refreshed, &respState, &diags, 99)

	assert.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []string{http.MethodPost}, methods,
		"re-creating an out-of-band-deleted association must POST; a PATCH here is the 405")

	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.False(t, tfObj.IsNull(), "the re-created association must be back in state")
}
