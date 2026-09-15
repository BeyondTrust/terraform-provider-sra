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
	assert.False(t, tfObj.IsNull(), "the delete transition should clear to an empty association, not remove it")

	var apiSub api.AccountJumpItemAssociation
	d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	assert.False(t, d.HasError())
	assert.Equal(t, "", apiSub.FilterType, "a deleted association must clear filter_type, not carry over any prior value")
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

// B1: a 404 clears the association to an empty value rather than erroring.
func TestReadAccountJIA_NotFoundClears(t *testing.T) {
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
	assert.False(t, tfObj.IsNull(), "a 404 should clear to an empty association, not remove it")

	var apiSub api.AccountJumpItemAssociation
	d = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
	assert.False(t, d.HasError())
	assert.Equal(t, "", apiSub.FilterType)
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
