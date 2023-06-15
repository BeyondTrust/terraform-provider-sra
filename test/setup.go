package test

import (
	"os"
	"path/filepath"
	"strings"
	"terraform-provider-sra/api"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
)

var (
	client *api.APIClient = nil
	mechs  *api.MechList  = nil
)

func setEnvAndGetRandom(t *testing.T) string {
	// os.Setenv("SKIP_setup", "true")
	// os.Setenv("SKIP_teardown", "true")

	if client == nil {
		client_id := os.Getenv("BT_CLIENT_ID")
		client_secret := os.Getenv("BT_CLIENT_SECRET")
		client, _ = api.NewClient(os.Getenv("BT_API_HOST"), &client_id, &client_secret)

		mechs, _ = api.Get[api.MechList](client)
		t.Logf("Got mechs [%+v]", mechs)
	}

	randomBits := strings.ToLower(random.UniqueId())

	if test_structure.SkipStageEnvVarSet() {
		randomBits = "not_so_random"
	}

	return randomBits
}

func productPath() string {
	if mechs.IsRS() {
		return "rs"
	} else {
		return "pra"
	}
}

func withBaseTFOptions(t *testing.T, originalOptions *terraform.Options) *terraform.Options {
	newOpts := terraform.WithDefaultRetryableErrors(t, originalOptions)
	pluginPath, _ := filepath.Abs("../test-reg")
	newOpts.PluginDir = pluginPath
	return newOpts
}

func extractJson(t *testing.T, options *terraform.Options, key string) *gabs.Container {
	itemJson := terraform.OutputJson(t, options, key)
	parsed, err := gabs.ParseJSON([]byte(itemJson))
	assert.Nil(t, err)

	return parsed
}

func assertNoGPMembership(t *testing.T, parsed *gabs.Container) {
	membership, err := parsed.JSONPointer("/group_policy_memberships")
	assert.Nil(t, err)
	assert.Nil(t, membership.Data())
}
