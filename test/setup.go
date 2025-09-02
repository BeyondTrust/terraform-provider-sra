package test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"terraform-provider-sra/api"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/joho/godotenv"
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
		t.Logf("🚀 Running tests against [%s]", os.Getenv("BT_API_HOST"))
		client, _ = api.NewClient(os.Getenv("BT_API_HOST"), &client_id, &client_secret)
		client.SetTest(t)

		mechs, _ = api.Get[api.MechList](client)
		t.Logf("Got mechs [%+v]", mechs)
	}

	randomBits := strings.ToLower(random.UniqueId())

	if test_structure.SkipStageEnvVarSet() {
		randomBits = "not_so_random"
	}

	return randomBits
}

// Load .env file (if present) when the test package is initialized. This helps when running
// tests from editors or CI where the environment file isn't automatically sourced.
func init() {
	// Try to find repository root (where go.mod or .git lives) and load .env from there.
	if root, err := findRepoRoot(); err == nil {
		envPath := filepath.Join(root, ".env")
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded .env from repo root: %s", envPath)
			return
		}
		log.Printf(".env not found at repo root (%s): %v", envPath, err)
	} else {
		log.Printf("could not determine repo root: %v", err)
	}

	// Fallback: try to load .env from current working directory
	if err := godotenv.Load(); err == nil {
		log.Printf("Loaded .env from working directory")
	}
}

// findRepoRoot walks upward from this file's directory to find the repo root by
// locating a `go.mod` file or a `.git` directory. Returns an error if not found.
func findRepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get caller info")
	}

	dir := filepath.Dir(filename)
	for {
		// check for go.mod
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		// check for .git directory
		if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && fi.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("repo root not found")
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
	if newOpts.EnvVars == nil {
		newOpts.EnvVars = map[string]string{}
	}
	newOpts.EnvVars["BT_API_HOST"] = os.Getenv("BT_API_HOST")
	newOpts.EnvVars["BT_CLIENT_ID"] = os.Getenv("BT_CLIENT_ID")
	newOpts.EnvVars["BT_CLIENT_SECRET"] = os.Getenv("BT_CLIENT_SECRET")
	newOpts.MaxRetries = 0
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
