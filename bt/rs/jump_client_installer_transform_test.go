package rs

import (
	"context"
	"reflect"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// Regression guard for the TestJumpClientInstaller E2E failure:
//
//	Provider produced inconsistent result after apply
//	.elevate_install: was cty.True, but now cty.False
//
// The POST /jump-client/installer create response does NOT echo the requested
// elevate_install/elevate_prompt — it always returns them as false (verified
// against a live appliance). The generic Create copies the response over the
// plan, so the planned `true` became `false`.
//
// The fix tags these fields sra:"persist_state" so CopyAPItoTF preserves the
// planned value instead of overwriting it from the unreliable create response.
// This test reproduces the create flow at the transform level: plan=true, API
// response=false, and asserts the planned value survives.
func TestJumpClientInstaller_ElevatePersistsOverCreateResponse(t *testing.T) {
	ctx := context.Background()

	// The plan (already read into the TF object by the create flow) has the
	// defaulted true values.
	plan := models.JumpClientInstaller{
		ID:             types.StringValue("124"),
		ElevateInstall: types.BoolValue(true),
		ElevatePrompt:  types.BoolValue(true),
	}

	// The installer create endpoint returns elevate_install/elevate_prompt as
	// false regardless of what was sent.
	id := 124
	apiResp := api.JumpClientInstaller{
		ID:             &id,
		ElevateInstall: false,
		ElevatePrompt:  false,
	}

	err := api.CopyAPItoTF(ctx, reflect.ValueOf(&apiResp).Elem(), reflect.ValueOf(&plan).Elem(), reflect.TypeOf(apiResp), api.ProductRS)
	assert.NoError(t, err)

	assert.True(t, plan.ElevateInstall.ValueBool(), "elevate_install must persist the planned value, not the create response's false")
	assert.True(t, plan.ElevatePrompt.ValueBool(), "elevate_prompt must persist the planned value, not the create response's false")
}
