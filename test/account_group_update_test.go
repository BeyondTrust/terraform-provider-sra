package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/require"
)

// TestAccountGroupMembershipUpdate drives a genuine config-change update.
//
// Every other apply-twice in this suite reloads identical saved options, so it
// exercises idempotency rather than change: the provider's Update sees a plan
// equal to state. This test changes the config between applies, toggling
// group_policy_memberships present -> absent -> present.
//
// Step 2 is the valuable one. It is the only place in the suite that drives
// UpdateGPMemberships' remove path to completion, and it pins the null-vs-empty
// distinction: when the last membership is removed, the provider must write
// types.SetNull, not an empty set. The code under test is the tfGPList.IsNull()
// branch at bt/rs/gp_membership.go:325-332.
//
// Why writing an empty set would fail rather than merely look untidy:
// group_policy_memberships is Optional and non-Computed
// (bt/rs/vault_account_group.go:108-109), so for a null config the planned value
// is null. A provider returning an empty set where null was planned fails
// Terraform core's consistency check with "provider produced inconsistent result
// after apply" -- a hard error, so terraform.Apply below fails the test.
func TestAccountGroupMembershipUpdate(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/vault/account_group_update", productPath()))

	// Reassigned between stages to change the config. Options are saved/loaded
	// through test_structure, so the toggle has to be written back each time.
	withMembership := func(t *testing.T, on bool) *terraform.Options {
		opts := withBaseTFOptions(t, &terraform.Options{
			TerraformDir: testFolder,
			Vars: map[string]interface{}{
				"random_bits":        randomBits,
				"with_gp_membership": on,
			},
		})
		test_structure.SaveTerraformOptions(t, testFolder, opts)
		return opts
	}

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraform.Destroy(t, test_structure.LoadTerraformOptions(t, testFolder))
	})

	var groupPolicyID string

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := withMembership(t, true)
		terraform.InitAndApply(t, terraformOptions)

		groupPolicyID = terraform.OutputMap(t, terraformOptions, "gp")["id"]
		require.NotEmpty(t, groupPolicyID, "could not read the group policy id from outputs")
	})

	test_structure.RunTestStage(t, "Membership is present after the initial apply", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		assertSoleMembership(t, extractJson(t, terraformOptions, "group"), groupPolicyID,
			"the account group should start with one membership")
	})

	test_structure.RunTestStage(t, "Removing the last membership writes null, not an empty set", func() {
		// The apply itself is the assertion: if the remove path wrote an empty set
		// instead of types.SetNull, Terraform would reject the applied state as
		// inconsistent with the null plan and this call would fail.
		terraformOptions := withMembership(t, false)
		terraform.Apply(t, terraformOptions)

		assertNoGPMembership(t, extractJson(t, terraformOptions, "group"))

		// And the removal must be genuinely clean: re-planning the same config
		// must produce no diff. An empty-vs-null mismatch that somehow survived
		// the apply would surface here as a perpetual diff.
		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
			"removing the last membership left a perpetual diff")
	})

	test_structure.RunTestStage(t, "Re-adding the membership restores it", func() {
		terraformOptions := withMembership(t, true)
		terraform.Apply(t, terraformOptions)

		assertSoleMembership(t, extractJson(t, terraformOptions, "group"), groupPolicyID,
			"the membership should have been re-created")
	})
}
