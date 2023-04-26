package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func TestJumpointAndJumpGroup(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/jumpoint_and_jump_group")

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
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Test Jumpoint/Jump Group Creation", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		jpItem := terraform.OutputMap(t, terraformOptions, "jumpoint")
		jgItem := terraform.OutputMap(t, terraformOptions, "jump_group")
		jpList := terraform.OutputListOfObjects(t, terraformOptions, "jumpoint_list")
		jgList := terraform.OutputListOfObjects(t, terraformOptions, "jump_group_list")

		codeName := fmt.Sprintf("example_%s", randomBits)

		assert.Equal(t, codeName, jpItem["code_name"])
		assert.Equal(t, codeName, jgItem["code_name"])
		assert.Equal(t, 0, len(jpList))
		assert.Equal(t, 0, len(jgList))
	})

	test_structure.RunTestStage(t, "Test finding the new Jumpoint/Jump Group items with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		jpItem := terraform.OutputMap(t, terraformOptions, "jumpoint")
		jgItem := terraform.OutputMap(t, terraformOptions, "jump_group")
		jpList := terraform.OutputListOfObjects(t, terraformOptions, "jumpoint_list")
		jgList := terraform.OutputListOfObjects(t, terraformOptions, "jump_group_list")

		codeName := fmt.Sprintf("example_%s", randomBits)

		assert.Equal(t, codeName, jpItem["code_name"])
		assert.Equal(t, codeName, jgItem["code_name"])

		assert.Equal(t, 1, len(jpList))
		if len(jpList) > 0 {
			assert.Equal(t, jpItem["id"], jpList[0]["id"])
		}
		assert.Equal(t, 1, len(jgList))
		if len(jgList) > 0 {
			assert.Equal(t, jgItem["id"], jgList[0]["id"])
		}
	})
}

func TestShellJump(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/shell_jump")

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
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.InitAndApply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Test Shell Jump Creation", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Test finding the new Shell Jump item with the datasource", func() {
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

func TestRemoteRDP(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/remote_rdp")

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
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Create a new RDP item", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new RDP item with the datasource", func() {
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

func TestRemoteVNC(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/remote_vnc")

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
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Create a new VNC item", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new VNC item with the datasource", func() {
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

func TestProtocolTunnel(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/protocol_tunnel_jump")

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
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Create new Protocol Tunnel items", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		sqlItem := terraform.OutputMap(t, terraformOptions, "sql_item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, sqlItem["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Protocol Tunnel item with the datasource", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		sqlItem := terraform.OutputMap(t, terraformOptions, "sql_item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 2, len(list))
		if len(list) > 0 {
			idMap := map[string]string{
				"tcp":   item["id"],
				"mssql": sqlItem["id"],
			}

			assert.Equal(t, idMap[list[0]["tunnel_type"].(string)], list[0]["id"])
			assert.Equal(t, idMap[list[1]["tunnel_type"].(string)], list[1]["id"])
		}
	})
}
