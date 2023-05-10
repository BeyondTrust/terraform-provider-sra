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
		shell := terraform.OutputMap(t, terraformOptions, "shell")

		data := testData{
			randomBits: randomBits,
			shellID:    shell["id"],
		}

		assertAccountGroup(t, terraformOptions, "group", data, true, true)
		assertAccountGroup(t, terraformOptions, "group_nothing", data, false, false)
		assertAccountGroup(t, terraformOptions, "group_gp", data, true, false)
		assertAccountGroup(t, terraformOptions, "group_jia", data, false, true)

		list := terraform.OutputListOfObjects(t, terraformOptions, "list")
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

	test_structure.RunTestStage(t, "Test Vault SSH Accounts", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		group := terraform.OutputMap(t, terraformOptions, "group")
		shell := terraform.OutputMap(t, terraformOptions, "shell")

		data := testData{
			randomBits: randomBits,
			groupID:    group["id"],
			shellID:    shell["id"],
			ssh:        true,
		}

		assertAccount(t, terraformOptions, "item", data, false, false)

		data.groupID = "1"
		assertAccount(t, terraformOptions, "stand_alone", data, false, false)
		assertAccount(t, terraformOptions, "stand_alone_gp", data, true, false)
		assertAccount(t, terraformOptions, "stand_alone_ji", data, false, true)
		assertAccount(t, terraformOptions, "stand_alone_both", data, true, true)

		list := terraform.OutputListOfObjects(t, terraformOptions, "list")
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Test finding the new SSH account with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 1, len(list))
		if len(list) > 0 {
			assert.Equal(t, item["id"], list[0]["id"])
		}
	})
}

func TestVaultUserPass(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/vault/user_pass_account")

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

	test_structure.RunTestStage(t, "Test Vault User/Pass Accounts", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		group := terraform.OutputMap(t, terraformOptions, "group")
		shell := terraform.OutputMap(t, terraformOptions, "shell")

		data := testData{
			randomBits: randomBits,
			groupID:    group["id"],
			shellID:    shell["id"],
			ssh:        false,
		}

		assertAccount(t, terraformOptions, "item", data, false, false)

		data.groupID = "1"
		assertAccount(t, terraformOptions, "stand_alone", data, false, false)
		assertAccount(t, terraformOptions, "stand_alone_gp", data, true, false)
		assertAccount(t, terraformOptions, "stand_alone_ji", data, false, true)
		assertAccount(t, terraformOptions, "stand_alone_both", data, true, true)

		list := terraform.OutputListOfObjects(t, terraformOptions, "list")
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Test finding the new User/Pass account with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 1, len(list))
		if len(list) > 0 {
			assert.Equal(t, item["id"], list[0]["id"])
		}
	})
}

type testData struct {
	randomBits string
	groupID    string
	shellID    string
	ssh        bool
}

func assertAccountGroup(t *testing.T, options *terraform.Options, key string, data testData, assertGP bool, assertJI bool) {
	item := terraform.OutputMap(t, options, key)
	itemJson := terraform.OutputJson(t, options, key)
	parsed, err := gabs.ParseJSON([]byte(itemJson))
	assert.Nil(t, err)
	assert.Equal(t, data.randomBits, item["description"])

	assertExtras(t, options, parsed, data, assertGP, assertJI)
}

func assertAccount(t *testing.T, options *terraform.Options, key string, data testData, assertGP bool, assertJI bool) {
	item := terraform.OutputMap(t, options, key)
	itemJson := terraform.OutputJson(t, options, key)
	parsed, err := gabs.ParseJSON([]byte(itemJson))
	assert.Nil(t, err)
	assertAccountCommonValues(t, item, data.randomBits, data.groupID)
	if data.ssh {
		assert.Equal(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC8QhNX9O8WIN5XmF+Qyqwtc5kkTddgPh77FmDEers1e", item["public_key"])
	}

	assertExtras(t, options, parsed, data, assertGP, assertJI)
}

func assertExtras(t *testing.T, options *terraform.Options, parsed *gabs.Container, data testData, assertGP bool, assertJI bool) {
	if assertGP {
		assertGPMembership(t, parsed)
	} else {
		assertNoGPMembership(t, parsed)
	}
	if assertJI {
		assertJumpItemAssociations(t, parsed, data.randomBits, data.shellID)
	} else {
		assertNoJumpItemAssociations(t, parsed)
	}
}

func assertAccountCommonValues(t *testing.T, item map[string]string, randomBits string, accountGroupID string) {
	assert.Equal(t, randomBits, item["username"])
	assert.Equal(t, accountGroupID, item["account_group_id"])
}

func assertGPMembership(t *testing.T, parsed *gabs.Container) {
	membershipsData, err := parsed.JSONPointer("/group_policy_memberships")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(membershipsData.Data().([]any)))
	membershipsData, err = parsed.JSONPointer("/group_policy_memberships/0")
	assert.Nil(t, err)
	membership := membershipsData.Data().(map[string]any)
	assert.Equal(t, "inject", membership["role"])
}

func assertJumpItemAssociations(t *testing.T, parsed *gabs.Container, randomBits string, shellID string) {
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
	assert.Equal(t, shellID, fmt.Sprintf("%v", item["id"]))
}

func assertNoJumpItemAssociations(t *testing.T, parsed *gabs.Container) {
	membership, err := parsed.JSONPointer("/jump_item_association")
	assert.Nil(t, err)
	assert.Nil(t, membership.Data())
}
