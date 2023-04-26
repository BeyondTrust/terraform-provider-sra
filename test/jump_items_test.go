package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func TestShellJump(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/shell_jump")

	defer test_structure.RunTestStage(t, "Shell Jump teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Sell Jump setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
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

	defer test_structure.RunTestStage(t, "RDP teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "RDP setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
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

	defer test_structure.RunTestStage(t, "VNC teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "VNC setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
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

	defer test_structure.RunTestStage(t, "Protocol Tunnel teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Protocol Tunnel setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
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
