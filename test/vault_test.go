package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func TestVaultSSHKey(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/vault/ssh_account")

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := withBaseTFOptions(t, &terraform.Options{
			TerraformDir: testFolder,
			Vars: map[string]interface{}{
				"random_bits": randomBits,
				"name":        "This is a Name",
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.InitAndApply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Test Vault SSH Account Creation", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "item_list")
		group := terraform.OutputMap(t, terraformOptions, "group")
		gList := terraform.OutputListOfObjects(t, terraformOptions, "group_list")

		assert.Equal(t, randomBits, item["username"])
		assert.Equal(t, 0, len(list))
		assert.Equal(t, randomBits, group["description"])
		assert.Equal(t, 0, len(gList))
	})

	test_structure.RunTestStage(t, "Test finding the new SSH account with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "item_list")
		group := terraform.OutputMap(t, terraformOptions, "group")
		gList := terraform.OutputListOfObjects(t, terraformOptions, "group_list")

		assert.Equal(t, 1, len(list))
		if len(list) > 0 {
			assert.Equal(t, item["id"], list[0]["id"])
		}

		assert.Equal(t, 1, len(gList))
		if len(list) > 0 {
			assert.Equal(t, group["id"], gList[0]["id"])
		}
	})
}
