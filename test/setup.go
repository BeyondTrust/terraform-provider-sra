package test

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"terraform-provider-sra/api"
	"testing"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadMechs fetches the appliance's mech list once per test binary: the
// product never changes mid-run. sync.OnceValues makes the first fetch safe
// under t.Parallel and keeps its error for every later caller. The client that
// fetched it is not kept (see freshClient).
var loadMechs = sync.OnceValues(func() (*api.MechList, error) {
	host := os.Getenv("BT_API_HOST")
	id, secret := os.Getenv("BT_CLIENT_ID"), os.Getenv("BT_CLIENT_SECRET")
	client, err := api.NewClient(host, &id, &secret)
	if err != nil {
		return nil, fmt.Errorf("could not create API client for BT_API_HOST [%s]: %w — check BT_API_HOST/BT_CLIENT_ID/BT_CLIENT_SECRET", host, err)
	}
	m, err := api.Get[api.MechList](client)
	if err != nil {
		return nil, fmt.Errorf("could not fetch mechs from BT_API_HOST [%s]: %w", host, err)
	}
	return m, nil
})

// mechs returns the appliance's mech list, failing the test if it can't be
// fetched.
func mechs(t *testing.T) *api.MechList {
	t.Helper()
	m, err := loadMechs()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// parallel runs the calling E2E test in parallel with the others, except under
// SKIP_<stage>. Stage-skipping runs share the source fixture folders (no temp
// copy is made) and the not_so_random marker, and three fixture folders are each
// used by two tests, so those tests would trample each other's state.
//
// Do not introduce t.Setenv in this package: it panics in a test that has
// called t.Parallel, and changes the environment of every test running beside
// it.
func parallel(t *testing.T) {
	t.Helper()
	if !test_structure.SkipStageEnvVarSet() {
		t.Parallel()
	}
}

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

	t.Logf("🚀 Running tests against [%s], product %s", os.Getenv("BT_API_HOST"), mechs(t).Product)

	randomBits := newMarker(time.Now())

	if test_structure.SkipStageEnvVarSet() {
		randomBits = stageSkipRandomBits
	}

	return randomBits
}

// markerUsable reports whether randomBits can establish ownership at all. An empty
// marker makes a Contains check unconditionally true, and the stage-skip
// constant is shared by every test in such a run (see setEnvAndGetRandom), so it
// also matches objects left behind by earlier runs.
func markerUsable(randomBits string) bool {
	return randomBits != "" && randomBits != stageSkipRandomBits
}

// A marker is the random_bits every fixture object is named, tagged or
// code-named with: "e2e", then the creation time as 6 zero-padded base36 digits
// of Unix minutes, then 6 random [0-9a-z] characters. 15 characters; the longest
// field built from one is code_name "gp_example_<marker>2", 28 characters,
// within the appliance's 64.
//
// The timestamp is what lets the orphan sweeper (sweep.go) tell an abandoned
// object from a live run's: no fixture type has a creation timestamp of its own.
// The random part comes from crypto/rand. terratest's random.UniqueId re-seeds
// from UnixNano on every call, so parallel tests starting in the same instant
// could draw the same value.
const (
	markerPrefix      = "e2e"
	markerTimeDigits  = 6
	markerRandomChars = 6
	markerAlphabet    = "0123456789abcdefghijklmnopqrstuvwxyz"
	markerLen         = len(markerPrefix) + markerTimeDigits + markerRandomChars
)

func newMarker(now time.Time) string {
	minutes := strconv.FormatInt(now.Unix()/60, 36)
	b := []byte(markerPrefix + strings.Repeat("0", markerTimeDigits-len(minutes)) + minutes)

	// Rejection sampling keeps every character equally likely: 252 is the
	// largest multiple of 36 that fits in a byte.
	buf := make([]byte, 1)
	for len(b) < markerLen {
		if _, err := rand.Read(buf); err != nil {
			panic(fmt.Sprintf("crypto/rand: %v", err))
		}
		if buf[0] < 252 {
			b = append(b, markerAlphabet[buf[0]%36])
		}
	}
	return string(b)
}

type markerKind int

const (
	markerUnusable markerKind = iota
	// markerLegacy is the 6-character UniqueId marker used before timestamps.
	// Nothing about it says when it was made.
	markerLegacy
	markerTimestamped
)

var (
	legacyMarkerRE      = regexp.MustCompile(`^[0-9a-z]{6}$`)
	timestampedMarkerRE = regexp.MustCompile(`^e2e[0-9a-z]{12}$`)
)

// parseMarker classifies s, splitting off the single trailing "2" that the
// vault/secret fixture appends when it passes "${var.random_bits}2" down to the
// ssh_account module. base is the marker without it. For a timestamped marker,
// at is its creation time, to the minute.
func parseMarker(s string) (kind markerKind, base, suffix string, at time.Time) {
	if !markerUsable(s) {
		return markerUnusable, "", "", time.Time{}
	}
	base = s
	if (len(s) == markerLen+1 || len(s) == 7) && strings.HasSuffix(s, "2") {
		base, suffix = s[:len(s)-1], "2"
	}
	switch {
	case timestampedMarkerRE.MatchString(base):
		minutes, err := strconv.ParseInt(base[len(markerPrefix):len(markerPrefix)+markerTimeDigits], 36, 64)
		if err != nil {
			return markerUnusable, "", "", time.Time{}
		}
		return markerTimestamped, base, suffix, time.Unix(minutes*60, 0)
	case legacyMarkerRE.MatchString(base):
		return markerLegacy, base, suffix, time.Time{}
	}
	return markerUnusable, "", "", time.Time{}
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

func productPath(t *testing.T) string {
	t.Helper()
	if mechs(t).IsRS() {
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
	if !mechs(t).IsPRA() {
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
