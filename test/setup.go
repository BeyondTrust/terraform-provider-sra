package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
)

func setEnvAndGetRandom() string {
	// os.Setenv("SKIP_setup", "true")
	// os.Setenv("SKIP_teardown", "true")

	randomBits := strings.ToLower(random.UniqueId())

	if test_structure.SkipStageEnvVarSet() {
		randomBits = "not_so_random"
	}

	return randomBits
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
