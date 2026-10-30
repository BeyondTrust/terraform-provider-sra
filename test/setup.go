package test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"terraform-provider-sra/api"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mechs is fetched once per test binary: the appliance's product never changes
// mid-run. The client that fetched it is not kept (see freshClient).
var mechs *api.MechList

// stageSkipRandomBits is substituted for the usual random suffix whenever any
// SKIP_<stage> env var is set, so cached fixture state keeps naming the same
// objects between runs.
//
// The orphan guard in import_test.go treats it as NOT establishing ownership: it
// is shared by every test in such a run, so an object carrying it may belong to
// any earlier run. Declared here, beside its only producer, so the two cannot
// drift apart.
const stageSkipRandomBits = "not_so_random"

func setEnvAndGetRandom(t *testing.T) string {
	// os.Setenv("SKIP_setup", "true")
	// os.Setenv("SKIP_teardown", "true")

	if mechs == nil {
		t.Logf("🚀 Running tests against [%s]", os.Getenv("BT_API_HOST"))
		client, err := freshClientE(t)
		if err != nil {
			t.Fatalf("could not create API client for BT_API_HOST [%s]: %v — check BT_API_HOST/BT_CLIENT_ID/BT_CLIENT_SECRET", os.Getenv("BT_API_HOST"), err)
		}

		mechs, err = api.Get[api.MechList](client)
		if err != nil {
			t.Fatalf("could not fetch mechs from BT_API_HOST [%s]: %v", os.Getenv("BT_API_HOST"), err)
		}
		t.Logf("Got mechs [%+v]", mechs)
	}

	randomBits := strings.ToLower(random.UniqueId())

	if test_structure.SkipStageEnvVarSet() {
		randomBits = stageSkipRandomBits
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
	// Keep BT_* credentials out of EnvVars. terraform inherits this process's
	// environment anyway (terratest's shell.formatEnvVars starts from
	// os.Environ), and SaveTerraformOptions logs the options and writes them in
	// plaintext to a 0644 .test-data/TerraformOptions.json.
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

// assertSoleMembership asserts exactly one membership, pointing at the expected
// group policy. It deliberately checks group_policy_id as well as role: the
// existing assertGPMembership checks only the role, so a membership recorded
// against the WRONG group policy would satisfy it -- and that is precisely the
// corruption this test exists to catch.
func assertSoleMembership(t *testing.T, parsed *gabs.Container, wantGroupPolicyID, msg string) {
	t.Helper()

	members, err := parsed.JSONPointer("/group_policy_memberships")
	require.NoError(t, err)
	require.NotNil(t, members.Data(), msg)
	require.Len(t, members.Data().([]any), 1, msg)

	entry, err := parsed.JSONPointer("/group_policy_memberships/0")
	require.NoError(t, err)
	membership := entry.Data().(map[string]any)
	assert.Equal(t, wantGroupPolicyID, membership["group_policy_id"],
		"the membership must reference the group policy from the config")
	assert.Equal(t, "inject", membership["role"])
}

// freshClient builds a new API client for each use. Never cache one across a
// terraform command.
//
// This is load-bearing, not defensive, and the mechanism was measured rather
// than guessed (against mpam, 2026-09-15):
//
//   - The appliance retains only a bounded number of concurrent tokens per client
//     credential and evicts the OLDEST when that bound is exceeded. Probing with
//     repeated mints, the first token kept working through 30 further mints and
//     returned 401 on the 31st.
//   - Tokens carry expires_in=3600, so oauth2's ReuseTokenSource considers one
//     valid for an hour and will not re-mint. Eviction is server-side and
//     structurally invisible to it (it refreshes only on expiry).
//   - Every terraform command spawns a provider subprocess that mints its own
//     token with the same credentials, so a full suite run burns through the
//     bound quickly.
//
// Net effect: a cached client's token is evicted partway through a run
// while oauth2 still believes it valid, and the next call returns
// "status: 401 ... Access token is invalid". Observed, not theorised -- this test
// passed when run alone and failed when run fourth.
func freshClient(t *testing.T) *api.APIClient {
	t.Helper()

	c, err := freshClientE(t)
	require.NoError(t, err, "could not build a fresh API client")
	return c
}

// freshClientE is freshClient without the t.FailNow. The orphan cleanup needs it:
// api.NewClient is not a pure constructor -- it mints a token and can fail -- and
// a require inside t.Cleanup calls FailNow, which Goexits the cleanup goroutine
// before any of the messages naming the leaked id can print. The one moment the
// guard exists for is the moment it must not lose its diagnostic.
func freshClientE(t *testing.T) (*api.APIClient, error) {
	t.Helper()

	id := os.Getenv("BT_CLIENT_ID")
	secret := os.Getenv("BT_CLIENT_SECRET")
	c, err := api.NewClient(os.Getenv("BT_API_HOST"), &id, &secret)
	if err != nil {
		return nil, err
	}

	// Without a logger, doRequest's request/response tracing is gated off and
	// these calls run silently -- so an "orphan recovery FAILED ... LEAKED"
	// message would arrive with no URL or response body to act on, exactly when
	// a human needs them.
	c.SetTestLogger(t)
	return c, nil
}

// skipWithoutEndpoint skips a PRA test when the appliance has no endpoint for
// item type T, because it predates minVersion. The appliance has no fallback
// route, so a missing endpoint answers 404; any other error fails the test. On
// RS it does nothing: the test still asserts the PRA-only error there.
//
// The probe filters on a name nothing has, so only the status matters, not the
// items already on the appliance. T's endpoint must accept a name filter: an
// unknown filter key is a 400.
func skipWithoutEndpoint[T api.APIResource](t *testing.T, minVersion string) {
	t.Helper()
	if !mechs.IsPRA() {
		return
	}
	var tmp T
	_, err := api.ListItems[T](freshClient(t), map[string]string{"name": "probe-" + random.UniqueId()})
	if api.IsNotFound(err) {
		t.Skipf("%s requires PRA %s or later; this appliance has no such endpoint", tmp.Endpoint(), minVersion)
	}
	require.NoError(t, err, "probing %s", tmp.Endpoint())
}

// requireNoResourceChanges plans options and requires every resource change to
// be a no-op. Call it straight after the apply under test, with nothing applied
// in between, so a diff that one more apply would absorb still fails.
//
// Output changes are ignored on purpose: a list datasource with no dependency
// on the resources is read before they change, so its output lags one apply and
// the plan after any apply that changed them shows "Changes to Outputs".
func requireNoResourceChanges(t *testing.T, options *terraform.Options, msg string) {
	t.Helper()
	planOptions := *options
	planOptions.PlanFilePath = filepath.Join(t.TempDir(), "plan.tfplan")
	terraform.PlanContext(t, t.Context(), &planOptions)
	// The plan output above already shows any diff; the JSON adds only noise.
	planOptions.Logger = logger.Discard
	plan := terraform.ShowWithStructContext(t, t.Context(), &planOptions)

	var changed []string
	for address, rc := range plan.ResourceChangesMap {
		if rc.Change == nil || !rc.Change.Actions.NoOp() {
			changed = append(changed, address)
		}
	}
	require.Empty(t, changed, "%s: these resources plan a change", msg)
}

// itemID parses the id attribute of a resource output.
func itemID(t *testing.T, item map[string]string) int {
	t.Helper()
	id, err := strconv.Atoi(item["id"])
	require.NoError(t, err)
	return id
}

// assertListContainsIDs asserts that every id is the id of an item in a list
// datasource output.
func assertListContainsIDs(t *testing.T, list []map[string]interface{}, ids ...string) {
	t.Helper()
	listed := map[string]bool{}
	for _, v := range list {
		if id, ok := v["id"].(string); ok {
			listed[id] = true
		}
	}
	for _, id := range ids {
		assert.True(t, listed[id], "item %s is not in the list", id)
	}
}

// resaveOptions rebuilds the options from vars and saves them, so later stages
// load the changed variables.
func resaveOptions(t *testing.T, testFolder string, vars map[string]interface{}) *terraform.Options {
	t.Helper()
	options := withBaseTFOptions(t, &terraform.Options{TerraformDir: testFolder, Vars: vars})
	test_structure.SaveTerraformOptions(t, testFolder, options)
	return options
}
