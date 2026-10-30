package rs

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// Minimal synthetic API/TF type pair for exercising the generic
// apiResource.Create path without any real resource's schema complexity.
// api.APIResource requires only Endpoint(), and api.IsProductAllowed
// defaults to true for any type that doesn't implement RestrictsProducts.
type a2TestAPIItem struct {
	ID   *int   `json:"id,omitempty"`
	Name string `json:"name"`
}

func (a2TestAPIItem) Endpoint() string { return "a2-test-item" }

type a2TestTFItem struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func a2TestSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true},
			"name": schema.StringAttribute{Required: true},
		},
	}
}

// A2: CreateItem returns (nil, nil) on a 204 No Content. Before the fix,
// reflect.ValueOf(newItem).Elem() on a nil *TApi produced a zero
// reflect.Value, and CopyAPItoTF's FieldByName then panicked. The fix must
// surface a diagnostic instead of writing a half-null state.
func TestApiResourceCreate_204NoContent(t *testing.T) {
	ctx := context.Background()

	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		w.WriteHeader(http.StatusNoContent)
		return true
	})

	r := &apiResource[a2TestAPIItem, a2TestTFItem]{ApiClient: client}

	sch := a2TestSchema()
	model := a2TestTFItem{ID: types.StringNull(), Name: types.StringValue("widget")}
	plan := tfsdk.Plan{Schema: sch}
	diags := plan.Set(ctx, &model)
	assert.False(t, diags.HasError(), "%v", diags)

	req := resource.CreateRequest{Plan: plan}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}

	assert.NotPanics(t, func() {
		r.Create(ctx, req, resp)
	})

	assert.True(t, resp.Diagnostics.HasError(), "a 204 create response must surface a diagnostic, not panic")
}

// a2UpdateHarness runs apiResource.update against a stub that records the PATCH
// body and answers with respBody. State holds name "from-state", the plan
// "from-plan", both with ID 7.
func a2UpdateHarness(t *testing.T, respBody string, adjust func(state, plan a2TestTFItem, item *a2TestAPIItem)) (string, *resource.UpdateResponse) {
	t.Helper()
	ctx := context.Background()

	var sent string
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method != http.MethodPatch {
			return false
		}
		assert.Equal(t, "/api/config/v1/a2-test-item/7", r.URL.Path)
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
		return true
	})
	r := &apiResource[a2TestAPIItem, a2TestTFItem]{ApiClient: client}

	sch := a2TestSchema()
	plan := tfsdk.Plan{Schema: sch}
	assert.False(t, plan.Set(ctx, &a2TestTFItem{ID: types.StringValue("7"), Name: types.StringValue("from-plan")}).HasError())
	state := tfsdk.State{Schema: sch}
	assert.False(t, state.Set(ctx, &a2TestTFItem{ID: types.StringValue("7"), Name: types.StringValue("from-state")}).HasError())

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: sch}}
	r.update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp, adjust)
	return sent, resp
}

// Update is shared by every resource, and its body moved into update so one
// resource could adjust the request. With no adjuster the request must still be
// built from the plan alone, and the response must still become the new state.
func TestApiResourceUpdate_NoAdjustSendsThePlan(t *testing.T) {
	sent, resp := a2UpdateHarness(t, `{"id":7,"name":"from-server"}`, nil)
	assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	assert.JSONEq(t, `{"id":7,"name":"from-plan"}`, sent)

	var got a2TestTFItem
	assert.False(t, resp.State.Get(context.Background(), &got).HasError())
	assert.Equal(t, "from-server", got.Name.ValueString())
	assert.Equal(t, "7", got.ID.ValueString())
}

// The adjuster sees the prior state, the plan, and the item already populated
// from the plan, and what it leaves in the item is what goes on the wire.
func TestApiResourceUpdate_AdjustSeesStateAndPlanAndChangesTheBody(t *testing.T) {
	called := false
	sent, resp := a2UpdateHarness(t, `{"id":7,"name":"from-server"}`, func(state, plan a2TestTFItem, item *a2TestAPIItem) {
		called = true
		assert.Equal(t, "from-state", state.Name.ValueString())
		assert.Equal(t, "from-plan", plan.Name.ValueString())
		assert.Equal(t, "from-plan", item.Name, "the adjuster must run after the plan is copied in")
		item.Name = "adjusted"
	})
	assert.True(t, called)
	assert.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	assert.JSONEq(t, `{"id":7,"name":"adjusted"}`, sent)
}

// setHasElements calls Set.Length, which panics on a null or unknown set unless
// it is told not to. The guards above the call are the only thing preventing
// that, and a panic inside a validator aborts the plan rather than failing a
// check -- so this pins every state the value can arrive in.
func TestSetHasElementsSurvivesEverySetState(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  attr.Value
		want bool
	}{
		{"null set", types.SetNull(types.StringType), false},
		{"unknown set", types.SetUnknown(types.StringType), true},
		{"empty known set", types.SetValueMust(types.StringType, []attr.Value{}), false},
		{"populated set", types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")}), true},
		{"not a set at all", types.StringValue("x"), false},
		{"a null object rather than a set", types.ObjectNull(map[string]attr.Type{}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				assert.Equal(t, tc.want, setHasElements(tc.set))
			})
		})
	}
}
