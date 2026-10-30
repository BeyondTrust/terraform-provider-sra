package test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"terraform-provider-sra/api"
	"terraform-provider-sra/api/apimock"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EOrphanSweep finds, and in delete mode removes, objects that interrupted
// E2E runs left on the appliance. It does nothing unless E2E_SWEEP is set in the
// real process environment (never from .env). See CONTRIBUTING.md before using
// delete mode. It is never parallel.
func TestE2EOrphanSweep(t *testing.T) {
	if !sweepEnv.Set {
		t.Skip("set E2E_SWEEP=dry-run, delete or gp-snapshot to run the orphan sweep")
	}
	host := os.Getenv("BT_API_HOST")
	set, err := parseSweepSettings(sweepEnv, host, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// No test logger: request tracing would print every URL.
	id, secret := os.Getenv("BT_CLIENT_ID"), os.Getenv("BT_CLIENT_SECRET")
	c, err := api.NewClient(host, &id, &secret)
	if err != nil {
		t.Fatalf("could not create an API client: %v", err)
	}
	if err := runSweep(c, set, t.Logf); err != nil {
		t.Fatal(err)
	}
}

// --- markers ----------------------------------------------------------------

func TestMarkerRoundTrip(t *testing.T) {
	for _, at := range []time.Time{
		time.Unix(0, 0),
		time.Date(2026, 10, 30, 12, 34, 56, 0, time.UTC),
		time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		m := newMarker(at)
		require.Len(t, m, 15)

		kind, base, suffix, got := parseMarker(m)
		assert.Equal(t, markerTimestamped, kind)
		assert.Equal(t, m, base)
		assert.Empty(t, suffix)
		assert.True(t, at.Truncate(time.Minute).Equal(got), "%s round-trips to %s", at, got)

		// The vault/secret fixture passes "${var.random_bits}2" to a module.
		kind, base, suffix, got = parseMarker(m + "2")
		assert.Equal(t, markerTimestamped, kind)
		assert.Equal(t, m, base)
		assert.Equal(t, "2", suffix)
		assert.True(t, at.Truncate(time.Minute).Equal(got))
	}

	seen := map[string]bool{}
	now := time.Now()
	for i := 0; i < 1000; i++ {
		m := newMarker(now)
		assert.False(t, seen[m], "parallel tests starting together must not share a marker")
		seen[m] = true
	}
}

func TestMarkerClassify(t *testing.T) {
	for s, want := range map[string]markerKind{
		"":                  markerUnusable,
		stageSkipRandomBits: markerUnusable,
		"abc123":            markerLegacy,
		"abc1232":           markerLegacy,
		"abc122":            markerLegacy,
		"abc12":             markerUnusable,
		"ABC123":            markerUnusable,
		"abc1234":           markerUnusable,
		"abc12322":          markerUnusable,
		"e2e00abcd0xyz12a":  markerUnusable,
		"e2e00abcd0xyz12":   markerTimestamped,
		"e2e00abcd0xyz122":  markerTimestamped,
		"e2e00abcd0xyz1":    markerUnusable,
		"e2e-0abcd0xyz12":   markerUnusable,
	} {
		kind, _, _, _ := parseMarker(s)
		assert.Equal(t, want, kind, "%q", s)
	}
	_, base, suffix, _ := parseMarker("abc1232")
	assert.Equal(t, "abc123", base)
	assert.Equal(t, "2", suffix)
}

// --- matchers ---------------------------------------------------------------

func TestSweepMatchers(t *testing.T) {
	T := newMarker(time.Now())
	groups := map[int]string{20: T}

	for _, tc := range []struct {
		name  string
		typ   string
		row   sweepRow
		match bool
	}{
		{"account group", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Name " + T, "description": T}, true},
		{"account group GP", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Renamed Name " + T + " GP", "description": T}, true},
		{"toggle group", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Name " + T + " Toggle", "description": T}, true},
		{"legacy group", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Name abc123 Nothing", "description": "abc123"}, true},
		{"group with no marker", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Name", "description": ""}, false},
		{"group with a trailing word", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Name " + T + " Prod", "description": T}, false},
		{"group whose description is not the marker", "vault_account_group", sweepRow{"id": 7.0, "name": "This is a Name " + T, "description": "Payroll"}, false},
		{"the Default group", "vault_account_group", sweepRow{"id": 1.0, "name": "This is a Name " + T, "description": T}, false},

		{"policy", "vault_account_policy", sweepRow{"name": "fun_policy " + T, "code_name": T}, true},
		{"policy mixed", "vault_account_policy", sweepRow{"name": "fun_policy " + T + " Mixed", "code_name": T + "_mixed"}, true},
		{"policy with another code name", "vault_account_policy", sweepRow{"name": "fun_policy " + T, "code_name": "payroll"}, false},

		{"user/pass account", "vault_account", sweepRow{"name": "Group UP This is a Name " + T, "type": "username_password", "username": T}, true},
		{"ssh account any", "vault_account", sweepRow{"name": "Standalone Key Any This is a Name " + T, "type": "ssh", "username": T + "any"}, true},
		{"ssh account in the secret fixture", "vault_account", sweepRow{"name": "Group Key This is a Name2 " + T + "2", "type": "ssh", "username": T + "2"}, true},
		{"token account", "vault_account", sweepRow{"name": "Standalone TOKEN GP fun_account " + T, "type": "opaque_token"}, true},
		{"account with a real username", "vault_account", sweepRow{"name": "Group UP This is a Name " + T, "type": "username_password", "username": "admin"}, false},
		{"account with an unknown prefix", "vault_account", sweepRow{"name": "Prod This is a Name " + T, "type": "opaque_token"}, false},

		{"jump group", "jump_group", sweepRow{"name": "Test JG " + T, "code_name": "example_" + T}, true},
		{"jump group GP", "jump_group", sweepRow{"name": "Test JG GP " + T, "code_name": "gp_example_" + T}, true},
		{"production jump group", "jump_group", sweepRow{"name": "Production Jump Group", "code_name": "prod"}, false},
		{"jump group with mismatched code name", "jump_group", sweepRow{"name": "Test JG GP " + T, "code_name": "example_" + T}, false},
		{"the sentinel", "jump_group", sweepRow{"name": "Test JG " + T, "code_name": sentinelCodeName}, false},
		{"jumpoint", "jumpoint", sweepRow{"name": "Test JP " + T, "code_name": "example_" + T}, true},
		{"jump group name on a jumpoint", "jumpoint", sweepRow{"name": "Test JG " + T, "code_name": "example_" + T}, false},

		{"shell jump", "shell_jump", sweepRow{"name": "This is a Name", "tag": T, "jump_group_id": 20.0, "jump_group_type": "shared"}, true},
		{"secondary item, updated tag", "mysql_tunnel_jump", sweepRow{"name": "This is a Name-secondary", "tag": T + "-updated", "jump_group_id": 20.0, "jump_group_type": "shared"}, true},
		{"tag match in a non-fixture group", "shell_jump", sweepRow{"name": "This is a Name", "tag": T, "jump_group_id": 200.0, "jump_group_type": "shared"}, false},
		{"tag match in a personal group", "shell_jump", sweepRow{"name": "This is a Name", "tag": T, "jump_group_id": 20.0, "jump_group_type": "personal"}, false},
		{"item named like a real one", "shell_jump", sweepRow{"name": "db01", "tag": T, "jump_group_id": 20.0, "jump_group_type": "shared"}, false},
		{"oracle in the fixture group", "oracle_tunnel_jump", sweepRow{"name": "This is a Name", "tag": T, "asset_group_id": 20.0, "asset_group_type": "shared"}, true},
		{"oracle in a non-fixture asset group", "oracle_tunnel_jump", sweepRow{"name": "This is a Name", "tag": T, "asset_group_id": 200.0, "asset_group_type": "shared"}, false},
		{"installer", "jump_client_installer", sweepRow{"name": "This is a Name", "tag": T, "jump_group_id": 20.0, "jump_group_type": "shared"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := matchFixture(typeByName(tc.typ), tc.row, groups)
			assert.Equal(t, tc.match, ok)
		})
	}
}

// --- a mock appliance ---------------------------------------------------------

const testHost = "appliance.example"

var (
	sweepNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	oldT     = newMarker(sweepNow.Add(-48 * time.Hour))
)

func sweepClient(srv *apimock.Server) *api.APIClient {
	return &api.APIClient{RootURL: srv.URL, BaseURL: srv.URL + apimock.BasePath, HTTPClient: srv.Client()}
}

func settings(mode sweepMode) sweepSettings {
	return sweepSettings{mode: mode, host: testHost, minAge: defaultSweepMinAge, max: defaultSweepMax, now: sweepNow}
}

func runOn(t *testing.T, srv *apimock.Server, set sweepSettings) (string, error) {
	t.Helper()
	var mu sync.Mutex
	var out strings.Builder
	err := runSweep(sweepClient(srv), set, func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(&out, format+"\n", args...)
	})
	return out.String(), err
}

// accountMatch is how vault/account filters: personal accounts only with
// include_personal=true, then field equality.
func accountMatch(row apimock.Row, filter url.Values) bool {
	if p, _ := row["personal"].(bool); p && filter.Get("include_personal") != "true" {
		return false
	}
	for k := range filter {
		if v, ok := row[k]; ok && fmt.Sprint(v) != filter.Get(k) {
			return false
		}
	}
	return true
}

func group(id int, T string) apimock.Row {
	return apimock.Row{"id": id, "name": "This is a Name " + T, "description": T}
}

func account(id int, T string, groupID int) apimock.Row {
	return apimock.Row{"id": id, "name": "Group UP This is a Name " + T, "type": "username_password", "username": T, "account_group_id": groupID}
}

func jumpGroup(id int, T string) apimock.Row {
	return apimock.Row{"id": id, "name": "Test JG " + T, "code_name": "example_" + T, "comments": ""}
}

func jumpoint(id int, T string) apimock.Row {
	return apimock.Row{"id": id, "name": "Test JP " + T, "code_name": "example_" + T}
}

func shellJump(id int, T string, groupID, jumpointID int) apimock.Row {
	return apimock.Row{"id": id, "name": "This is a Name", "tag": T, "hostname": "fun.jump.host",
		"jump_group_id": groupID, "jump_group_type": "shared", "jumpoint_id": jumpointID}
}

func sentinel() apimock.Row {
	return apimock.Row{"id": 500, "name": "Sweep allowed", "code_name": sentinelCodeName, "comments": "https://" + testHost + "/"}
}

// fixtureWorld is one run's leftovers beside some real configuration:
// account group 7 holding account 8, policy 9, jump group 20, jumpoint 21, and
// shell jump 30 in both. The real side is account group 100 holding account
// 101, jump group 200 and shell jump 300.
func fixtureWorld(srv *apimock.Server, T string) {
	srv.Rows("vault/account-group", group(7, T), apimock.Row{"id": 100, "name": "Payroll", "description": "real"})
	srv.Set("vault/account", &apimock.Collection{Match: accountMatch, Rows: []apimock.Row{
		account(8, T, 7),
		{"id": 101, "name": "Payroll admin", "type": "username_password", "username": "payroll", "account_group_id": 100},
	}})
	srv.Rows("vault/account-policy", apimock.Row{"id": 9, "name": "fun_policy " + T, "code_name": T})
	srv.Rows("jump-group", jumpGroup(20, T), apimock.Row{"id": 200, "name": "Production Jump Group", "code_name": "prod"}, sentinel())
	srv.Rows("jumpoint", jumpoint(21, T), apimock.Row{"id": 201, "name": "Prod JP", "code_name": "prod"})
	srv.Rows("jump-item/shell-jump", shellJump(30, T, 20, 21), shellJump(300, "abcdef", 200, 201))
	srv.Rows("jump-client")
}

func deletes(srv *apimock.Server) []string {
	var out []string
	for _, r := range srv.Requests() {
		if strings.HasPrefix(r, http.MethodDelete+" ") {
			out = append(out, strings.TrimPrefix(r, http.MethodDelete+" "))
		}
	}
	return out
}

// writePage answers a collection GET with one complete page.
func writePage(w http.ResponseWriter, rows []apimock.Row) {
	w.Header().Set("X-BT-Pagination-Total", strconv.Itoa(len(rows)))
	w.Header().Set("X-BT-Pagination-Last-Page", "1")
	_ = json.NewEncoder(w).Encode(rows)
}

// afterInventory serves first to the inventory list (the first GET of a path)
// and hands every later GET, the reference re-checks, to later.
func afterInventory(first []apimock.Row, later http.HandlerFunc) http.HandlerFunc {
	var mu sync.Mutex
	calls := 0
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			writePage(w, first)
			return
		}
		later(w, r)
	}
}

// --- the sweep --------------------------------------------------------------

func TestSweepOrder(t *testing.T) {
	last := -1
	for _, st := range sweepTypes {
		p := sweepPhase(st.kind)
		if p < 0 {
			continue
		}
		assert.GreaterOrEqual(t, p, last, "%s is out of order", st.name)
		last = p
	}
	assert.Less(t, sweepPhase(kindAccount), sweepPhase(kindAccountGroup), "accounts before the groups that hold them")
	assert.Less(t, sweepPhase(kindAccountGroup), sweepPhase(kindPolicy), "groups before the policies they use")
	assert.Less(t, sweepPhase(kindJumpItem), sweepPhase(kindJumpGroup), "items before their jump groups")
	assert.Less(t, sweepPhase(kindInstaller), sweepPhase(kindJumpGroup), "installers before their jump groups")
	assert.Less(t, sweepPhase(kindJumpItem), sweepPhase(kindJumpoint), "items before their jumpoints")
	assert.Negative(t, sweepPhase(kindJumpClient), "deployed jump clients are never deleted")

	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)

	out, err := runOn(t, srv, settings(sweepDelete))
	require.NoError(t, err, out)
	assert.Equal(t, []string{
		"vault/account/8",
		"vault/account-group/7",
		"vault/account-policy/9",
		"jump-item/shell-jump/30",
		"jump-group/20",
		"jumpoint/21",
	}, deletes(srv), "fixture objects only, contents before containers")
}

func TestSweepDryRunSendsNoDelete(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)

	out, err := runOn(t, srv, settings(sweepDryRun))
	require.NoError(t, err)
	assert.Zero(t, srv.CallsWithMethod(http.MethodDelete))
	for _, line := range []string{
		"sweep vault_account 8 ", "sweep vault_account_group 7 ", "sweep vault_account_policy 9 ",
		"sweep shell_jump 30 ", "sweep jump_group 20 ", "sweep jumpoint 21 ",
	} {
		assert.Contains(t, out, line)
	}
	assert.Equal(t, 6, strings.Count(out, " eligible \n"), out)
	assert.Contains(t, out, "sweep-total remote_rdp absent")
}

func TestSweepGPSnapshotSendsNoDelete(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)
	srv.Rows("group-policy", apimock.Row{"id": 3, "name": "first"}, apimock.Row{"id": 4, "name": "second"})
	srv.Rows("group-policy/3/jump-group", apimock.Row{"jump_group_id": 20}, apimock.Row{"jump_group_id": 200})
	srv.Rows("group-policy/3/jumpoint", apimock.Row{"jumpoint_id": 21})
	srv.Rows("group-policy/3/vault-account-group", apimock.Row{"account_group_id": 100, "role": "inject"})
	srv.Rows("group-policy/3/vault-account")

	out, err := runOn(t, srv, settings(sweepGPSnapshot))
	require.NoError(t, err)
	assert.Zero(t, srv.CallsWithMethod(http.MethodDelete))
	assert.Contains(t, out, "gp-snapshot jump-group 20 fixture\n")
	assert.Contains(t, out, "gp-snapshot jump-group 200 non-fixture\n")
	assert.Contains(t, out, "gp-snapshot jumpoint 21 fixture\n")
	assert.Contains(t, out, "gp-snapshot vault-account-group 100 non-fixture\n")
	assert.Contains(t, out, "gp-snapshot-total vault-account members=0 fixture=0 non-fixture=0")
	assert.Zero(t, srv.Calls(http.MethodGet, "group-policy/4/jump-group"), "only items[0]")
	assert.NotContains(t, out, "Production", "classification only, never names")
}

// Collection is not the last word: each delete re-reads the object and re-runs
// the container's reference queries.
func TestSweepRevalidatesBeforeDelete(t *testing.T) {
	t.Run("renamed after collection", func(t *testing.T) {
		srv := apimock.New()
		defer srv.Close()
		fixtureWorld(srv, oldT)
		srv.Handle(http.MethodGet, "vault/account-group/7", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(apimock.Row{"id": 7, "name": "Payroll 2", "description": oldT})
		})

		out, err := runOn(t, srv, settings(sweepDelete))
		require.NoError(t, err)
		assert.NotContains(t, deletes(srv), "vault/account-group/7")
		assert.Contains(t, out, "sweep vault_account_group 7 kept-at-delete no-longer-matches")
	})

	// Oracle names its jump group asset_group_id; the re-check must use that.
	t.Run("an oracle item appears in the jump group", func(t *testing.T) {
		srv := apimock.New()
		defer srv.Close()
		fixtureWorld(srv, oldT)
		var gotFilter url.Values
		srv.Handle(http.MethodGet, "jump-item/oracle-tunnel-jump", afterInventory(nil, func(w http.ResponseWriter, r *http.Request) {
			gotFilter = r.URL.Query()
			if r.URL.Query().Get("asset_group_id") == "20" {
				writePage(w, []apimock.Row{{"id": 99, "name": "Real DB", "asset_group_id": 20, "asset_group_type": "shared", "gateway_id": 201}})
				return
			}
			writePage(w, nil)
		}))

		out, err := runOn(t, srv, settings(sweepDelete))
		require.NoError(t, err)
		assert.NotContains(t, deletes(srv), "jump-group/20")
		assert.Contains(t, out, "sweep jump_group 20 kept-at-delete used by oracle_tunnel_jump 99")
		assert.Contains(t, deletes(srv), "jumpoint/21", "the jumpoint check uses gateway_id, which this item does not name")
		assert.NotEmpty(t, gotFilter)
	})
}

// Any failure in a reference check keeps the container.
func TestSweepReferenceCheckFailsClosed(t *testing.T) {
	for name, later := range map[string]http.HandlerFunc{
		"400": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) },
		// More pages than the listing will read: an incomplete list.
		"incomplete list": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-BT-Pagination-Total", "100001")
			w.Header().Set("X-BT-Pagination-Last-Page", "1001")
			_ = json.NewEncoder(w).Encode([]apimock.Row{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv := apimock.New()
			defer srv.Close()
			fixtureWorld(srv, oldT)
			srv.Handle(http.MethodGet, "jump-item/shell-jump", afterInventory(
				[]apimock.Row{shellJump(30, oldT, 20, 21)}, later))

			out, err := runOn(t, srv, settings(sweepDelete))
			require.NoError(t, err)
			assert.NotContains(t, deletes(srv), "jump-group/20")
			assert.NotContains(t, deletes(srv), "jumpoint/21")
			assert.Contains(t, out, "sweep jump_group 20 kept-at-delete reference-check-failed shell_jump")
		})
	}

	t.Run("a dependency list failed", func(t *testing.T) {
		srv := apimock.New()
		defer srv.Close()
		fixtureWorld(srv, oldT)
		srv.Set("jump-client", &apimock.Collection{Status: http.StatusInternalServerError})

		out, err := runOn(t, srv, settings(sweepDryRun))
		require.NoError(t, err)
		assert.Contains(t, out, "sweep jump_group 20 \"Test JG "+oldT+"\" skip list-failed:jump_client")
		assert.Contains(t, out, "sweep jumpoint 21 \"Test JP "+oldT+"\" eligible", "jumpoints do not depend on jump clients")
	})
}

// Deleting a policy re-points everything that uses it, personal accounts
// included, so any user outside this run's objects keeps it.
func TestSweepPolicyReferencedIsSkipped(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)
	srv.Set("vault/account", &apimock.Collection{Match: accountMatch, Rows: []apimock.Row{
		account(8, oldT, 7),
		{"id": 50, "name": "Mine", "type": "username_password", "username": "me", "personal": true, "account_group_id": 1, "account_policy": oldT},
	}})

	out, err := runOn(t, srv, settings(sweepDelete))
	require.NoError(t, err)
	assert.Contains(t, out, "sweep vault_account_policy 9 \"fun_policy "+oldT+"\" skip used by vault_account 50")
	assert.NotContains(t, deletes(srv), "vault/account-policy/9")
}

// A reference that appears after collection still keeps the container.
func TestSweepReferenceRecheckedAtDelete(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)
	srv.Handle(http.MethodGet, "vault/account", afterInventory([]apimock.Row{account(8, oldT, 7)},
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("account_group_id") == "7" {
				writePage(w, []apimock.Row{{"id": 61, "name": "Added since", "type": "username_password", "username": "x", "account_group_id": 7}})
				return
			}
			writePage(w, nil)
		}))
	srv.Handle(http.MethodGet, "vault/account/8", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(account(8, oldT, 7))
	})

	out, err := runOn(t, srv, settings(sweepDelete))
	require.NoError(t, err)
	assert.Contains(t, out, "sweep vault_account_group 7 kept-at-delete contains vault_account 61")
	assert.NotContains(t, deletes(srv), "vault/account-group/7")
}

// In a reference check a 404 keeps the container. Only an inventory 404 means
// "no such type here".
func TestSweepReference404Skips(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)
	srv.Handle(http.MethodGet, "jump-item/shell-jump", afterInventory([]apimock.Row{shellJump(30, oldT, 20, 21)},
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }))

	out, err := runOn(t, srv, settings(sweepDelete))
	require.NoError(t, err)
	assert.Contains(t, out, "sweep jump_group 20 kept-at-delete reference-check-failed shell_jump")
	assert.NotContains(t, deletes(srv), "jump-group/20")
}

// The sweeper's switches are read before setup.go's init() loads .env, so a
// .env file can't turn it on.
func TestSweepEnvNotFromDotenv(t *testing.T) {
	if sweepEnv.Set {
		t.Skip("E2E_SWEEP is set in this process's environment")
	}
	vars := map[string]string{
		"E2E_SWEEP":              "delete",
		"E2E_SWEEP_CONFIRM_HOST": testHost,
		"E2E_SWEEP_MIN_AGE":      "0s",
		"E2E_SWEEP_MAX":          "1000",
	}
	var lines []string
	for k, v := range vars {
		if _, set := os.LookupEnv(k); set {
			t.Skipf("%s is set in this process's environment", k)
		}
		lines = append(lines, k+"="+v)
		k := k
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}
	envFile := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(strings.Join(lines, "\n")+"\n"), 0o600))

	before := sweepEnv
	require.NoError(t, godotenv.Load(envFile))
	require.Equal(t, "delete", os.Getenv("E2E_SWEEP"), "fixture: the file did load")

	assert.Equal(t, before, sweepEnv)
	assert.False(t, sweepEnv.Set)
}

func TestSweepHostBinding(t *testing.T) {
	t.Run("confirmation", func(t *testing.T) {
		env := sweepEnvironment{Set: true, Mode: "delete", ConfirmHost: "other.example"}
		_, err := parseSweepSettings(env, "https://"+testHost, sweepNow)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), testHost)
		assert.NotContains(t, err.Error(), "other.example")

		env.ConfirmHost = ""
		_, err = parseSweepSettings(env, testHost, sweepNow)
		require.ErrorContains(t, err, "E2E_SWEEP_CONFIRM_HOST")

		env.ConfirmHost = "HTTPS://Appliance.Example/"
		set, err := parseSweepSettings(env, testHost, sweepNow)
		require.NoError(t, err, "scheme, case and trailing slash are normalised away")
		assert.Equal(t, sweepDelete, set.mode)
	})

	t.Run("delete mode has a 1h age floor", func(t *testing.T) {
		set, err := parseSweepSettings(sweepEnvironment{Set: true, Mode: "delete", ConfirmHost: testHost, MinAge: "1m"}, testHost, sweepNow)
		require.NoError(t, err)
		assert.Equal(t, time.Hour, set.minAge)
		set, err = parseSweepSettings(sweepEnvironment{Set: true, Mode: "dry-run", MinAge: "0s"}, testHost, sweepNow)
		require.NoError(t, err)
		assert.Zero(t, set.minAge)
	})

	t.Run("mode is an exact enum", func(t *testing.T) {
		for _, mode := range []string{"", "true", "Delete", "delete ", "dryrun"} {
			_, err := parseSweepSettings(sweepEnvironment{Set: true, Mode: mode}, testHost, sweepNow)
			assert.Error(t, err, "%q", mode)
		}
	})

	for name, tc := range map[string]struct {
		sentinels []apimock.Row
		wantErr   string
	}{
		"missing sentinel": {nil, "exactly one jump group"},
		"sentinel for a different host": {[]apimock.Row{{"id": 500, "name": "Sweep allowed", "code_name": sentinelCodeName, "comments": "restored-from.example"}},
			"do not name this host"},
		"two sentinels": {[]apimock.Row{sentinel(), {"id": 501, "name": "x", "code_name": sentinelCodeName, "comments": testHost}}, "found 2"},
	} {
		t.Run(name, func(t *testing.T) {
			srv := apimock.New()
			defer srv.Close()
			fixtureWorld(srv, oldT)
			srv.Rows("jump-group", append([]apimock.Row{jumpGroup(20, oldT)}, tc.sentinels...)...)

			_, err := runOn(t, srv, settings(sweepDelete))
			require.ErrorContains(t, err, tc.wantErr)
			assert.NotContains(t, err.Error(), testHost)
			assert.NotContains(t, err.Error(), "restored-from")
			assert.Zero(t, srv.CallsWithMethod(http.MethodDelete))
		})
	}
}

func TestSweepLegacyFileFormatAgeAndScope(t *testing.T) {
	listed := sweepNow.Add(-time.Hour).UTC().Format(time.RFC3339)
	for name, tc := range map[string]struct {
		file    string
		wantErr string
	}{
		"no header":        {"vault_account_group\t7\n", "first line"},
		"bad time":         {"# listed-at yesterday\n", "listed-at"},
		"too recent":       {"# listed-at " + sweepNow.Add(-10*time.Minute).Format(time.RFC3339) + "\n", "at least 35m"},
		"space for tab":    {"# listed-at " + listed + "\nvault_account_group 7\n", "line 2"},
		"unknown type":     {"# listed-at " + listed + "\ngroup_policy\t3\n", "line 2"},
		"never-swept type": {"# listed-at " + listed + "\njump_client\t3\n", "line 2"},
		"blank line":       {"# listed-at " + listed + "\n\n", "line 2"},
		"zero id":          {"# listed-at " + listed + "\nvault_account_group\t0\n", "line 2"},
		"valid":            {"# listed-at " + listed + "\nvault_account_group\t7\t# This is a Name abc123\njump_group\t20\n", ""},
	} {
		t.Run(name, func(t *testing.T) {
			ids, err := parseLegacyIDs(strings.NewReader(tc.file), sweepNow)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, map[sweepKey]bool{{"vault_account_group", 7}: true, {"jump_group", 20}: true}, ids)
		})
	}

	t.Run("dry-run lists legacy groups; delete removes only the reviewed ones", func(t *testing.T) {
		srv := apimock.New()
		defer srv.Close()
		srv.Rows("vault/account-group",
			apimock.Row{"id": 40, "name": "This is a Name abc123 Nothing", "description": "abc123"},
			apimock.Row{"id": 41, "name": "This is a Name zyx987 GP", "description": "zyx987"})
		srv.Set("vault/account", &apimock.Collection{Match: accountMatch})
		srv.Rows("jump-group", sentinel())

		candidates := filepath.Join(t.TempDir(), "candidates.txt")
		dry := settings(sweepDryRun)
		dry.candidatesOut = candidates
		out, err := runOn(t, srv, dry)
		require.NoError(t, err)
		assert.Contains(t, out, "sweep vault_account_group 40 \"This is a Name abc123 Nothing\" legacy-candidate no-creation-time")
		assert.Contains(t, out, "eligible=0 legacy-candidate=2")

		file, err := os.ReadFile(candidates)
		require.NoError(t, err)
		assert.Equal(t, "# listed-at "+sweepNow.Format(time.RFC3339)+"\n"+
			"vault_account_group\t40\t# This is a Name abc123 Nothing\n"+
			"vault_account_group\t41\t# This is a Name zyx987 GP\n", string(file))

		// A human keeps only 41; the delete runs 40 minutes later.
		reviewed := strings.Replace(string(file), "vault_account_group\t40\t# This is a Name abc123 Nothing\n", "", 1)
		ids, err := parseLegacyIDs(strings.NewReader(reviewed), sweepNow.Add(40*time.Minute))
		require.NoError(t, err)
		del := settings(sweepDelete)
		del.now = sweepNow.Add(40 * time.Minute)
		del.legacy = ids
		_, err = runOn(t, srv, del)
		require.NoError(t, err)
		assert.Equal(t, []string{"vault/account-group/41"}, deletes(srv))
	})

	t.Run("only delete mode reads an ids file", func(t *testing.T) {
		_, err := parseSweepSettings(sweepEnvironment{Set: true, Mode: "dry-run", LegacyIDs: "x"}, testHost, sweepNow)
		require.ErrorContains(t, err, "only to E2E_SWEEP=delete")
	})
}

func TestSweepCapCountsOnlyDeletable(t *testing.T) {
	young := newMarker(sweepNow.Add(-time.Hour))
	setup := func() *apimock.Server {
		srv := apimock.New()
		srv.Rows("vault/account-group",
			group(70, oldT), group(71, oldT), group(72, oldT),
			group(73, young), group(74, young),
			apimock.Row{"id": 75, "name": "This is a Name abc123", "description": "abc123"},
			apimock.Row{"id": 76, "name": "This is a Name zyx987", "description": "zyx987"})
		srv.Set("vault/account", &apimock.Collection{Match: accountMatch})
		srv.Rows("jump-group", sentinel())
		return srv
	}

	srv := setup()
	defer srv.Close()
	dry := settings(sweepDryRun)
	dry.max = 2
	out, err := runOn(t, srv, dry)
	require.NoError(t, err, "dry-run only warns")
	assert.Contains(t, out, "WARNING: more than E2E_SWEEP_MAX=2 would be deleted for vault_account_group")

	del := settings(sweepDelete)
	del.max = 2
	_, err = runOn(t, srv, del)
	require.ErrorContains(t, err, "E2E_SWEEP_MAX=2")
	assert.Zero(t, srv.CallsWithMethod(http.MethodDelete), "aborts before any deletion")

	srv2 := setup()
	defer srv2.Close()
	del.max = 3
	_, err = runOn(t, srv2, del)
	require.NoError(t, err, "3 eligible; the young and legacy groups don't count")
	got := deletes(srv2)
	sort.Strings(got)
	assert.Equal(t, []string{"vault/account-group/70", "vault/account-group/71", "vault/account-group/72"}, got)
}

// CI logs are public: nothing that isn't a fixture object is ever named.
func TestSweepOutputNeverNamesNonFixture(t *testing.T) {
	secrets := []string{"Secret Payroll Group", "secret-db.corp.example", "svc_secret_user", "Sweep allowed secretly", testHost}
	for _, mode := range []sweepMode{sweepDryRun, sweepDelete, sweepGPSnapshot} {
		t.Run(string(mode), func(t *testing.T) {
			srv := apimock.New()
			defer srv.Close()
			fixtureWorld(srv, oldT)
			srv.Rows("vault/account-group", group(7, oldT), apimock.Row{"id": 100, "name": secrets[0], "description": secrets[0]})
			srv.Set("vault/account", &apimock.Collection{Match: accountMatch, Rows: []apimock.Row{
				account(8, oldT, 7),
				{"id": 101, "name": "Group UP This is a Name " + oldT, "type": "username_password", "username": secrets[2], "account_group_id": 100},
			}})
			srv.Rows("jump-item/shell-jump", shellJump(30, oldT, 20, 21),
				apimock.Row{"id": 300, "name": "This is a Name", "tag": oldT, "hostname": secrets[1], "jump_group_id": 200, "jump_group_type": "shared", "jumpoint_id": 201})
			s := sentinel()
			s["name"] = secrets[3]
			srv.Rows("jump-group", jumpGroup(20, oldT), apimock.Row{"id": 200, "name": "Production Jump Group", "code_name": "prod"}, s)
			srv.Rows("group-policy", apimock.Row{"id": 3})
			srv.Rows("group-policy/3/vault-account-group", apimock.Row{"account_group_id": 100})
			srv.Rows("group-policy/3/jump-group")
			srv.Rows("group-policy/3/jumpoint")
			srv.Rows("group-policy/3/vault-account")

			out, err := runOn(t, srv, settings(mode))
			require.NoError(t, err)
			for _, secret := range secrets {
				assert.NotContains(t, out, secret)
			}
			assert.NotContains(t, out, "Production")
		})
	}
}

// Deleting an account group moves its accounts into Default, so a group holding
// anything this run won't delete is kept.
func TestSweepAccountGroupHoldingOtherAccountsIsSkipped(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)
	srv.Set("vault/account", &apimock.Collection{Match: accountMatch, Rows: []apimock.Row{
		account(8, oldT, 7),
		{"id": 102, "name": "Moved in by hand", "type": "username_password", "username": "ops", "account_group_id": 7},
	}})

	out, err := runOn(t, srv, settings(sweepDryRun))
	require.NoError(t, err)
	assert.Contains(t, out, "sweep vault_account_group 7 \"This is a Name "+oldT+"\" skip contains vault_account 102")
	assert.Contains(t, out, "sweep vault_account 8 \"Group UP This is a Name "+oldT+"\" eligible")
}

// A jump group or jumpoint something else still uses is kept, whatever the
// type of the user: here a real item placed in the fixture jump group and on
// the fixture jumpoint.
func TestSweepJumpGroupInUseIsSkipped(t *testing.T) {
	srv := apimock.New()
	defer srv.Close()
	fixtureWorld(srv, oldT)
	srv.Rows("jump-item/shell-jump", shellJump(30, oldT, 20, 21),
		apimock.Row{"id": 301, "name": "db01", "tag": "", "jump_group_id": 20, "jump_group_type": "shared", "jumpoint_id": 21})

	out, err := runOn(t, srv, settings(sweepDryRun))
	require.NoError(t, err)
	assert.Contains(t, out, "sweep jump_group 20 \"Test JG "+oldT+"\" skip used by shell_jump 301")
	assert.Contains(t, out, "sweep jumpoint 21 \"Test JP "+oldT+"\" skip used by shell_jump 301")
	assert.NotContains(t, out, "db01")
}

// An account group or standalone account whose jump item association names the
// fixture jump group in criteria.shared_jump_groups keeps it. An error reading
// an association keeps every jump group.
func TestSweepJumpGroupInAssociationCriteriaIsSkipped(t *testing.T) {
	for name, tc := range map[string]struct {
		path    string
		handler http.HandlerFunc
		want    string
	}{
		"account group": {"vault/account-group/100/jump-item-association", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"filter_type":"criteria","criteria":{"shared_jump_groups":[20],"host":[],"name":[],"tag":[],"comment":[]},"jump_items":[]}`))
		}, "skip criteria of vault_account_group 100"},
		"standalone account": {"vault/account/101/jump-item-association", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"filter_type":"criteria","criteria":{"shared_jump_groups":[20]},"jump_items":[]}`))
		}, "skip criteria of vault_account 101"},
		"read error": {"vault/account-group/100/jump-item-association", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}, "skip jump-item-association-check-failed"},
	} {
		t.Run(name, func(t *testing.T) {
			srv := apimock.New()
			defer srv.Close()
			fixtureWorld(srv, oldT)
			// Account 101 is standalone (Default group) for this test.
			srv.Set("vault/account", &apimock.Collection{Match: accountMatch, Rows: []apimock.Row{
				account(8, oldT, 7),
				{"id": 101, "name": "Payroll admin", "type": "username_password", "username": "payroll", "account_group_id": 1},
			}})
			srv.Handle(http.MethodGet, tc.path, tc.handler)

			out, err := runOn(t, srv, settings(sweepDryRun))
			require.NoError(t, err)
			assert.Contains(t, out, "sweep jump_group 20 \"Test JG "+oldT+"\" "+tc.want)
		})
	}
}

// --- fixture coverage -------------------------------------------------------

// TestSweepCoversFixtures renders every fixture object's name, tag, code_name,
// username and description from the bindings the tests actually pass (read
// from the test sources) and the variables.tf defaults they leave in place, and
// requires the sweeper to recognise each one. A new fixture, name or naming
// pattern fails here instead of leaking silently.
//
// Values it can't render are skipped: function calls such as concat(...), and
// anything nested in a block (jump_item_association criteria among them).
func TestSweepCoversFixtures(t *testing.T) {
	bindings := testFixtureBindings(t)
	require.NotEmpty(t, bindings)

	marker := newMarker(time.Now())
	rendered := map[string]int{}
	for _, product := range []string{"pra", "rs"} {
		for _, b := range bindings {
			dir := filepath.Join("..", "test-tf-files", product, b.fixture)
			for _, vars := range b.varSets(marker) {
				for _, obj := range renderFixture(t, dir, vars) {
					rendered[obj.resourceType]++
					st := typeByName(obj.sweepType)
					require.NotNil(t, st, obj.resourceType)
					got, ok := matchFixture(st, obj.row, map[int]string{20: obj.groupMarker})
					if assert.True(t, ok, "%s: sweeper does not recognise %s.%s rendered as %v",
						b.test, obj.resourceType, obj.name, obj.row) {
						assert.Equal(t, obj.wantMarker, got, "%s: %s.%s", b.test, obj.resourceType, obj.name)
					}
				}
			}
		}
	}

	t.Logf("rendered: %v", rendered)
	// Guards against a parser that silently renders nothing.
	for _, rt := range []string{
		"sra_vault_account_group", "sra_vault_account_policy", "sra_vault_ssh_account",
		"sra_vault_token_account", "sra_vault_username_password_account",
		"sra_jump_group", "sra_jumpoint", "sra_shell_jump", "sra_oracle_tunnel_jump",
		"sra_jump_client_installer",
	} {
		assert.Positive(t, rendered[rt], "no %s was rendered", rt)
	}
}

type fixtureBinding struct {
	test      string
	fixture   string
	vars      map[string]any // a string, or markerVar for randomBits
	overrides map[string]string
}

type markerVar struct{}

// varSets is the bindings as first applied, and with the later stage's
// vars["x"] = "..." overrides.
func (b fixtureBinding) varSets(marker string) []map[string]string {
	base := map[string]string{}
	for k, v := range b.vars {
		switch v := v.(type) {
		case string:
			base[k] = v
		case markerVar:
			base[k] = marker
		}
	}
	sets := []map[string]string{base}
	if len(b.overrides) > 0 {
		o := map[string]string{}
		for k, v := range base {
			o[k] = v
		}
		for k, v := range b.overrides {
			o[k] = v
		}
		sets = append(sets, o)
	}
	return sets
}

// testFixtureBindings reads every Test function and records which fixtures it
// copies and the Vars it passes.
func testFixtureBindings(t *testing.T) []fixtureBinding {
	t.Helper()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*_test.go")
	require.NoError(t, err)

	var out []fixtureBinding
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Body == nil {
				continue
			}
			var fixtures []string
			vars := map[string]any{}
			overrides := map[string]string{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "CopyTerraformFolderToTemp" && len(n.Args) == 3 {
						if sp, ok := n.Args[2].(*ast.CallExpr); ok && len(sp.Args) > 0 {
							if lit, ok := sp.Args[0].(*ast.BasicLit); ok {
								format, _ := strconv.Unquote(lit.Value)
								if p := strings.TrimPrefix(format, "test-tf-files/%s/"); p != format && p != "%s" {
									fixtures = append(fixtures, p)
								}
							}
						}
					}
				case *ast.KeyValueExpr:
					key, ok := n.Key.(*ast.BasicLit)
					if ident, isIdent := n.Key.(*ast.Ident); isIdent && ident.Name == "fixture" {
						if lit, ok := n.Value.(*ast.BasicLit); ok {
							p, _ := strconv.Unquote(lit.Value)
							fixtures = append(fixtures, p)
						}
					}
					if !ok || key.Kind != token.STRING {
						break
					}
					name, _ := strconv.Unquote(key.Value)
					switch v := n.Value.(type) {
					case *ast.BasicLit:
						if v.Kind == token.STRING {
							vars[name], _ = strconv.Unquote(v.Value)
						}
					case *ast.Ident:
						if v.Name == "randomBits" {
							vars[name] = markerVar{}
						}
					}
				case *ast.AssignStmt:
					if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
						break
					}
					idx, ok := n.Lhs[0].(*ast.IndexExpr)
					if !ok {
						break
					}
					key, ok1 := idx.Index.(*ast.BasicLit)
					val, ok2 := n.Rhs[0].(*ast.BasicLit)
					if ok1 && ok2 && key.Kind == token.STRING && val.Kind == token.STRING {
						k, _ := strconv.Unquote(key.Value)
						overrides[k], _ = strconv.Unquote(val.Value)
					}
				}
				return true
			})
			for _, fx := range fixtures {
				out = append(out, fixtureBinding{test: fn.Name.Name, fixture: fx, vars: vars, overrides: overrides})
			}
		}
	}
	return out
}

type renderedObject struct {
	resourceType, name string
	sweepType          string
	row                sweepRow
	wantMarker         string
	groupMarker        string
}

// tfBlock is a top-level block: its labels and its scalar, top-level attributes
// as raw expressions.
type tfBlock struct {
	kind   string
	labels []string
	attrs  map[string]string
}

var (
	tfStringRE = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	tfOpenRE   = regexp.MustCompile(`^(resource|data|module|variable|locals|output|terraform)((?:\s+"[^"]*")*)\s*\{$`)
	tfLabelRE  = regexp.MustCompile(`"([^"]*)"`)
	tfAttrRE   = regexp.MustCompile(`^([a-z_]+)\s*=\s*(.+)$`)
	tfInterpRE = regexp.MustCompile(`\$\{(var|local)\.([a-z_]+)\}`)
	tfRefRE    = regexp.MustCompile(`^(var|local)\.([a-z_]+)$`)
)

func parseTFDir(t *testing.T, dir string) []tfBlock {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no .tf files in %s", dir)

	var blocks []tfBlock
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		depth := 0
		var cur *tfBlock
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			bare := tfStringRE.ReplaceAllString(line, `""`)
			if depth == 0 {
				if m := tfOpenRE.FindStringSubmatch(line); m != nil {
					b := tfBlock{kind: m[1], attrs: map[string]string{}}
					for _, l := range tfLabelRE.FindAllStringSubmatch(m[2], -1) {
						b.labels = append(b.labels, l[1])
					}
					blocks = append(blocks, b)
					cur = &blocks[len(blocks)-1]
				}
			} else if depth == 1 && cur != nil {
				if m := tfAttrRE.FindStringSubmatch(line); m != nil {
					cur.attrs[m[1]] = m[2]
				}
			}
			depth += strings.Count(bare, "{") + strings.Count(bare, "[") - strings.Count(bare, "}") - strings.Count(bare, "]")
		}
	}
	return blocks
}

// renderTF renders a string template, var.x or local.x, or reports false.
func renderTF(expr string, vars, locals map[string]string) (string, bool) {
	lookup := func(kind, name string) (string, bool) {
		if kind == "var" {
			v, ok := vars[name]
			return v, ok
		}
		v, ok := locals[name]
		return v, ok
	}
	if m := tfRefRE.FindStringSubmatch(expr); m != nil {
		return lookup(m[1], m[2])
	}
	if !strings.HasPrefix(expr, `"`) || !strings.HasSuffix(expr, `"`) || len(expr) < 2 {
		return "", false
	}
	ok := true
	out := tfInterpRE.ReplaceAllStringFunc(expr[1:len(expr)-1], func(s string) string {
		m := tfInterpRE.FindStringSubmatch(s)
		v, found := lookup(m[1], m[2])
		ok = ok && found
		return v
	})
	return out, ok && !strings.Contains(out, "${")
}

var fixtureResourceTypes = map[string]string{
	"sra_vault_account_group":             "vault_account_group",
	"sra_vault_account_policy":            "vault_account_policy",
	"sra_vault_username_password_account": "vault_account",
	"sra_vault_ssh_account":               "vault_account",
	"sra_vault_token_account":             "vault_account",
	"sra_jump_group":                      "jump_group",
	"sra_jumpoint":                        "jumpoint",
	"sra_shell_jump":                      "shell_jump",
	"sra_remote_rdp":                      "remote_rdp",
	"sra_remote_vnc":                      "remote_vnc",
	"sra_protocol_tunnel_jump":            "protocol_tunnel_jump",
	"sra_postgresql_tunnel_jump":          "postgresql_tunnel_jump",
	"sra_my_sql_tunnel_jump":              "mysql_tunnel_jump",
	"sra_mongodb_tunnel_jump":             "mongodb_tunnel_jump",
	"sra_oracle_tunnel_jump":              "oracle_tunnel_jump",
	"sra_network_tunnel_jump":             "network_tunnel_jump",
	"sra_web_jump":                        "web_jump",
	"sra_jump_client_installer":           "jump_client_installer",
}

// renderFixture renders the sweepable resources of the module in dir, and of
// every module it calls, under the given variable bindings.
func renderFixture(t *testing.T, dir string, bound map[string]string) []renderedObject {
	t.Helper()
	blocks := parseTFDir(t, dir)

	vars := map[string]string{}
	for _, b := range blocks {
		if b.kind == "variable" && len(b.labels) == 1 {
			if v, ok := renderTF(b.attrs["default"], nil, nil); ok {
				vars[b.labels[0]] = v
			}
		}
	}
	for k, v := range bound {
		vars[k] = v
	}
	locals := map[string]string{}
	for pass := 0; pass < 2; pass++ {
		for _, b := range blocks {
			if b.kind == "locals" {
				for k, expr := range b.attrs {
					if v, ok := renderTF(expr, vars, locals); ok {
						locals[k] = v
					}
				}
			}
		}
	}

	var out []renderedObject
	for _, b := range blocks {
		switch {
		case b.kind == "module":
			source, ok := renderTF(b.attrs["source"], nil, nil)
			require.True(t, ok, "module %v in %s", b.labels, dir)
			args := map[string]string{}
			for k, expr := range b.attrs {
				if v, ok := renderTF(expr, vars, locals); ok && k != "source" {
					args[k] = v
				}
			}
			out = append(out, renderFixture(t, filepath.Join(dir, source), args)...)

		case b.kind == "resource" && len(b.labels) == 2:
			st, ok := fixtureResourceTypes[b.labels[0]]
			if !ok {
				continue
			}
			row := sweepRow{"id": 7.0}
			for _, attr := range []string{"name", "tag", "code_name", "username", "description", "type"} {
				if expr, present := b.attrs[attr]; present {
					v, ok := renderTF(expr, vars, locals)
					if !ok {
						t.Logf("%s: %s.%s.%s = %s is not rendered", dir, b.labels[0], b.labels[1], attr, expr)
						continue
					}
					row[attr] = v
				}
			}
			switch b.labels[0] {
			case "sra_vault_username_password_account":
				row["type"] = "username_password"
			case "sra_vault_token_account":
				row["type"] = "opaque_token"
			case "sra_vault_ssh_account":
				if _, set := row["type"]; !set {
					row["type"] = "ssh"
				}
			}
			if typ := typeByName(st); typ.groupField != "" {
				row[typ.groupField] = 20.0
				row[typ.groupTypeField] = "shared"
			}
			out = append(out, renderedObject{
				resourceType: b.labels[0], name: b.labels[1], sweepType: st, row: row,
				wantMarker: vars["random_bits"], groupMarker: vars["random_bits"],
			})
		}
	}
	return out
}
