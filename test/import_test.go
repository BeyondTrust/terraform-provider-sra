package test

// End-to-end coverage for the two lifecycle phases this suite never entered:
// `terraform import` (previously zero coverage) and, in
// account_group_update_test.go, a genuine config-change update.
//
// WHAT THIS DOES NOT CATCH -- stated so it is not overestimated later:
//
//   - A jump-item-association PATCH that returns 200 echoing the request but does
//     not persist. Update writes the API *response* into state
//     (bt/rs/vault_account_group.go:196), and nothing here reads the association
//     back out of band -- the existence check below fetches the account group, not
//     .../jump-item-association. State and plan agree, the apply succeeds, and the
//     appliance is wrong. Closing this needs a direct read of the sub-resource.
//   - A PATCH writing wrong field values IS caught, but incidentally: by Terraform
//     core's inconsistent-result check, because the fixture declares filter_type,
//     criteria.tag and jump_items explicitly, so those planned values are known.
//     That coverage disappears for any field the fixture stops declaring.
//   - An import populating a field incorrectly is caught for every attribute the
//     fixture declares (a wrong value is a diff), but NOT for unset
//     Optional+Computed attributes -- which is why the account-policy case pins
//     maximum_password_age explicitly.
//
// All of this was verified against a PRA appliance. The rs/ fixtures are mirrored
// from the pra/ ones and compile-checked but have not been executed locally;
// CI's terratest_rs job is the first thing that runs them.

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"terraform-provider-sra/api"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/require"
)

// safeToReclaim encodes the whole safety rule for orphan recovery: may this object
// be deleted as this run's orphan?
//
// It is a separate pure function so it can be exercised directly. The guard's
// branches otherwise run only when a `terraform import` fails, so a green suite
// never executes them -- leaving the one path that deletes real objects from a
// shared appliance as the only unverified logic in the file.
//
// The rule is split in two because the guard applies it at two points: the marker
// is checked before any network call, and ownership only after the object is read.
//
// markerUsable reports whether randomBits can establish ownership at all. An empty
// marker makes the Contains check below unconditionally true, and the stage-skip
// constant is shared by every test in such a run (test/setup.go:52), so it also
// matches objects left behind by earlier runs.
func markerUsable(randomBits string) bool {
	return randomBits != "" && randomBits != stageSkipRandomBits
}

// safeToReclaim reports whether blob -- the marshalled object as read back from
// the API -- may be deleted as this run's orphan. It re-checks the marker rather
// than assuming the caller did, so the rule holds wherever it is applied.
func safeToReclaim(blob []byte, randomBits string) bool {
	return markerUsable(randomBits) && strings.Contains(string(blob), randomBits)
}

// importGuard tracks the one piece of state the orphan recovery below needs: the
// id of the object we are about to detach from Terraform state, and whether the
// re-import failed and left it detached.
type importGuard struct {
	addr string // resource address; also the label in recovery messages

	// randomBits is this run's unique naming marker. The guard refuses to delete
	// anything that does not carry it -- see the ownership check below.
	randomBits string

	// orphanID is set ONLY by reimport's error branch. Non-empty means "the
	// import failed and this id is detached from state", which is the single
	// signal the cleanup fires on -- so a test that never reaches reimport, or
	// whose import succeeded, cannot trigger recovery.
	orphanID string
}

// guardImportOrphan registers cleanup that reclaims an object stranded by a failed
// `terraform import`.
//
// Between `terraform state rm` and a successful `terraform import` the object
// exists on the appliance with no state entry at all, so the deferred
// `terraform.Destroy` cannot see it and a failure there leaks a real object on
// every run. That window is why these tests recover explicitly rather than
// relying on Destroy.
//
// Ordering is deliberately not load-bearing. A raw `defer` in a test function
// always completes before any t.Cleanup registered in that same function, so
// Destroy runs first -- but that does not matter here, because `terraform import`
// is atomic: a failed import writes no state at all and does not re-add an
// address that `state rm` removed (verified against plugin-framework v1.19.0).
// So when `failed` is set, the object is genuinely absent from state and Destroy
// cannot have touched it.
//
// 404 is still tolerated rather than treated as an error, because it is the
// honest success condition for "make sure this object is gone" and costs nothing.
// A 422 (malformed id) or any other error still fails the run: a bad id capture
// is a real defect and must not be swallowed.
func guardImportOrphan[I api.APIResource](t *testing.T, g *importGuard) {
	t.Cleanup(func() {
		if g.orphanID == "" {
			return // clean run: the address is in state and Destroy owns it
		}
		id, convErr := strconv.Atoi(g.orphanID)
		if convErr != nil || id < 1 {
			t.Errorf("orphan recovery: %s has unusable id %q — LEAKED on the appliance", g.addr, g.orphanID)
			return
		}

		// The ownership check below is only as good as the marker. Two ways it can
		// silently become a no-op, both enforced here rather than left to the two
		// distant struct literals that populate the field:
		//
		//  - An empty marker makes strings.Contains unconditionally true, restoring
		//    exactly the blind cross-namespace delete the check exists to prevent.
		//    A guard built as &importGuard{addr: "..."} compiles fine.
		//  - Under stage-skipping, setEnvAndGetRandom pins randomBits to the shared
		//    constant "not_so_random" for every test (test/setup.go:51-52). The
		//    marker then matches leftovers from ANY earlier skip-mode run, not just
		//    this one, so it no longer establishes ownership. Refusing is also the
		//    behaviour stage-skipping wants: SKIP_teardown exists precisely to leave
		//    objects in place between runs, and a human is at the keyboard.
		if !markerUsable(g.randomBits) {
			t.Errorf("orphan recovery: %s has no usable ownership marker (%q), refusing to delete id %d. "+
				"If this is a stage-skipping run, remove it by hand; otherwise the guard was constructed without randomBits.",
				g.addr, g.randomBits, id)
			return
		}

		c, err := freshClientE(t)
		if err != nil {
			t.Errorf("orphan recovery: could not build a client to reclaim %s %d — LEAKED on the appliance: %v", g.addr, id, err)
			return
		}

		// Confirm ownership BEFORE deleting. The type parameter I and the address
		// string are supplied independently by the caller, and DeleteItem derives
		// its path solely from I.Endpoint() -- so a mis-paired case (say
		// sra_jump_group's address with api.VaultAccountGroup) would delete a
		// same-numbered object in a DIFFERENT id namespace, on a shared appliance,
		// and report success. The same applies if an outputKey were ever pointed at
		// a datasource rather than a managed resource. Every fixture names its
		// resources with the run's randomBits, so requiring that marker turns both
		// mistakes into a loud refusal instead of a silent deletion.
		item, err := api.GetItem[I](c, &id)
		if err != nil && !api.IsNotFound(err) {
			// Retry once with a new client. The documented failure here is a 401
			// from token eviction (see freshClient), for which a fresh mint IS the
			// remedy -- and without the retry a single unlucky read costs a real
			// object that the delete below would have reclaimed.
			if retry, rerr := freshClientE(t); rerr == nil {
				c = retry
				item, err = api.GetItem[I](c, &id)
			}
		}
		switch {
		case api.IsNotFound(err):
			return // already gone: destroy reclaimed it, or it was never created
		case err != nil:
			t.Errorf("orphan recovery: could not read %s %d to confirm ownership — LEAKED on the appliance: %v", g.addr, id, err)
			return
		}
		blob, err := json.Marshal(item)
		if err != nil || !safeToReclaim(blob, g.randomBits) {
			t.Errorf("orphan recovery REFUSED for %s %d: object does not carry this run's marker %q, so it is not ours to delete. "+
				"Check the case's API type matches its address. Nothing was deleted; if an object really did leak, remove it by hand.",
				g.addr, id, g.randomBits)
			return
		}

		switch err := api.DeleteItem[I](c, &id); {
		case err == nil:
			t.Logf("orphan recovery: reclaimed %s %d", g.addr, id)
		case api.IsNotFound(err):
			// Raced with something else that removed it. Still the desired end state.
		default:
			t.Errorf("orphan recovery FAILED for %s %d — LEAKED on the appliance: %v", g.addr, id, err)
		}
	})
}

// freshClient builds a new API client rather than reusing the package-level one.
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
// Net effect: the package-level client's token is evicted partway through a run
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
// api.NewClient is not a pure constructor -- it mints a token (api/client.go:105)
// and can fail -- and a require inside t.Cleanup calls FailNow, which Goexits the
// cleanup goroutine before any of the messages naming the leaked id can print. The
// one moment the guard exists for is the moment it must not lose its diagnostic.
func freshClientE(t *testing.T) (*api.APIClient, error) {
	t.Helper()

	id := os.Getenv("BT_CLIENT_ID")
	secret := os.Getenv("BT_CLIENT_SECRET")
	c, err := api.NewClient(os.Getenv("BT_API_HOST"), &id, &secret)
	if err != nil {
		return nil, err
	}

	// Mirror setEnvAndGetRandom (test/setup.go:39). Without a logger, doRequest's
	// request/response tracing is gated off (api/client.go:122) and these calls run
	// silently -- so an "orphan recovery FAILED ... LEAKED" message would arrive
	// with no URL or response body to act on, exactly when a human needs them.
	c.SetTestLogger(t)
	return c, nil
}

// reimport runs `terraform state rm` followed by `terraform import`, recording
// failure on the guard so the cleanup above can reclaim the object.
//
// terraform.FormatArgs is unusable here: it appends flags AFTER the args it is
// given, and Go's flag package stops at the first positional, so
// `import ADDR ID -var k=v` fails with "Import requires two arguments". It would
// also inject -var into `state rm`, which rejects it. Hence the hand-built slices
// with flags first.
//
// -input=false is mandatory: without it a missing variable opens an interactive
// prompt, and an unattended run blocks forever instead of failing.
func reimport(t *testing.T, opts *terraform.Options, g *importGuard, id string) {
	// Keep these two adjacent, with no assertions between them -- every statement
	// here widens the orphan window.
	_, err := terraform.RunTerraformCommandE(t, opts, "state", "rm", g.addr)
	require.NoError(t, err, "state rm failed for %s", g.addr)

	args := append([]string{"import", "-input=false"}, terraform.FormatTerraformVarsAsArgs(opts.Vars)...)
	args = append(args, g.addr, id)
	if _, err := terraform.RunTerraformCommandE(t, opts, args...); err != nil {
		g.orphanID = id // arms the cleanup; nothing else sets this
		require.NoError(t, err, "import failed — %s (id %s) is now orphaned on the appliance", g.addr, id)
	}
}

// TestImportThenApplyAccountGroup is the regression test for the defect that
// motivated this file: sra_vault_account_group's update path POSTed to an
// endpoint exposing only GET and PATCH, which made `terraform import` impossible
// to complete.
//
// It targets new_account_group_jia specifically. Do NOT retarget this at
// new_account_group -- that resource also declares group_policy_memberships,
// which are likewise null in state after import, so the post-import apply would
// call api.CreateItem (a bare POST, no upsert) for a membership that already
// exists. The appliance would 409 -- masking this regression behind an unrelated
// failure -- or silently duplicate the membership. new_account_group_jia has the
// jump item association and no memberships, so it exercises the same PATCH path
// with no duplicate-create surface.
//
// The assertion that matters is that the post-import APPLY succeeds. `terraform
// plan` never invokes the provider's Update -- Update is apply-only -- so a test
// ending at plan could not catch this bug nor fail when it was reintroduced.
func TestImportThenApplyAccountGroup(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	var id string
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/vault/account_group", productPath()))

	guard := &importGuard{addr: "sra_vault_account_group.new_account_group_jia", randomBits: randomBits}
	guardImportOrphan[api.VaultAccountGroup](t, guard)

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraform.Destroy(t, test_structure.LoadTerraformOptions(t, testFolder))
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

	test_structure.RunTestStage(t, "Import the account group and apply over it", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Capture the id BEFORE detaching from state -- once `state rm` has run,
		// this is the only handle left on the object.
		id = terraform.OutputMap(t, terraformOptions, "group_jia")["id"]
		require.NotEmpty(t, id, "could not read the account group id from outputs")

		reimport(t, terraformOptions, guard, id)

		// Guard against this test hollowing out. Its regression value depends
		// entirely on the post-import plan being NON-empty, because only then does
		// the apply below invoke Update. That is true today only because of a
		// pre-existing read gap: readJIA writes jump_item_association back to state
		// only when the pre-refresh value is non-null
		// (bt/rs/vault_account_group.go:263), and after import state holds just id,
		// so the attribute stays null while the config declares it.
		//
		// Close that gap -- a legitimate future fix -- and the plan goes clean, the
		// apply becomes a no-op, Update is never called, and this test would pass
		// green with the POST regression fully restored. That is exactly the vacuity
		// an earlier draft of this work was rejected for. Asserting the plan is
		// dirty converts that silent hollowing into a loud failure that says what to
		// do about it.
		//
		// If you are here because this assertion just failed: the fix is NOT to
		// delete it. Either the import null-gate was closed (good -- rebuild this
		// test around a resource that still diffs after import, since this one no
		// longer exercises Update), or something else made the fixture converge.
		// The gap itself is documented under "Known issues" in CHANGELOG.md.
		require.Equal(t, 2, terraform.PlanExitCode(t, terraformOptions),
			"post-import plan is CLEAN, so the apply below is a no-op and no longer exercises Update. "+
				"If readJIA's import null-gate was fixed, this test must be rebuilt around a resource "+
				"that still diffs after import -- do not simply delete this assertion.")

		// The regression assertion. Pre-fix this routed to CreateItem (POST)
		// against a GET/PATCH-only endpoint and failed; post-fix it PATCHes.
		//
		// Deliberately NOT asserting PlanExitCode == 0 here: a clean plan is
		// unreachable for this resource. readJIA gates its state write on the
		// pre-refresh value being non-null (bt/rs/vault_account_group.go:263) and
		// after import it is null, so jump_item_association stays null while the
		// schema default supplies a non-null value (vault_account_group.go:84).
		// ReadGPMemberships has the equivalent early return on a null state set
		// (bt/rs/gp_membership.go:155). Both are pre-existing and deliberately out
		// of scope here -- fixing them changes refresh behaviour for every existing
		// account group, not just imported ones (documented under "Known issues" in
		// CHANGELOG.md). TestImportRoundTrip covers the clean-plan property on
		// resources that do not have this gap.
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Confirm the account group still exists", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Guards against a silently destructive apply: the step above would also
		// "succeed" if it had deleted and recreated the object.
		// id crosses stages deliberately: this stage compares the pre-import id
		// against the post-apply one, so it cannot be re-derived here. Guard it
		// explicitly, because under SKIP_<stage> a value produced in a skipped
		// stage stays empty and the comparison below would otherwise fail with a
		// message implying the provider replaced the resource.
		require.NotEmpty(t, id, "the account group id was never captured — the import stage did not run "+
			"(SKIP_<stage> set?); this is a harness problem, not a provider one")

		numericID, err := strconv.Atoi(id)
		require.NoError(t, err)
		item, err := api.GetItem[api.VaultAccountGroup](freshClient(t), &numericID)
		require.NoError(t, err, "account group %d no longer exists after the post-import apply", numericID)
		require.NotNil(t, item)

		// And the id must be unchanged -- a destroy/recreate would renumber it.
		require.Equal(t, id, terraform.OutputMap(t, terraformOptions, "group_jia")["id"],
			"the post-import apply replaced the account group instead of updating it")
	})
}

// TestImportRoundTrip asserts that importing a flat resource yields a genuinely
// clean plan -- the property TestImportThenApplyAccountGroup cannot assert,
// because that resource's jump_item_association stays null after import while its
// schema default supplies a value.
//
// The two cases are separate resources rather than one because they cover
// different families: sra_jump_group has no vault surface, and
// sra_vault_account_policy does. Both are "flat" in the sense that matters here
// -- no TF-only sub-resource that CopyAPItoTF leaves null on import.
func TestImportRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		// fixture is relative to test-tf-files/<product>/.
		fixture string
		addr    string
		// outputKey names the output exposing the resource, read as
		// OutputMap(...)["id"] to capture the id before `state rm`.
		outputKey string
		// guard instantiates guardImportOrphan with the right API type. It cannot
		// be a plain type parameter on the struct, so each case supplies it.
		guard func(*testing.T, *importGuard)
		why   string
	}{
		{
			name:      "jump_group",
			fixture:   "jump_items/jumpoint_and_jump_group",
			addr:      "sra_jump_group.example",
			outputKey: "jump_group",
			guard:     guardImportOrphan[api.JumpGroup],
			// .example declares only name and code_name, so state-null equals
			// config-null throughout. Do NOT switch this to .example_gp -- that one
			// declares group_policy_memberships and will diff after import.
			why: "flat: only name and code_name declared",
		},
		{
			name:      "vault_account_policy",
			fixture:   "vault/account_policy",
			addr:      "sra_vault_account_policy.new_account_policy",
			outputKey: "policy",
			guard:     guardImportOrphan[api.VaultAccountPolicy],
			// new_account_policy is the only one of the fixture's three policies
			// that sets maximum_password_age explicitly. That attribute is
			// Optional+Computed with no default, so leaving it unset would make the
			// post-import value depend on the appliance rather than the config.
			why: "flat, and sets every Optional+Computed field explicitly",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			randomBits := setEnvAndGetRandom(t)
			testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/%s", productPath(), tc.fixture))

			guard := &importGuard{addr: tc.addr, randomBits: randomBits}
			tc.guard(t, guard)

			defer test_structure.RunTestStage(t, "teardown", func() {
				terraform.Destroy(t, test_structure.LoadTerraformOptions(t, testFolder))
			})

			test_structure.RunTestStage(t, "setup", func() {
				terraformOptions := withBaseTFOptions(t, &terraform.Options{
					TerraformDir: testFolder,
					Vars:         map[string]interface{}{"random_bits": randomBits},
				})
				test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
				terraform.InitAndApply(t, terraformOptions)

				// Apply twice before asserting anything about plan emptiness. Both
				// fixtures declare list datasources whose outputs are read at plan
				// time, before the resources exist; after one apply those outputs are
				// still empty and the next plan reports "Changes to Outputs", which
				// -detailed-exitcode reports as 2. Every existing test in this suite
				// applies twice for the same reason (see TestAccountGroup).
				terraform.Apply(t, terraformOptions)

				// Baseline the fixture BEFORE touching state. The post-import
				// assertion below covers the whole configuration, not just tc.addr,
				// so without this a pre-existing drift in a sibling resource would
				// surface as "importing <addr> does not round-trip" and point at the
				// wrong thing. Failing here instead says plainly that the fixture was
				// already unstable.
				require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
					"fixture is not stable before import — this is pre-existing drift, not an import defect")
			})

			test_structure.RunTestStage(t, "Round-trip through import", func() {
				terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

				id := terraform.OutputMap(t, terraformOptions, tc.outputKey)["id"]
				require.NotEmpty(t, id, "could not read the %s id from outputs", tc.name)

				reimport(t, terraformOptions, guard, id)

				// Assert == 0, never != 2: GetExitCodeForTerraformCommandContextE
				// returns 1 on a plan *error*, so != 2 would pass on a broken plan.
				//
				// Note this covers the WHOLE configuration, not just tc.addr. The
				// fixtures declare siblings (jumpoint_and_jump_group also creates
				// .example_gp with group_policy_memberships, refreshed through
				// ReadGPMemberships at plan time) and list datasources. A dirty plan
				// therefore does not prove the imported resource is at fault -- read
				// the plan output before concluding the import failed to round-trip.
				require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
					"plan after import was not clean. Most likely %s does not round-trip (%s), "+
						"but this assertion covers the whole fixture — check the plan output for drift "+
						"on sibling resources or datasources before blaming import.", tc.addr, tc.why)
			})
		})
	}
}
