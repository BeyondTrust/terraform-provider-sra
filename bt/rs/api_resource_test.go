package rs

import (
	"context"
	"net/http"
	"testing"

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
