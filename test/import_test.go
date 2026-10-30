package test

// End-to-end coverage for the two lifecycle phases this suite never entered:
// `terraform import` (previously zero coverage) and, in
// account_group_update_test.go, a genuine config-change update.
//
// WHAT THIS DOES NOT CATCH -- stated so it is not overestimated later:
//
//   - A jump-item-association PATCH that returns 200 echoing the request but does
//     not persist is now caught, but only indirectly: Update writes the API
//     *response* into state, and it is the final plan's refresh (which reads the
//     association back) that would show the difference.
//   - A PATCH writing wrong field values IS caught, but incidentally: by Terraform
//     core's inconsistent-result check, because the fixture declares filter_type,
//     criteria.tag and jump_items explicitly, so those planned values are known.
//     That coverage disappears for any field the fixture stops declaring.
//   - An import populating a field incorrectly is caught for every attribute the
//     fixture declares (a wrong value is a diff), but NOT for unset
//     Optional+Computed attributes -- which is why the account-policy case pins
//     maximum_password_age explicitly.
//
// The original tests were verified against a PRA appliance; the import-discovery
// rebuild (memberships and the account group association read on import) has
// not yet been run live. The rs/ fixtures are mirrored from the pra/ ones and
// compile-checked but have not been executed locally; CI's terratest_rs job is
// the first thing that runs them.

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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
// markerUsable, the first half, lives in setup.go beside the marker itself.

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
		//    constant "not_so_random" for every test (stageSkipRandomBits). The
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

		// Confirm ownership BEFORE deleting. The type parameter I and the address
		// string are supplied independently by the caller, and the delete derives
		// its path solely from I.Endpoint() -- so a mis-paired case (say
		// sra_jump_group's address with api.VaultAccountGroup) would delete a
		// same-numbered object in a DIFFERENT id namespace, on a shared appliance,
		// and report success. The same applies if an outputKey were ever pointed at
		// a datasource rather than a managed resource. Every fixture names its
		// resources with the run's randomBits, so requiring that marker turns both
		// mistakes into a loud refusal instead of a silent deletion.
		owned := func(item *I) error {
			blob, err := json.Marshal(item)
			if err != nil || !safeToReclaim(blob, g.randomBits) {
				return errors.New("object does not carry this run's marker")
			}
			return nil
		}

		var outcome deleteOutcome
		err := errors.New("no attempt made")
		// Two attempts, each with a new client. The documented failure here is a
		// 401 from token eviction (see freshClient), for which a fresh mint IS the
		// remedy -- and without the retry a single unlucky read costs a real
		// object that the delete would have reclaimed. A retry re-reads and
		// re-checks ownership, so it cannot delete on stale information.
		for attempt := 0; attempt < 2 && err != nil; attempt++ {
			c, cerr := freshClientE(t)
			if cerr != nil {
				err = fmt.Errorf("could not build a client: %w", cerr)
				continue
			}
			var tmp I
			outcome, err = deleteIfStill(c, tmp.Endpoint(), id, owned)
		}

		switch {
		case err != nil:
			t.Errorf("orphan recovery FAILED for %s %d — LEAKED on the appliance: %v", g.addr, id, err)
		case outcome.refused != nil:
			t.Errorf("orphan recovery REFUSED for %s %d: %v (%q), so it is not ours to delete. "+
				"Check the case's API type matches its address. Nothing was deleted; if an object really did leak, remove it by hand.",
				g.addr, id, outcome.refused, g.randomBits)
		case outcome.result == "deleted":
			t.Logf("orphan recovery: reclaimed %s %d", g.addr, id)
		}
	})
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

// TestImportThenApplyAccountGroup imports an account group that declares both
// jump_item_association and group_policy_memberships, asserts the import plans
// clean, and then changes the association so the apply goes through Update.
//
// Two regressions are covered. Import used to leave both attributes null (Read
// refreshed only what was already in state), so the post-import apply PATCHed
// the association and POSTed a membership that already existed, which the
// appliance rejects. And Update once POSTed to the association endpoint, which
// exposes only GET and PATCH; changing extra_tags after the import is what
// routes the apply through Update now that the import itself is clean. The
// POST-vs-PATCH choice with a null association in state stays covered by
// TestVaultAccountGroupUpdate_JIAUsesPatchNeverPost.
//
// The assertion that matters is that the post-import APPLY succeeds. `terraform
// plan` never invokes the provider's Update -- Update is apply-only -- so a test
// ending at plan could not catch a regression there.
func TestImportThenApplyAccountGroup(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	var id string
	extraTag := randomBits + "-extra"
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/vault/account_group", productPath()))

	guard := &importGuard{addr: "sra_vault_account_group.new_account_group", randomBits: randomBits}
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

		// The second apply settles the list data source's output, which is read
		// at plan time before the groups exist (see TestImportRoundTrip).
		terraform.Apply(t, terraformOptions)
		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
			"fixture is not stable before import — this is pre-existing drift, not an import defect")
	})

	test_structure.RunTestStage(t, "Import the account group and apply over it", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Capture the id BEFORE detaching from state -- once `state rm` has run,
		// this is the only handle left on the object.
		id = terraform.OutputMap(t, terraformOptions, "group")["id"]
		require.NotEmpty(t, id, "could not read the account group id from outputs")

		reimport(t, terraformOptions, guard, id)

		// Assert == 0, never != 2: PlanExitCode returns 1 on a plan error.
		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
			"plan after import was not clean: import should read jump_item_association and "+
				"group_policy_memberships, so a configuration that matches the appliance plans no changes")

		// Now change the association, so the apply goes through Update.
		terraformOptions.Vars["extra_tags"] = []string{extraTag}
		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		require.Equal(t, 2, terraform.PlanExitCode(t, terraformOptions),
			"adding a criteria tag should plan an update")

		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Confirm the update", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// id crosses stages deliberately: this stage compares the pre-import id
		// against the post-apply one, so it cannot be re-derived here. Guard it
		// explicitly, because under SKIP_<stage> a value produced in a skipped
		// stage stays empty and the comparison below would otherwise fail with a
		// message implying the provider replaced the resource.
		require.NotEmpty(t, id, "the account group id was never captured — the import stage did not run "+
			"(SKIP_<stage> set?); this is a harness problem, not a provider one")

		group := extractJson(t, terraformOptions, "group")
		require.Equal(t, id, group.Path("id").Data(),
			"the post-import apply replaced the account group instead of updating it")

		tags, _ := group.Path("jump_item_association.criteria.tag").Data().([]any)
		require.ElementsMatch(t, []any{randomBits, extraTag}, tags, "the association update did not persist")

		memberships, _ := group.Path("group_policy_memberships").Data().([]any)
		require.Len(t, memberships, 1, "the membership was dropped or duplicated by the post-import apply")

		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions), "plan after the update was not clean")
	})
}

// TestImportRoundTrip asserts that importing a resource whose configuration
// matches the appliance yields a genuinely clean plan.
//
// The cases cover different families: sra_jump_group has no vault surface and
// sra_vault_account_policy does. The example_gp cases declare
// group_policy_memberships, which import discovers by listing every group
// policy's members (refresh reads back only the memberships already in state).
//
// No vault account can be in this list: its secret (private_key, password or
// token) is write-only and the API never returns it, so an import always plans
// an update for it. The ssh_account_gp case therefore asserts on the plan JSON
// instead: memberships unchanged, and nothing but the secret planned.
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
		// assertImported replaces the default whole-plan PlanExitCode == 0 check.
		assertImported func(t *testing.T, opts *terraform.Options, addr string)
	}{
		{
			name:      "jump_group",
			fixture:   "jump_items/jumpoint_and_jump_group",
			addr:      "sra_jump_group.example",
			outputKey: "jump_group",
			guard:     guardImportOrphan[api.JumpGroup],
			why:       "flat: only name and code_name declared",
		},
		{
			name:      "jump_group_gp",
			fixture:   "jump_items/jumpoint_and_jump_group",
			addr:      "sra_jump_group.example_gp",
			outputKey: "jump_group_gp",
			guard:     guardImportOrphan[api.JumpGroup],
			why:       "import must discover the jump-group membership of data.sra_group_policy_list.gp.items[0]",
		},
		{
			name:      "jumpoint_gp",
			fixture:   "jump_items/jumpoint_and_jump_group",
			addr:      "sra_jumpoint.example_gp",
			outputKey: "jumpoint_gp",
			guard:     guardImportOrphan[api.Jumpoint],
			why:       "import must discover the jumpoint membership of data.sra_group_policy_list.gp.items[0]",
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
		{
			name:           "ssh_account_gp",
			fixture:        "vault/ssh_account",
			addr:           "sra_vault_ssh_account.stand_alone_gp",
			outputKey:      "stand_alone_gp",
			guard:          guardImportOrphan[api.VaultSSHAccount],
			why:            "import must discover the vault-account membership",
			assertImported: assertOnlySecretPlanned,
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

				if tc.assertImported != nil {
					tc.assertImported(t, terraformOptions, tc.addr)
					return
				}

				// Assert == 0, never != 2: GetExitCodeForTerraformCommandContextE
				// returns 1 on a plan *error*, so != 2 would pass on a broken plan.
				//
				// Note this covers the WHOLE configuration, not just tc.addr. The
				// fixtures declare siblings and list datasources, so a dirty plan
				// does not prove the imported resource is at fault -- read the plan
				// output before concluding the import failed to round-trip.
				require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
					"plan after import was not clean. Most likely %s does not round-trip (%s), "+
						"but this assertion covers the whole fixture — check the plan output for drift "+
						"on sibling resources or datasources before blaming import.", tc.addr, tc.why)
			})
		})
	}
}

// assertOnlySecretPlanned is the round-trip check for an imported SSH account.
// Its private_key and private_key_passphrase are write-only, so the import
// always plans an update for them; anything else changing, memberships above
// all, is an import that did not round-trip.
//
// The plan JSON holds the fixture's private key in clear, so it is never logged.
func assertOnlySecretPlanned(t *testing.T, opts *terraform.Options, addr string) {
	// A copy: the helper sets PlanFilePath, which would add -out to every later
	// plan made with these options.
	planOpts := *opts
	plan := terraform.InitAndPlanAndShowWithStructNoLogTempPlanFile(t, &planOpts)

	change, ok := plan.ResourceChangesMap[addr]
	require.True(t, ok, "%s is missing from the plan", addr)
	before, _ := change.Change.Before.(map[string]interface{})
	after, _ := change.Change.After.(map[string]interface{})
	unknown, _ := change.Change.AfterUnknown.(map[string]interface{})
	require.NotNil(t, before, "%s has no prior state in the plan", addr)
	require.NotNil(t, after, "%s is planned for deletion", addr)

	require.Equal(t, before["group_policy_memberships"], after["group_policy_memberships"],
		"import did not discover %s's group policy memberships", addr)

	changed := []string{}
	for k := range before {
		if _, isUnknown := unknown[k]; isUnknown || !reflect.DeepEqual(before[k], after[k]) {
			changed = append(changed, k)
		}
	}
	require.ElementsMatch(t, []string{"private_key", "private_key_passphrase"}, changed,
		"after import %s should plan only its write-only secret", addr)
}
