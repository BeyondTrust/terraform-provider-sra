package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func setEnvAndGetRandom() string {
	// os.Setenv("SKIP_setup", "true")
	os.Setenv("SKIP_teardown", "true")

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
