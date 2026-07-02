package rs

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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
