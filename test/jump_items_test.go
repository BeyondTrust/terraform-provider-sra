package test

import (
	"fmt"
	"terraform-provider-sra/api"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		k8sItem := terraform.OutputMap(t, terraformOptions, "k8s_item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, sqlItem["tag"])
		assert.Equal(t, randomBits, k8sItem["tag"])
		assert.Equal(t, k8sItem["url"], k8sItem["hostname"], "the appliance derives a k8s hostname from url")
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Protocol Tunnel items with the datasource", func() {
		if !mechs.IsPRA() {
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// The plan checks the setup apply: k8s items, and tcp and mssql items with
		// the same config v1.4.0 accepted, round-trip without drift. The apply
		// after it is only so the datasource output finds the new items.
		requireNoResourceChanges(t, terraformOptions, "a protocol tunnel left a diff")
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		sqlItem := terraform.OutputMap(t, terraformOptions, "sql_item")
		k8sItem := terraform.OutputMap(t, terraformOptions, "k8s_item")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 3, len(list))
		idMap := map[string]string{
			"tcp":   item["id"],
			"mssql": sqlItem["id"],
			"k8s":   k8sItem["id"],
		}
		byType := map[string]map[string]interface{}{}
		for _, v := range list {
			byType[v["tunnel_type"].(string)] = v
		}
		for tunnelType, id := range idMap {
			if assert.Contains(t, byType, tunnelType) {
				assert.Equal(t, id, byType[tunnelType]["id"], tunnelType)
			}
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
		assertListContainsIDs(t, list, item["id"], item2["id"])
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
		assertListContainsIDs(t, list, item["id"], item2["id"])
	})
}

func TestMongodbTunnel(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	skipWithoutEndpoint[api.MongodbTunnelJump](t, "26.1.1")
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/mongodb_tunnel_jump", productPath()))

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	vars := map[string]interface{}{
		"random_bits": randomBits,
		"name":        "This is a Name",
		"hostname":    "this.host",
	}

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := withBaseTFOptions(t, &terraform.Options{TerraformDir: testFolder, Vars: vars})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		if mechs.IsPRA() {
			terraform.InitAndApply(t, terraformOptions)
		} else {
			_, err := terraform.InitAndApplyE(t, terraformOptions)
			assert.NotNil(t, err)
		}
	})
	if !mechs.IsPRA() {
		return
	}

	test_structure.RunTestStage(t, "Create new MongoDB Tunnel items", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, item2["tag"])
		assert.Equal(t, "127.0.0.1", item2["tunnel_listen_address"], "the schema default")
		assert.Equal(t, "", item2["auth_database"], "state is empty, not null, on every appliance version")
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new MongoDB Tunnel items with the datasource and plan clean", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// The plan checks the setup apply; the apply after it is only so the
		// datasource output finds the new items.
		requireNoResourceChanges(t, terraformOptions, "a freshly created MongoDB tunnel left a diff")
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 2, len(list))
		assertListContainsIDs(t, list, item["id"], item2["id"])
	})

	// auth_database exists from PRA 26.3.1. Older appliances never return the
	// key and reject it on write, so it is only exercised where it is returned.
	var authDatabaseSupported bool

	test_structure.RunTestStage(t, "Update MongoDB Tunnel items in place, clearing database", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		id := itemID(t, terraform.OutputMap(t, terraformOptions, "item"))

		before, err := api.GetItem[api.MongodbTunnelJump](freshClient(t), &id)
		require.NoError(t, err)
		authDatabaseSupported = before.AuthDatabase != nil
		t.Logf("auth_database supported by this appliance: %v", authDatabaseSupported)

		vars["tunnel_listen_address"] = "127.0.0.9"
		vars["tag_suffix"] = "-updated"
		vars["username"] = "mongouser2"
		vars["database"] = ""
		if authDatabaseSupported {
			vars["auth_database"] = "authdb"
		}
		terraformOptions = resaveOptions(t, testFolder, vars)

		terraform.Apply(t, terraformOptions)
		requireNoResourceChanges(t, terraformOptions, "updating a MongoDB tunnel left a diff")
		assert.Equal(t, id, itemID(t, terraform.OutputMap(t, terraformOptions, "item")), "the update must be in place")

		got, err := api.GetItem[api.MongodbTunnelJump](freshClient(t), &id)
		require.NoError(t, err)
		require.NotNil(t, got.TunnelListenAddress)
		assert.Equal(t, "127.0.0.9", *got.TunnelListenAddress)
		assert.Equal(t, randomBits+"-updated", got.Tag)
		assert.Equal(t, "mongouser2", got.Username)
		assert.Equal(t, "", got.Database, `sending "" must clear database`)
		if authDatabaseSupported {
			require.NotNil(t, got.AuthDatabase)
			assert.Equal(t, "authdb", *got.AuthDatabase)
		}
	})

	test_structure.RunTestStage(t, "Clear auth_database in place", func() {
		if !authDatabaseSupported {
			t.Log("skipped: this appliance predates auth_database (PRA 26.3.1)")
			return
		}
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		id := itemID(t, terraform.OutputMap(t, terraformOptions, "item"))

		vars["auth_database"] = ""
		terraformOptions = resaveOptions(t, testFolder, vars)

		terraform.Apply(t, terraformOptions)
		requireNoResourceChanges(t, terraformOptions, "clearing auth_database left a diff")

		got, err := api.GetItem[api.MongodbTunnelJump](freshClient(t), &id)
		require.NoError(t, err)
		require.NotNil(t, got.AuthDatabase)
		assert.Equal(t, "", *got.AuthDatabase, "the PATCH must have cleared auth_database")
	})
}

func TestOracleTunnel(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	skipWithoutEndpoint[api.OracleTunnelJump](t, "26.3.1")
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/oracle_tunnel_jump", productPath()))

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		terraform.Destroy(t, terraformOptions)
	})

	vars := map[string]interface{}{
		"random_bits": randomBits,
		"name":        "This is a Name",
		"hostname":    "this.host",
	}

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := withBaseTFOptions(t, &terraform.Options{TerraformDir: testFolder, Vars: vars})

		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		if mechs.IsPRA() {
			terraform.InitAndApply(t, terraformOptions)
		} else {
			_, err := terraform.InitAndApplyE(t, terraformOptions)
			assert.NotNil(t, err)
		}
	})
	if !mechs.IsPRA() {
		return
	}

	test_structure.RunTestStage(t, "Create new Oracle Tunnel items", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, randomBits, item["tag"])
		assert.Equal(t, randomBits, item2["tag"])
		assert.Equal(t, "127.0.0.1", item2["tunnel_listen_address"], "the schema default")
		assert.Equal(t, 0, len(list))
	})

	test_structure.RunTestStage(t, "Find the new Oracle Tunnel items with the datasource and plan clean", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// The plan checks the setup apply; the apply after it is only so the
		// datasource output finds the new items.
		requireNoResourceChanges(t, terraformOptions, "a freshly created Oracle tunnel left a diff")
		terraform.Apply(t, terraformOptions)

		item := terraform.OutputMap(t, terraformOptions, "item")
		item2 := terraform.OutputMap(t, terraformOptions, "item_secondary")
		list := terraform.OutputListOfObjects(t, terraformOptions, "list")

		assert.Equal(t, 2, len(list))
		assertListContainsIDs(t, list, item["id"], item2["id"])

		// The list carries gateway_id/asset_group_id on the wire; these must land
		// on the standard attribute names.
		for _, v := range list {
			assert.Equal(t, item["jumpoint_id"], fmt.Sprint(v["jumpoint_id"]))
			assert.Equal(t, item["jump_group_id"], fmt.Sprint(v["jump_group_id"]))
		}
	})

	test_structure.RunTestStage(t, "Update Oracle Tunnel items in place, clearing database", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)
		id := itemID(t, terraform.OutputMap(t, terraformOptions, "item"))

		vars["tunnel_listen_address"] = "127.0.0.9"
		vars["tag_suffix"] = "-updated"
		vars["database"] = ""
		terraformOptions = resaveOptions(t, testFolder, vars)

		terraform.Apply(t, terraformOptions)
		requireNoResourceChanges(t, terraformOptions, "updating an Oracle tunnel left a diff")
		assert.Equal(t, id, itemID(t, terraform.OutputMap(t, terraformOptions, "item")), "the update must be in place")

		got, err := api.GetItem[api.OracleTunnelJump](freshClient(t), &id)
		require.NoError(t, err)
		require.NotNil(t, got.TunnelListenAddress)
		assert.Equal(t, "127.0.0.9", *got.TunnelListenAddress)
		assert.Equal(t, randomBits+"-updated", got.Tag)
		assert.Equal(t, "orauser", got.Username)
		assert.Equal(t, "", got.Database, `sending "" must clear database`)
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
		assertListContainsIDs(t, list, item["id"], item2["id"])
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
