package test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"terraform-provider-sra/api"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/require"
)

// importGuard tracks the one piece of state the orphan recovery below needs: the
// id of the object we are about to detach from Terraform state, and whether the
// re-import failed and left it detached.
type importGuard struct {
	id     string // captured BEFORE `terraform state rm`
	failed bool   // set ONLY by the import step's error branch
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
func guardImportOrphan[I api.APIResource](t *testing.T, g *importGuard, what string) {
	t.Cleanup(func() {
		if !g.failed {
			return // clean run: the address is in state and Destroy owns it
		}
		id, err := strconv.Atoi(g.id)
		if err != nil || id < 1 {
			t.Errorf("orphan recovery: %s has unusable id %q — LEAKED on the appliance", what, g.id)
			return
		}
		switch err := api.DeleteItem[I](freshClient(t), &id); {
		case err == nil:
			t.Logf("orphan recovery: reclaimed %s %d", what, id)
		case api.IsNotFound(err):
			// Already gone: destroy reclaimed it, or it was never created.
		default:
			t.Errorf("orphan recovery FAILED for %s %d — LEAKED on the appliance: %v", what, id, err)
		}
	})
}

// freshClient builds a new API client rather than reusing the package-level one.
//
// This is load-bearing, not defensive. api.NewClient uses oauth2
// clientcredentials, which caches its token and refreshes only on expiry -- but
// the appliance invalidates previously-issued tokens when a new one is issued,
// and every `terraform apply` spawns a provider subprocess that requests its own
// token with the same credentials. So the token held by the package-level client
// is silently invalidated by the applies these tests run, while oauth2 still
// believes it is valid and declines to refresh. Reusing it yields
// "status: 401 ... Access token is invalid" on the first call after an apply.
func freshClient(t *testing.T) *api.APIClient {
	t.Helper()

	id := os.Getenv("BT_CLIENT_ID")
	secret := os.Getenv("BT_CLIENT_SECRET")
	c, err := api.NewClient(os.Getenv("BT_API_HOST"), &id, &secret)
	require.NoError(t, err, "could not build a fresh API client")
	return c
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
func reimport(t *testing.T, opts *terraform.Options, g *importGuard, addr string) {
	// Keep these two adjacent, with no assertions between them -- every statement
	// here widens the orphan window.
	_, err := terraform.RunTerraformCommandE(t, opts, "state", "rm", addr)
	require.NoError(t, err, "state rm failed for %s", addr)

	args := append([]string{"import", "-input=false"}, terraform.FormatTerraformVarsAsArgs(opts.Vars)...)
	args = append(args, addr, g.id)
	if _, err := terraform.RunTerraformCommandE(t, opts, args...); err != nil {
		g.failed = true
		require.NoError(t, err, "import failed — %s (id %s) is now orphaned on the appliance", addr, g.id)
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
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/vault/account_group", productPath()))

	const addr = "sra_vault_account_group.new_account_group_jia"
	guard := &importGuard{}
	guardImportOrphan[api.VaultAccountGroup](t, guard, addr)

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
		guard.id = terraform.OutputMap(t, terraformOptions, "group_jia")["id"]
		require.NotEmpty(t, guard.id, "could not read the account group id from outputs")

		reimport(t, terraformOptions, guard, addr)

		// The regression assertion. Pre-fix this routed to CreateItem (POST)
		// against a GET/PATCH-only endpoint and failed; post-fix it PATCHes.
		//
		// Deliberately NOT asserting PlanExitCode == 0 here: a clean plan is
		// unreachable for this resource. readJIA gates its state write on the
		// pre-refresh value being non-null (bt/rs/vault_account_group.go:263) and
		// after import it is null, so jump_item_association stays null while the
		// schema default supplies a non-null value. That gap is pre-existing and
		// tracked in BUGS.md; Test 1b covers the clean round-trip on resources
		// that do not have it.
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Confirm the account group still exists", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		// Guards against a silently destructive apply: the step above would also
		// "succeed" if it had deleted and recreated the object.
		id, err := strconv.Atoi(guard.id)
		require.NoError(t, err)
		item, err := api.GetItem[api.VaultAccountGroup](freshClient(t), &id)
		require.NoError(t, err, "account group %d no longer exists after the post-import apply", id)
		require.NotNil(t, item)

		// And the id must be unchanged -- a destroy/recreate would renumber it.
		require.Equal(t, guard.id, terraform.OutputMap(t, terraformOptions, "group_jia")["id"],
			"the post-import apply replaced the account group instead of updating it")
	})
}

// TestImportRoundTripJumpGroup asserts that importing a flat resource yields a
// genuinely clean plan -- the property TestImportThenApplyAccountGroup cannot
// assert because of the null-gate documented there.
//
// sra_jump_group.example is used rather than .example_gp: .example declares only
// name and code_name, so state-null equals config-null throughout, while
// .example_gp declares group_policy_memberships and would diff.
func TestImportRoundTripJumpGroup(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/jump_items/jumpoint_and_jump_group", productPath()))

	const addr = "sra_jump_group.example"
	guard := &importGuard{}
	guardImportOrphan[api.JumpGroup](t, guard, addr)

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraform.Destroy(t, test_structure.LoadTerraformOptions(t, testFolder))
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

		// Apply twice before asserting anything about plan emptiness. The fixture
		// declares list datasources whose outputs are read at plan time, before
		// the resources exist; after one apply those outputs are still empty and
		// the next plan reports "Changes to Outputs", which -detailed-exitcode
		// reports as 2. Every existing test in this suite applies twice for the
		// same reason (see TestAccountGroup).
		terraform.Apply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "Round-trip the jump group through import", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		guard.id = terraform.OutputMap(t, terraformOptions, "jump_group")["id"]
		require.NotEmpty(t, guard.id, "could not read the jump group id from outputs")

		reimport(t, terraformOptions, guard, addr)

		// Assert == 0, never != 2: GetExitCodeForTerraformCommandContextE returns
		// 1 on a plan *error*, so != 2 would pass on a broken plan.
		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
			"plan after import was not clean — importing %s does not round-trip", addr)
	})
}

// TestImportRoundTripAccountPolicy is the second clean round-trip, covering the
// vault family that TestImportRoundTripJumpGroup does not.
//
// new_account_policy is used rather than its _false/_mixed siblings because it is
// the only one that sets maximum_password_age explicitly; that attribute is
// Optional+Computed with no default, so leaving it unset makes the post-import
// value depend on what the appliance returns rather than on the config.
func TestImportRoundTripAccountPolicy(t *testing.T) {
	randomBits := setEnvAndGetRandom(t)
	testFolder := test_structure.CopyTerraformFolderToTemp(t, "../", fmt.Sprintf("test-tf-files/%s/vault/account_policy", productPath()))

	const addr = "sra_vault_account_policy.new_account_policy"
	guard := &importGuard{}
	guardImportOrphan[api.VaultAccountPolicy](t, guard, addr)

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraform.Destroy(t, test_structure.LoadTerraformOptions(t, testFolder))
	})

	test_structure.RunTestStage(t, "setup", func() {
		terraformOptions := withBaseTFOptions(t, &terraform.Options{
			TerraformDir: testFolder,
			Vars: map[string]interface{}{
				"random_bits": randomBits,
				"name":        "fun_policy",
			},
		})
		test_structure.SaveTerraformOptions(t, testFolder, terraformOptions)
		terraform.InitAndApply(t, terraformOptions)
		terraform.Apply(t, terraformOptions) // settle the list datasource output
	})

	test_structure.RunTestStage(t, "Round-trip the account policy through import", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, testFolder)

		guard.id = terraform.OutputMap(t, terraformOptions, "policy")["id"]
		require.NotEmpty(t, guard.id, "could not read the account policy id from outputs")

		reimport(t, terraformOptions, guard, addr)

		require.Equal(t, 0, terraform.PlanExitCode(t, terraformOptions),
			"plan after import was not clean — importing %s does not round-trip", addr)
	})
}
