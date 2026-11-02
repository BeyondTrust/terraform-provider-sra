package test

import (
	"fmt"
	"testing"

	"terraform-provider-sra/api"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupPolicy(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/group_policy", productPath(t)))

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := withBaseTFOptions(t, &terraform.Options{
			TerraformDir: testFolder,
			Vars: map[string]interface{}{
				"random_bits": randomBits,
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.InitAndApply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "verify", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		assert.Equal(t, fmt.Sprintf("terraform_group_policy_%s", randomBits), item["name"])
		assert.Equal(t, "true", item["perm_jump_client"])

		terraform.Apply(t, terraformOptions)
		listed := terraform.OutputListOfObjects(t, terraformOptions, "listed")
		assert.Len(t, listed, 1)
		if len(listed) == 1 {
			assert.Equal(t, item["id"], listed[0]["id"])
		}
	})

	test_structure.RunTestStage(t, "removing a permission revokes it", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		const addr = "sra_group_policy.item"

		// perm_invite_external_user (PRA) and perm_accept_team_sessions (RS)
		// have no schema default, so only ModifyPlan can plan their revocation.
		perm := "perm_invite_external_user"
		granted := func(gp *api.GroupPolicy) *bool { return gp.PermInviteExternalUser }
		if mechs(t).IsRS() {
			perm = "perm_accept_team_sessions"
			granted = func(gp *api.GroupPolicy) *bool { return gp.PermAcceptTeamSessions }
		}
		item := terraform.OutputMap(t, terraformOptions, "item")
		require.Equal(t, "true", item[perm], "the fixture grants %s", perm)

		terraformOptions.Vars["grant_optional_permission"] = false
		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)

		plan := planStruct(t, terraformOptions)
		change, ok := plan.ResourceChangesMap[addr]
		require.True(t, ok, "%s is missing from the plan", addr)
		require.True(t, change.Change.Actions.Update(), "removing %s must update the policy, got %v", perm, change.Change.Actions)
		after, _ := change.Change.After.(map[string]interface{})
		assert.Equal(t, false, after[perm], "removing %s from the configuration must plan it as false", perm)

		terraform.Apply(t, terraformOptions)
		id := itemID(t, item)
		gp, err := api.GetItem[api.GroupPolicy](freshClient(t), &id)
		require.NoError(t, err)
		require.NotNil(t, granted(gp), "the appliance did not return %s", perm)
		assert.False(t, *granted(gp), "the appliance still grants %s after it was removed from the configuration", perm)
		requireNoResourceChanges(t, terraformOptions, "a revoked permission must settle in one apply")
	})
}
