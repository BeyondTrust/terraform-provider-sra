package rs

import (
	"bytes"
	"context"
	"testing"

	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// group_policy_id is interpolated into the request path by the Endpoint() methods
// in api/models.go. These cover the schema-level constraint that keeps
// non-conforming values out; api/endpoint_test.go covers the escaping that makes
// the path safe if this constraint is ever relaxed.

func TestGroupPolicyIDValidator(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		valid bool
		why   string
	}{
		{"numeric", "42", true, "the documented form: ObjectId is integer, minimum 1"},
		{"multi-digit", "1234567", true, "no upper bound is imposed here; the API rejects out-of-range ids"},
		{"leading zero", "007", true, "still numeric; the appliance normalises it"},

		{"empty", "", false, "an empty id would produce group-policy//vault-account"},
		{"contains separators", "../../admin", false, "would add path segments when interpolated into a request path"},
		{"trailing segment", "42/extra", false, "a separator would add a path segment"},
		{"negative", "-1", false, "the spec sets minimum: 1"},
		{"non-numeric", "abc", false, "ids are numeric; a name is not an id"},
		{"numeric with space", "4 2", false, "would be escaped, but is not a valid id"},
		{"pre-encoded separator", "42%2Fadmin", false, "pre-encoded input must not be accepted either"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			validators := groupPolicyIDValidators()
			require.Len(t, validators, 1, "one validator is expected; update this test if that changes")

			resp := &validator.StringResponse{}
			validators[0].ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("group_policy_memberships").AtListIndex(0).AtName("group_policy_id"),
				ConfigValue: types.StringValue(tc.value),
			}, resp)

			if tc.valid {
				assert.False(t, resp.Diagnostics.HasError(), "%q should be accepted (%s): %v", tc.value, tc.why, resp.Diagnostics)
				return
			}
			assert.True(t, resp.Diagnostics.HasError(), "%q should be rejected (%s)", tc.value, tc.why)
		})
	}
}

// TestGroupPolicyIDValidatorIgnoresNullAndUnknown keeps the validator from failing
// a plan where the id is not yet resolvable — the common case of it coming from
// another resource's computed output, where the value is unknown at plan time.
func TestGroupPolicyIDValidatorIgnoresNullAndUnknown(t *testing.T) {
	for name, value := range map[string]types.String{
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			groupPolicyIDValidators()[0].ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("group_policy_id"),
				ConfigValue: value,
			}, resp)

			assert.False(t, resp.Diagnostics.HasError(),
				"a %s id must not fail validation; it is resolved before the request is built", name)
		})
	}
}

// TestLogItemRecordsTypeNotContent guards the generic handlers, which carry every
// resource type and build their item from the Terraform plan. The invariant is
// that an item's values never reach a log field; only its type does.
func TestLogItemRecordsTypeNotContent(t *testing.T) {
	const canary = "c4n4ryPasswordValue"

	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)

	logItem(ctx, "🙀 executing item update", api.VaultUsernamePasswordAccount{
		Username: "someone",
		Password: canary,
	})

	out := buf.String()
	assert.NotContains(t, out, canary, "logItem must never emit field values: %s", out)
	assert.Contains(t, out, "VaultUsernamePasswordAccount",
		"the type is the diagnostic value and should be kept: %s", out)
}

// TestLogItemSurvivesAnUnpopulatedItem pins the reason logItem does not call
// Endpoint(). Several implementations dereference an ID that is not set at the
// point these logs fire — AccountGroupJumpItemAssociation.Endpoint does *a.ID —
// so calling it turned a log line into a nil-pointer panic that failed the apply
// it was describing.
func TestLogItemSurvivesAnUnpopulatedItem(t *testing.T) {
	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)

	assert.NotPanics(t, func() {
		logItem(ctx, "🙀 got item", api.AccountGroupJumpItemAssociation{}) // ID is nil
	}, "a logging helper must not be able to fail the operation it describes")
}
