package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Jeffail/gabs"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func TestAccountGroupKey(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/vault/account_group")

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

	test_structure.RunTestStage(t, "Test Vault Account Group Creation", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		group := terraform.OutputMap(t, terraformOptions, "group")
		shell := terraform.OutputMap(t, terraformOptions, "shell")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, group["description"])

		groupJson := terraform.OutputJson(t, terraformOptions, "group")
		parsed, err := gabs.ParseJSON([]byte(groupJson))
		assert.Nil(t, err)

		membershipsData, exists := parsed.JSONPointer("/group_policy_memberships")
		assert.Nil(t, exists)
		assert.Equal(t, 1, len(membershipsData.Data().([]any)))
		membershipsData, exists = parsed.JSONPointer("/group_policy_memberships/0")
		assert.Nil(t, exists)
		membership := membershipsData.Data().(map[string]any)
		assert.Equal(t, "inject", membership["role"])

		filterType, ok := parsed.Path("jump_item_association.filter_type").Data().(string)
		assert.True(t, ok)
		assert.Equal(t, "criteria", filterType)
		tagData, err := parsed.JSONPointer("/jump_item_association/criteria/tag/0")
		assert.Nil(t, err)
		tag, ok := tagData.Data().(string)
		assert.True(t, ok)
		assert.Equal(t, randomBits, tag)

		items, ok := parsed.Path("jump_item_association.jump_items").Data().([]any)
		assert.True(t, ok)
		assert.Equal(t, 1, len(items))

		itemData, err := parsed.JSONPointer("/jump_item_association/jump_items/0")
		assert.Nil(t, err)
		item, ok := itemData.Data().(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "shell_jump", item["type"])
		assert.Equal(t, shell["id"], fmt.Sprintf("%v", item["id"]))

		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Test finding the new SSH account with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		group := terraform.OutputMap(t, terraformOptions, "group")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 1, len(list))
		if len(list) > 0 {
			assert.Equal(t, group["id"], list[0]["id"])
		}
	})
}

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

	test_structure.RunTestStage(t, "Test Vault SSH Account with Account Group", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")
		group := terraform.OutputMap(t, terraformOptions, "group")
		groupID := group["id"]

		assertSSHCommonValues(t, item, randomBits, &groupID)
		assert.Equal(t, 0, len(list))

		item = terraform.OutputMap(t, terraformOptions, "stand_alone")
		groupID = ""
		assertSSHCommonValues(t, item, randomBits, &groupID)
	})

	test_structure.RunTestStage(t, "Test finding the new SSH account with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "item_list")

		assert.Equal(t, 1, len(list))
		if len(list) > 0 {
			assert.Equal(t, item["id"], list[0]["id"])
		}
	})
}

func assertSSHCommonValues(t *testing.T, sshItem map[string]string, randomBits string, accountGroupID *string) {
	assert.Equal(t, randomBits, sshItem["username"])
	assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC8QhNX9O8WIN5XmF+Qyqwtc5kkTddgPh77FmDEers1e", sshItem["public_key"])
	if accountGroupID == nil {
		assert.Nil(t, sshItem["account_group_id"])
	} else {
		assert.Equal(t, *accountGroupID, sshItem["account_group_id"])
	}
}
