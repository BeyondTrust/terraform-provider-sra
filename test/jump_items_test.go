package test

import (
	"fmt"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/stretchr/testify/assert"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func TestJumpointAndJumpGroup(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/jumpoint_and_jump_group", productPath()))

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

	test_structure.RunTestStage(t, "Test Jumpoint/Jump Group Creation", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		jpItem := terraform.OutputMap(t, terraformOptions, "jumpoint")
		jgItem := terraform.OutputMap(t, terraformOptions, "jump_group")

		codeName := fmt.Sprintf("example_%s", randomBits)

		assert.Equal(t, codeName, jpItem["code_name"])
		assert.Equal(t, codeName, jgItem["code_name"])

		assertNoGPMembership(t, extractJson(t, terraformOptions, "jumpoint"))
		assertNoGPMembership(t, extractJson(t, terraformOptions, "jump_group"))

		gp := terraform.OutputMap(t, terraformOptions, "gp")
		jp := terraform.OutputMap(t, terraformOptions, "jp")
		jir := terraform.OutputMap(t, terraformOptions, "jir")

		assertJPGPMembership(t, extractJson(t, terraformOptions, "jumpoint_gp"), gp["id"])
		assertJGGPMembership(t, extractJson(t, terraformOptions, "jump_group_gp"), gp["id"], jp["id"], jir["id"])

		jpList := terraform.OutputListOfObjects(t, terraformOptions, "jumpoint_list")
		jgList := terraform.OutputListOfObjects(t, terraformOptions, "jump_group_list")
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

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/shell_jump", productPath()))

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

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/remote_rdp", productPath()))

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

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/remote_vnc", productPath()))

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

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/protocol_tunnel_jump", productPath()))

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
		_, err := terraform.InitAndApplyE(t, terraformOptions)
		if mechs.IsPRA() {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	})

	test_structure.RunTestStage(t, "Create new Protocol Tunnel items", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		sqlItem := terraform.OutputMap(t, terraformOptions, "sql_item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, sqlItem["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Protocol Tunnel items with the datasource", func() {
		if !mechs.IsPRA() {
			return
		}
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

func TestPostgresTunnel(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/postgresql_tunnel_jump", productPath()))

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
		if mechs.IsPRA() {
			terraform.InitAndApply(t, terraformOptions)
		} else {
			// Expect non-PRA to error on PRA-only resources
			_, err := terraform.InitAndApplyE(t, terraformOptions)
			assert.NotNil(t, err)
		}
	})

	test_structure.RunTestStage(t, "Create new Postgres Tunnel items", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, item2["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Postgres Tunnel items with the datasource", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Need to re-run apply so that the datasource output finds the new item
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 2, len(list))
		if len(list) > 0 {
			// Ensure both items are present by comparing ids
			ids := map[string]bool{item["id"]: true, item2["id"]: true}
			found := 0
			for _, v := range list {
				if _, ok := ids[v["id"].(string)]; ok {
					found++
				}
			}
			assert.Equal(t, 2, found)
		}
	})
}

func TestMySQLTunnel(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/mysql_tunnel_jump", productPath()))

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
		if mechs.IsPRA() {
			terraform.InitAndApply(t, terraformOptions)
		} else {
			_, err := terraform.InitAndApplyE(t, terraformOptions)
			assert.NotNil(t, err)
		}
	})

	test_structure.RunTestStage(t, "Create new MySQL Tunnel items", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, item2["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new MySQL Tunnel items with the datasource", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 2, len(list))
		if len(list) > 0 {
			ids := map[string]bool{item["id"]: true, item2["id"]: true}
			found := 0
			for _, v := range list {
				if _, ok := ids[v["id"].(string)]; ok {
					found++
				}
			}
			assert.Equal(t, 2, found)
		}
	})
}

func TestNetworkTunnel(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/network_tunnel_jump", productPath()))

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
		if mechs.IsPRA() {
			terraform.InitAndApply(t, terraformOptions)
		} else {
			_, err := terraform.InitAndApplyE(t, terraformOptions)
			assert.NotNil(t, err)
		}
	})

	test_structure.RunTestStage(t, "Create new Network Tunnel items", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, item2["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Network Tunnel items with the datasource", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 2, len(list))
		if len(list) > 0 {
			ids := map[string]bool{item["id"]: true, item2["id"]: true}
			found := 0
			for _, v := range list {
				if _, ok := ids[v["id"].(string)]; ok {
					found++
				}
			}
			assert.Equal(t, 2, found)
		}
	})
}

func TestWebJump(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/web_jump", productPath()))

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
				"url":         "https://this.host",
			},
		})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		_, err := terraform.InitAndApplyE(t, terraformOptions)
		if mechs.IsPRA() {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	})

	test_structure.RunTestStage(t, "Create a new Web Jump item", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Web Jump item with the datasource", func() {
		if !mechs.IsPRA() {
			return
		}
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

func TestJumpClientInstaller(t *testing.T) {
	// t.Parallel()

	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/jump_client_installer", productPath()))

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

	test_structure.RunTestStage(t, "Create a new Jump Client Installer", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Jump Client installer with the datasource", func() {
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

func assertJPGPMembership(t *testing.T, parsed *gabs.Container, gpID string) {
	membershipsData, err := parsed.JSONPointer("/group_policy_memberships")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(membershipsData.Data().([]any)))
	membershipsData, err = parsed.JSONPointer("/group_policy_memberships/0")
	assert.Nil(t, err)
	membership := membershipsData.Data().(map[string]any)
	assert.Equal(t, gpID, membership["group_policy_id"])
}
func assertJGGPMembership(t *testing.T, parsed *gabs.Container, gpID string, jpID string, jirID string) {
	membershipsData, err := parsed.JSONPointer("/group_policy_memberships")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(membershipsData.Data().([]any)))
	membershipsData, err = parsed.JSONPointer("/group_policy_memberships/0")
	assert.Nil(t, err)
	membership := membershipsData.Data().(map[string]any)
	assert.Equal(t, gpID, membership["group_policy_id"])
	if mechs.IsPRA() {
		assert.Equal(t, jpID, fmt.Sprintf("%v", membership["jump_policy_id"]))
	}
	assert.Equal(t, jirID, fmt.Sprintf("%v", membership["jump_item_role_id"]))
}
