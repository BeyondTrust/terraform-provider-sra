package rs

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/stretchr/testify/assert"
)

// C1 regression guard: CopyAPItoTF's reflect.Slice branch assigned types.Set /
// types.String values into NetworkTunnelJump.FilterRules, which is a types.List.
// reflect.Value.Set enforces assignability, so a null/empty/malformed filter_rules
// from the API panicked the provider (the old unsafe.Pointer version silently
// reinterpreted memory). These branches must produce a (null) types.List instead.

func runFilterRulesCopy(t *testing.T, apiObj api.NetworkTunnelJump) models.NetworkTunnelJump {
	t.Helper()
	var tf models.NetworkTunnelJump
	err := api.CopyAPItoTF(
		context.Background(),
		reflect.ValueOf(&apiObj).Elem(),
		reflect.ValueOf(&tf).Elem(),
		reflect.TypeOf(apiObj),
		api.ProductPRA,
	)
	assert.NoError(t, err)
	return tf
}

// Fails (panics) before the fix: nil *json.RawMessage -> setToNil -> types.SetNull
// assigned into the types.List field.
func TestCopyAPItoTF_FilterRulesNil(t *testing.T) {
	id := 5
	tf := runFilterRulesCopy(t, api.NetworkTunnelJump{ID: &id}) // FilterRules nil
	assert.True(t, tf.FilterRules.IsNull(), "nil API filter_rules must map to a null types.List")
}

// Fails (panics) before the fix: unmarshal error -> types.StringValue assigned into
// the types.List field.
func TestCopyAPItoTF_FilterRulesMalformed(t *testing.T) {
	id := 5
	bad := json.RawMessage(`{ not valid json`)
	tf := runFilterRulesCopy(t, api.NetworkTunnelJump{ID: &id, FilterRules: &bad})
	assert.True(t, tf.FilterRules.IsNull(), "malformed API filter_rules must map to a null types.List, not panic")
}

// Non-regression guard: a valid filter_rules array still maps to a populated list.
func TestCopyAPItoTF_FilterRulesValid(t *testing.T) {
	id := 5
	valid := json.RawMessage(`[{"ip_addresses":{"cidr":"10.0.0.0/24"},"protocol":"ANY","ports":{"list":[22]}}]`)
	tf := runFilterRulesCopy(t, api.NetworkTunnelJump{ID: &id, FilterRules: &valid})
	assert.False(t, tf.FilterRules.IsNull(), "valid API filter_rules must map to a non-null list")
	assert.Equal(t, 1, len(tf.FilterRules.Elements()))
}
