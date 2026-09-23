package test

import (
	"fmt"
	"strconv"
	"testing"

	"terraform-provider-sra/api"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
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

	test_structure.RunTestStage(t, "setup", func() {
		terraform.InitAndApply(t, withMembership(t, true))
	})

	test_structure.RunTestStage(t, "Membership is present after the initial apply", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		groupPolicyID, accountGroupID := fixtureIDs(t, terraformOptions)

		assertSoleMembership(t, extractJson(t, terraformOptions, "group"), groupPolicyID,
			"the account group should start with one membership")
		assert.True(t, membershipLiveOnAppliance(t, groupPolicyID, accountGroupID),
			"the membership should exist on the appliance, not just in state")
	})

	test_structure.RunTestStage(t, "Removing the last membership writes null, not an empty set", func() {
		// The apply itself is the assertion: if the remove path wrote an empty set
		// instead of types.SetNull, Terraform would reject the applied state as
		// inconsistent with the null plan and this call would fail.
		terraformOptions := withMembership(t, false)
		terraform.Apply(t, terraformOptions)
		groupPolicyID, accountGroupID := fixtureIDs(t, terraformOptions)

		assertNoGPMembership(t, extractJson(t, terraformOptions, "group"))
		assert.False(t, membershipLiveOnAppliance(t, groupPolicyID, accountGroupID),
			"the membership is gone from state but still live on the appliance — "+
				"the remove path recorded the removal without performing it")

		// And the removal must be genuinely clean: re-planning the same config
		// must produce no diff. An empty-vs-null mismatch that somehow survived
		// the apply would surface here as a perpetual diff.
		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
			"removing the last membership left a perpetual diff")
	})

	test_structure.RunTestStage(t, "Re-adding the membership restores it", func() {
		terraformOptions := withMembership(t, true)
		terraform.Apply(t, terraformOptions)
		groupPolicyID, accountGroupID := fixtureIDs(t, terraformOptions)

		assertSoleMembership(t, extractJson(t, terraformOptions, "group"), groupPolicyID,
			"the membership should have been re-created")
		assert.True(t, membershipLiveOnAppliance(t, groupPolicyID, accountGroupID),
			"the membership should have been re-created on the appliance, not just in state")
	})
}

// fixtureIDs re-reads the group policy and account group ids from outputs.
//
// These are deliberately NOT held in cross-stage Go variables. test_structure's
// SKIP_<stage> workflow lets a developer re-run later stages against cached state
// (SKIP_setup=true go test -run ...), and a variable produced in a skipped stage
// stays at its zero value. The damage is not just a nil id: assertSoleMembership
// would then compare against "" and report "the membership must reference the
// group policy from the config" -- announcing the exact corruption it exists to
// detect, for a test that simply did not run its producing stage. Every
// pre-existing test in this suite re-derives cross-stage values the same way and
// holds only randomBits.
func fixtureIDs(t *testing.T, opts *terraform.Options) (groupPolicyID, accountGroupID string) {
	t.Helper()

	groupPolicyID = terraform.OutputMap(t, opts, "gp")["id"]
	require.NotEmpty(t, groupPolicyID, "could not read the group policy id from outputs")

	accountGroupID = terraform.OutputMap(t, opts, "group")["id"]
	require.NotEmpty(t, accountGroupID, "could not read the account group id from outputs")

	return groupPolicyID, accountGroupID
}

// membershipLiveOnAppliance asks the appliance directly whether the group policy
// membership still exists, bypassing Terraform state entirely.
//
// This is the difference between testing that the provider RECORDED a removal and
// testing that it PERFORMED one. After step 2 the state attribute is null, and
// ReadGPMemberships returns early on a null attribute (bt/rs/gp_membership.go:155),
// so a refresh never queries this endpoint -- meaning a regression where
// UpdateGPMemberships writes types.SetNull correctly but skips or swallows its
// DeleteItem calls would leave the membership live and still show a clean plan.
func membershipLiveOnAppliance(t *testing.T, groupPolicyID, accountGroupID string) bool {
	t.Helper()

	id, err := strconv.Atoi(accountGroupID)
	require.NoError(t, err, "unusable account group id %q", accountGroupID)

	endpoint := fmt.Sprintf("group-policy/%s/vault-account-group/%d", groupPolicyID, id)
	items, err := api.ListItemsEndpoint[api.GroupPolicyVaultAccountGroup](freshClient(t), endpoint)
	if api.IsNotFound(err) {
		// Object-returning endpoints 404 for a removed membership; array-returning
		// ones answer with an empty list. Both mean "gone".
		return false
	}
	require.NoError(t, err, "could not read memberships for account group %d from the appliance", id)
	return len(items) > 0
}
