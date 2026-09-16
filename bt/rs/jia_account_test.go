package rs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"terraform-provider-sra/api"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
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

// A note on how these tests seed respState, because it is deliberate and looks
// wrong at first glance.
//
// The real caller does resp.State.Set(ctx, plan) before calling into these
// helpers (bt/rs/api_resource.go:276), so in production respState arrives
// holding the PLANNED value. Most tests here instead seed it with the opposite
// of the expected outcome. That is on purpose: it is what makes "the arm
// returned without writing anything" a failure rather than an invisible no-op,
// and it is what kills the mutation that deletes a SetAttribute call outright.
//
// The cost of that choice is real — seeding unfaithfully cannot detect a bug
// that only manifests when respState starts from the plan, which is exactly how
// a missing write in the planIsGone && stateIsGone arm stayed invisible until it
// was found against a live appliance. TestUpdateAccountJIA_NoOpResolvesAnUnknownPlan
// closes that by seeding faithfully, from jiaRawUnknown(). Keep both shapes: they
// catch different things.

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
		"re-creating an out-of-band-deleted association must POST; a PATCH here is the 400")

	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.False(t, tfObj.IsNull(), "the re-created association must be back in state")
}

// jiaRawUnknown is the plan shape Terraform produces when a configuration drops
// the jump_item_association block: the attribute is Optional + Computed with no
// default, so a null config plans as unknown rather than as null.
func jiaRawUnknown() tftypes.Value {
	return tftypes.NewValue(jiaSchemaType, map[string]tftypes.Value{
		"jump_item_association": tftypes.NewValue(jiaInnerType, tftypes.UnknownValue),
	})
}

// Removing a jump_item_association block from configuration must DELETE, and the
// planned value that expresses the removal is unknown rather than null.
//
// This pins the IsUnknown() half of planIsGone, which nothing else does — every
// other test here reaches the delete arm through a null plan, so dropping
// IsUnknown() leaves them all green. Without the fold, an unknown falls into the
// !planIsGone branch, tfObj.As runs with UnhandledUnknownAsEmpty, and the
// provider PATCHes filter_type: "" — reintroducing the value this whole change
// exists to keep out.
func TestUpdateAccountJIA_UnknownPlanIsARemoval(t *testing.T) {
	ctx := context.Background()

	var methods []string
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			return false
		}
		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusNoContent)
		return true
	})

	sch := jiaTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRawUnknown()}
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	respState := tfsdk.State{Schema: sch, Raw: jiaRaw(true)}
	var diags diag.Diagnostics

	UpdateAccountJIA(ctx, client, plan, state, &respState, &diags, 99)

	assert.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []string{http.MethodDelete}, methods,
		"an unknown planned association means the block was removed, so it must DELETE and nothing else")

	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.True(t, tfObj.IsNull(), "and the removal must leave state absent, not unknown or empty")
}

// The no-op arm still has to write. An account whose config omits the block
// plans that attribute as unknown as soon as any OTHER attribute changes, so
// this arm runs with plan unknown and state null — and returning without
// writing leaves the unknown in the applied state, which Terraform rejects with
// "provider returned invalid result object after apply", after the account PATCH
// has already gone to the appliance.
//
// Measured against a live appliance 2026-09-15: changing only `description` on a
// vault SSH account with no jump_item_association block reproduced exactly that
// error. TestUpdateAccountJIA_NoOp covers the plan-null/state-null pair and
// cannot see this, because a null plan leaves a null behind either way.
func TestUpdateAccountJIA_NoOpResolvesAnUnknownPlan(t *testing.T) {
	ctx := context.Background()

	var calls int
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			calls++
		}
		return false
	})

	sch := jiaTestSchema()
	plan := tfsdk.Plan{Schema: sch, Raw: jiaRawUnknown()} // config omits the block
	state := tfsdk.State{Schema: sch, Raw: jiaRaw(false)} // and there is no association
	respState := tfsdk.State{Schema: sch, Raw: jiaRawUnknown()}
	var diags diag.Diagnostics

	UpdateAccountJIA(ctx, client, plan, state, &respState, &diags, 99)

	assert.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, 0, calls, "there is nothing to create, update or delete")

	var tfObj types.Object
	d := respState.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
	assert.False(t, d.HasError())
	assert.False(t, tfObj.IsUnknown(), "an unknown left in applied state fails the apply outright")
	assert.True(t, tfObj.IsNull(), "and the value it resolves to is null, because there is no association")
}

// The association's criteria say which Jump Items a stored credential may be
// injected into. That is infrastructure scope rather than a secret, but it has
// no business in a debug log either, and the repo's logging convention is to
// record the type via logItem rather than the value.
//
// Masking cannot be the safety net here: it rides on a context, and a resource
// handler's context is built per RPC rather than inherited from the one the
// provider configured. So the only thing keeping scope out of these lines is the
// call sites not passing it, which is exactly what this test pins.
//
// The canary contains no character json.Marshal would escape. A value with a
// quote or backslash serialises differently from its Go form, so NotContains
// against the raw string would pass while the value sat in the output escaped.
func TestAccountJIAHandlersDoNotLogAssociationScope(t *testing.T) {
	const canary = "c4n4ryJumpItemTagMustNotBeLogged"

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/jump-item-association") {
			return false
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	})

	sch := jiaTestSchema()
	withTag := tftypes.NewValue(jiaSchemaType, map[string]tftypes.Value{
		"jump_item_association": tftypes.NewValue(jiaInnerType, map[string]tftypes.Value{
			"filter_type": tftypes.NewValue(tftypes.String, "criteria"),
			"criteria": tftypes.NewValue(jiaCriteriaType, map[string]tftypes.Value{
				"shared_jump_groups": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, nil),
				"host":               tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
				"name":               tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
				"tag": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, canary),
				}),
				"comment": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
			}),
			"jump_items": tftypes.NewValue(tftypes.Set{ElementType: jiaJumpItemType}, nil),
		}),
	})

	for _, tc := range []struct {
		name string
		run  func(ctx context.Context, diags *diag.Diagnostics)
	}{
		{"create", func(ctx context.Context, diags *diag.Diagnostics) {
			state := tfsdk.State{Schema: sch, Raw: withTag}
			CreateAccountJIA(ctx, client, tfsdk.Plan{Schema: sch, Raw: withTag}, &state, diags, 99)
		}},
		{"read", func(ctx context.Context, diags *diag.Diagnostics) {
			respState := tfsdk.State{Schema: sch, Raw: withTag}
			ReadAccountJIA(ctx, client, tfsdk.State{Schema: sch, Raw: withTag}, &respState, diags, 99)
		}},
		{"update", func(ctx context.Context, diags *diag.Diagnostics) {
			respState := tfsdk.State{Schema: sch, Raw: withTag}
			UpdateAccountJIA(ctx, client, tfsdk.Plan{Schema: sch, Raw: withTag},
				tfsdk.State{Schema: sch, Raw: withTag}, &respState, diags, 99)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &buf)
			var diags diag.Diagnostics
			tc.run(ctx, &diags)

			require.NotEmpty(t, buf.String(), "nothing was logged, so this would pass vacuously")
			assert.NotContains(t, buf.String(), canary,
				"the %s handler logged the association's criteria: %s", tc.name, buf.String())
		})
	}
}

// jump_item_association must be Optional and NOT Computed on the three vault
// ACCOUNT resources, and Computed on the account GROUP resource.
//
// The distinction is not stylistic. Computed tells Terraform the provider will
// supply a value when the configuration does not, so for a null config the
// planned value becomes unknown rather than null, and a plan that is in fact
// removing an association renders it as "(known after apply)" under "1 to
// change". Measured against a live appliance: an account imported with an
// association, whose configuration declares no block, had that association
// deleted by an apply that changed only its description, with nothing in the
// plan saying so. Without Computed the same plan renders the removal.
//
// The account group is the opposite case and keeps Computed: it carries an
// objectdefault, so it really does supply a value the configuration omits, and
// the framework rejects a default on a non-computed attribute outright.
func TestJumpItemAssociationIsComputedOnlyWhereItSuppliesAValue(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		res  resource.Resource
	}{
		{"sra_vault_ssh_account", &vaultSSHAccountResource{}},
		{"sra_vault_token_account", &vaultTokenAccountResource{}},
		{"sra_vault_username_password_account", &vaultUsernamePasswordAccountResource{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var resp resource.SchemaResponse
			tc.res.Schema(ctx, resource.SchemaRequest{}, &resp)
			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			attr, ok := resp.Schema.Attributes["jump_item_association"]
			require.True(t, ok, "attribute missing from the schema")

			assert.True(t, attr.IsOptional(), "must stay settable")
			assert.False(t, attr.IsComputed(),
				"Computed here makes a removal plan as (known after apply), so an association "+
					"is deleted without the plan announcing it")
		})
	}

	t.Run("sra_vault_account_group keeps it", func(t *testing.T) {
		var resp resource.SchemaResponse
		(&vaultAccountGroupResource{}).Schema(ctx, resource.SchemaRequest{}, &resp)
		require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

		attr, ok := resp.Schema.Attributes["jump_item_association"]
		require.True(t, ok, "attribute missing from the schema")
		assert.True(t, attr.IsComputed(),
			"this resource carries an objectdefault, which the framework requires be Computed")
	})
}

// jiaConfig builds a jump_item_association object value the way a configuration
// would, so these tests exercise the real schema rather than a hand-built struct.
// criteriaTags == nil means the criteria block is absent entirely.
func jiaConfig(filterType string, criteriaTags []string, withJumpItem bool) types.Object {
	strSet := func(vals []string) types.Set {
		elems := make([]attr.Value, 0, len(vals))
		for _, v := range vals {
			elems = append(elems, types.StringValue(v))
		}
		return types.SetValueMust(types.StringType, elems)
	}
	// Derived from the schema, not restated: a hand-written copy silently drifts
	// when a criteria property is added, and the symptom is ObjectValueMust
	// panicking rather than a test reporting the gap.
	outer := accountJumpItemAssociationSchema().GetType().(types.ObjectType).AttrTypes
	criteriaTypes := outer["criteria"].(types.ObjectType).AttrTypes
	criteria := types.ObjectNull(criteriaTypes)
	if criteriaTags != nil {
		criteria = types.ObjectValueMust(criteriaTypes, map[string]attr.Value{
			"shared_jump_groups": types.SetValueMust(types.Int64Type, []attr.Value{}),
			"host":               strSet([]string{}),
			"name":               strSet([]string{}),
			"tag":                strSet(criteriaTags),
			"comment":            strSet([]string{}),
		})
	}
	jumpItemType := outer["jump_items"].(types.SetType).ElementType().(types.ObjectType)
	jumpItems := types.SetValueMust(jumpItemType, []attr.Value{})
	if withJumpItem {
		jumpItems = types.SetValueMust(jumpItemType, []attr.Value{
			types.ObjectValueMust(jumpItemType.AttrTypes, map[string]attr.Value{
				"id": types.Int64Value(1), "type": types.StringValue("shell_jump"),
			}),
		})
	}
	return types.ObjectValueMust(outer, map[string]attr.Value{
		"filter_type": types.StringValue(filterType),
		"criteria":    criteria,
		"jump_items":  jumpItems,
	})
}

// What goes on the wire for criteria, in all three shapes the appliance
// distinguishes. Each expectation below was measured against a live appliance.
//
//   - criteria sent as an explicit null -> 422 "This value must be an array."
//     This is what the provider used to send for a bare any_jump_items or
//     no_jump_items association, so neither could be created at all.
//   - criteria key omitted -> the appliance PRESERVES whatever the association
//     already had. Safe only when the filter type ignores criteria.
//   - criteria present with all five properties empty -> the appliance CLEARS it
//     and afterwards reports criteria: null, which is what a configuration
//     declaring no criteria block plans.
//
// So the rule is not "omit when nil" -- that would keep a scope the configuration
// no longer asks for, and the applied state would disagree with the plan. The rule
// is "omit only when the filter type ignores criteria".
func TestJumpItemAssociationCriteriaOnTheWire(t *testing.T) {
	ctx := context.Background()

	for _, filterType := range []string{"any_jump_items", "no_jump_items"} {
		t.Run(filterType+" with no criteria block", func(t *testing.T) {
			var apiSub api.AccountJumpItemAssociation
			d := jiaConfig(filterType, nil, false).As(ctx, &apiSub,
				basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
			require.False(t, d.HasError(), "%v", d)

			blob, err := json.Marshal(apiSub)
			require.NoError(t, err)
			var body map[string]any
			require.NoError(t, json.Unmarshal(blob, &body))

			_, present := body["criteria"]
			assert.False(t, present,
				"the appliance rejects an explicit null, so the key must be absent entirely: %s", blob)
		})
	}

	// The case that makes omitempty safe. jump_items carries the scope, the
	// configuration declares no criteria block, and the validator allows it -- so
	// the wire shape is the only thing standing between this and an association
	// quietly keeping criteria the configuration dropped.
	t.Run("criteria filter with no criteria block sends an empty one, not nothing", func(t *testing.T) {
		var apiSub api.AccountJumpItemAssociation
		d := jiaConfig("criteria", nil, true).As(ctx, &apiSub,
			basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		require.False(t, d.HasError(), "%v", d)
		require.Nil(t, apiSub.Criteria, "fixture assumption: no criteria block means a nil Criteria")

		blob, err := json.Marshal(apiSub)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(blob, &body))

		criteria, present := body["criteria"]
		require.True(t, present,
			"omitting it makes the appliance PRESERVE the old criteria, so state would disagree with the plan: %s", blob)
		require.NotNil(t, criteria, "an explicit null is rejected with 422: %s", blob)

		for name, value := range criteria.(map[string]any) {
			assert.Equal(t, []any{}, value,
				"%s must be an empty array: a null there is the 422, and omitting it preserves instead of clearing", name)
		}
	})

	t.Run("a populated criteria still sends its empty sets", func(t *testing.T) {
		var apiSub api.AccountJumpItemAssociation
		d := jiaConfig("criteria", []string{"keep"}, false).As(ctx, &apiSub,
			basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		require.False(t, d.HasError(), "%v", d)

		blob, err := json.Marshal(apiSub)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(blob, &body))

		criteria, ok := body["criteria"].(map[string]any)
		require.True(t, ok, "criteria must be present when it is configured: %s", blob)
		assert.Equal(t, []any{"keep"}, criteria["tag"])
		for _, empty := range []string{"host", "name", "comment", "shared_jump_groups"} {
			value, present := criteria[empty]
			assert.True(t, present, "%s must be sent, not omitted: omitting it PRESERVES the old value", empty)
			assert.Equal(t, []any{}, value, "%s must marshal as an empty array, not null", empty)
		}
	})
}

// The validator encodes the appliance's own precondition. Every rejected case
// here was measured returning 422 from a live appliance; every accepted one
// returned 200.
func TestJumpItemAssociationFilterValidator(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		value   types.Object
		wantErr bool
	}{
		{"criteria with a tag", jiaConfig("criteria", []string{"t"}, false), false},
		{"criteria with only jump_items", jiaConfig("criteria", nil, true), false},
		{"criteria with no criteria block and no jump items", jiaConfig("criteria", nil, false), true},
		{"criteria with an all-empty criteria block", jiaConfig("criteria", []string{}, false), true},
		{"any_jump_items needs nothing", jiaConfig("any_jump_items", nil, false), false},
		{"no_jump_items needs nothing", jiaConfig("no_jump_items", nil, false), false},
		{"a null association is not this validator's business", types.ObjectNull(jiaConfig("criteria", nil, false).AttributeTypes(ctx)), false},
		{"an unknown association cannot be judged", types.ObjectUnknown(jiaConfig("criteria", nil, false).AttributeTypes(ctx)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.ObjectResponse{}
			jumpItemAssociationFilterValidator{}.ValidateObject(ctx,
				validator.ObjectRequest{Path: path.Root("jump_item_association"), ConfigValue: tc.value}, resp)

			if tc.wantErr {
				require.True(t, resp.Diagnostics.HasError(),
					"this configuration is rejected by the appliance with a 422; it must not reach apply")
				assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "criteria",
					"the message must name the attribute the operator has to fix")
				return
			}
			assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
		})
	}
}
