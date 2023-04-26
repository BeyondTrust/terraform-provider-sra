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
	exampleFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/shell_jump")

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: exampleFolder,

			Vars: map[string]interface{}{
				"random_bits": randomBits,
				"name":        "This is a Name",
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, exampleFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))

		terraform.Apply(t, terraformOptions)

		planList := terraform.OutputListOfObjects(t, terraformOptions, "list")
		assert.Equal(t, 1, len(planList))
		if len(planList) > 0 {
			assert.Equal(t, item["id"], planList[0]["id"])
		}
	})
}

func TestRemoteRDP(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	exampleFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/remote_rdp")

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: exampleFolder,

			Vars: map[string]interface{}{
				"random_bits": randomBits,
				"name":        "This is a Name",
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, exampleFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))

		terraform.Apply(t, terraformOptions)

		planList := terraform.OutputListOfObjects(t, terraformOptions, "list")
		assert.Equal(t, 1, len(planList))
		if len(planList) > 0 {
			assert.Equal(t, item["id"], planList[0]["id"])
		}
	})
}

func TestRemoteVNC(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	exampleFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/remote_vnc")

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: exampleFolder,

			Vars: map[string]interface{}{
				"random_bits": randomBits,
				"name":        "This is a Name",
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, exampleFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))

		terraform.Apply(t, terraformOptions)

		planList := terraform.OutputListOfObjects(t, terraformOptions, "list")
		assert.Equal(t, 1, len(planList))
		if len(planList) > 0 {
			assert.Equal(t, item["id"], planList[0]["id"])
		}
	})
}

func TestProtocolTunnel(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom()
	exampleFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "test-tf-files/jump_items/protocol_tunnel_jump")

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: exampleFolder,

			Vars: map[string]interface{}{
				"random_bits": randomBits,
				"name":        "This is a Name",
				"hostname":    "this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, exampleFolder, terraformOptions)
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, exampleFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		sqlItem := terraform.OutputMap(t, terraformOptions, "sql_item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, sqlItem["tag"])
		assert.Equal(t, 0, len(list))

		terraform.Apply(t, terraformOptions)

		planList := terraform.OutputListOfObjects(t, terraformOptions, "list")
		assert.Equal(t, 2, len(planList))
		if len(planList) > 0 {
			idMap := map[string]string{
				"tcp":   item["id"],
				"mssql": sqlItem["id"],
			}

			assert.Equal(t, idMap[planList[0]["tunnel_type"].(string)], planList[0]["id"])
			assert.Equal(t, idMap[planList[1]["tunnel_type"].(string)], planList[1]["id"])
		}
	})
}
