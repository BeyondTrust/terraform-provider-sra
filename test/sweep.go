package test

// The E2E orphan sweeper: finds objects that interrupted E2E runs left on the
// appliance and, when told to, deletes them. TestE2EOrphanSweep (sweep_test.go)
// is the entry point; CONTRIBUTING.md documents the modes, variables and the
// review flow for legacy objects.
//
// Every rule here leans towards keeping an object. The appliance under test is
// shared, and a few of its deletes cascade into objects the sweeper did not
// choose: deleting an account group moves its accounts into the Default group,
// and deleting an account policy re-points every account and group using it to
// the global default, with nothing in the response to say so.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"terraform-provider-sra/api"

	"golang.org/x/oauth2"
)

// sweepEnvironment is every E2E_SWEEP* variable, read once.
//
// sweepEnv is a package-level initialiser, which Go runs before any init()
// function, and so before setup.go's init() loads the repository's .env. Only
// the real process environment can turn the sweeper on, never a .env file a
// developer keeps for the ordinary suite.
type sweepEnvironment struct {
	Set           bool
	Mode          string
	ConfirmHost   string
	MinAge        string
	LegacyIDs     string
	Max           string
	CandidatesOut string
}

var sweepEnv = readSweepEnv()

func readSweepEnv() sweepEnvironment {
	mode, set := os.LookupEnv("E2E_SWEEP")
	return sweepEnvironment{
		Set:           set,
		Mode:          mode,
		ConfirmHost:   os.Getenv("E2E_SWEEP_CONFIRM_HOST"),
		MinAge:        os.Getenv("E2E_SWEEP_MIN_AGE"),
		LegacyIDs:     os.Getenv("E2E_SWEEP_LEGACY_IDS"),
		Max:           os.Getenv("E2E_SWEEP_MAX"),
		CandidatesOut: os.Getenv("E2E_SWEEP_CANDIDATES_OUT"),
	}
}

type sweepMode string

const (
	sweepDryRun     sweepMode = "dry-run"
	sweepDelete     sweepMode = "delete"
	sweepGPSnapshot sweepMode = "gp-snapshot"
)

const (
	// sentinelCodeName is the jump group a human creates once on each test
	// appliance, with comments set to the normalised host, to allow deletion
	// there. Without it delete mode refuses to run.
	sentinelCodeName = "e2e_sweep_allowed"

	defaultSweepMinAge = 24 * time.Hour
	// deleteMinAgeFloor is well past CI's `go test -timeout 30m`, after which no
	// run that made the object can still be using it.
	deleteMinAgeFloor = time.Hour
	// futureSkew tolerates clocks a little ahead of this one.
	futureSkew = 5 * time.Minute
	// legacyListMinAge is how old a reviewed candidates file must be: longer
	// than any run that could have created an object it lists.
	legacyListMinAge = 35 * time.Minute
	defaultSweepMax  = 20

	listedAtPrefix = "# listed-at "
)

type sweepKey struct {
	typ string
	id  int
}

type sweepSettings struct {
	mode          sweepMode
	host          string // normalised BT_API_HOST
	minAge        time.Duration
	max           int
	legacy        map[sweepKey]bool
	candidatesOut string
	now           time.Time
}

// normaliseHost reduces a host to the form the sentinel's comments and the
// confirmation must match: no scheme, no trailing slash, lower case.
func normaliseHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimPrefix(strings.TrimPrefix(h, "https://"), "http://")
	return strings.TrimRight(h, "/")
}

// parseSweepSettings validates the environment. Its errors never repeat the
// host: CI logs are public.
func parseSweepSettings(env sweepEnvironment, apiHost string, now time.Time) (sweepSettings, error) {
	s := sweepSettings{mode: sweepMode(env.Mode), host: normaliseHost(apiHost), minAge: defaultSweepMinAge, max: defaultSweepMax, now: now}

	switch s.mode {
	case sweepDryRun, sweepDelete, sweepGPSnapshot:
	default:
		return s, errors.New("E2E_SWEEP must be exactly dry-run, delete or gp-snapshot")
	}

	if s.mode == sweepDelete {
		if env.ConfirmHost == "" {
			return s, errors.New("delete mode needs E2E_SWEEP_CONFIRM_HOST: type the appliance host yourself (host[:port], no scheme); never derive it from $BT_API_HOST")
		}
		if normaliseHost(env.ConfirmHost) != s.host {
			return s, errors.New("E2E_SWEEP_CONFIRM_HOST does not match BT_API_HOST; refusing to delete")
		}
	}

	if env.MinAge != "" {
		d, err := time.ParseDuration(env.MinAge)
		if err != nil || d < 0 {
			return s, fmt.Errorf("E2E_SWEEP_MIN_AGE must be a non-negative Go duration such as 24h, not %q", env.MinAge)
		}
		s.minAge = d
	}
	if s.mode == sweepDelete && s.minAge < deleteMinAgeFloor {
		s.minAge = deleteMinAgeFloor
	}

	if env.Max != "" {
		n, err := strconv.Atoi(env.Max)
		if err != nil || n < 0 {
			return s, fmt.Errorf("E2E_SWEEP_MAX must be a non-negative integer, not %q", env.Max)
		}
		s.max = n
	}

	if env.LegacyIDs != "" {
		if s.mode != sweepDelete {
			return s, errors.New("E2E_SWEEP_LEGACY_IDS applies only to E2E_SWEEP=delete")
		}
		f, err := os.Open(env.LegacyIDs)
		if err != nil {
			return s, fmt.Errorf("E2E_SWEEP_LEGACY_IDS: %w", err)
		}
		defer func() { _ = f.Close() }()
		if s.legacy, err = parseLegacyIDs(f, now); err != nil {
			return s, fmt.Errorf("E2E_SWEEP_LEGACY_IDS: %w", err)
		}
	}

	if env.CandidatesOut != "" {
		if s.mode != sweepDryRun {
			return s, errors.New("E2E_SWEEP_CANDIDATES_OUT applies only to E2E_SWEEP=dry-run")
		}
		s.candidatesOut = env.CandidatesOut
	}
	return s, nil
}

var legacyLineRE = regexp.MustCompile(`^([a-z_]+)\t([1-9][0-9]*)(\t# .*)?$`)

// parseLegacyIDs reads a reviewed candidates file, the dry-run's own format and
// nothing else: a "# listed-at <RFC 3339>" first line, then one "type<TAB>id"
// line per object, each optionally ending in "<TAB># name".
func parseLegacyIDs(r io.Reader, now time.Time) (map[sweepKey]bool, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() || !strings.HasPrefix(sc.Text(), listedAtPrefix) {
		return nil, errors.New("first line must be \"" + listedAtPrefix + "<RFC 3339 time>\", as the dry-run writes it")
	}
	listedAt, err := time.Parse(time.RFC3339, strings.TrimPrefix(sc.Text(), listedAtPrefix))
	if err != nil {
		return nil, fmt.Errorf("listed-at: %w", err)
	}
	if age := now.Sub(listedAt); age < legacyListMinAge {
		return nil, fmt.Errorf("the list is %s old; it must be at least %s old, so no run that could own a listed object is still going", age.Round(time.Minute), legacyListMinAge)
	}

	known := map[string]bool{}
	for _, st := range sweepTypes {
		known[st.name] = st.phase != neverSwept
	}
	ids := map[sweepKey]bool{}
	for line := 2; sc.Scan(); line++ {
		m := legacyLineRE.FindStringSubmatch(sc.Text())
		if m == nil || !known[m[1]] {
			return nil, fmt.Errorf("line %d is not \"type<TAB>id\" for a known type", line)
		}
		id, _ := strconv.Atoi(m[2])
		ids[sweepKey{m[1], id}] = true
	}
	return ids, sc.Err()
}

// sweepRow is one object as the list endpoints return it.
type sweepRow map[string]any

func (r sweepRow) str(k string) string {
	if v, ok := r[k].(string); ok {
		return v
	}
	return ""
}

func (r sweepRow) num(k string) (int, bool) {
	switch v := r[k].(type) {
	case float64:
		return int(v), v == float64(int(v))
	case string:
		n, err := strconv.Atoi(v)
		return n, err == nil
	}
	return 0, false
}

func (r sweepRow) id() int {
	n, _ := r.num("id")
	return n
}

func rowKey(r *sweepRow) string { return fmt.Sprint((*r)["id"]) }

type sweepKind int

const (
	kindAccount sweepKind = iota
	kindAccountGroup
	kindPolicy
	kindJumpItem
	kindInstaller
	kindJumpGroup
	kindJumpoint
	// kindJumpClient is listed only as a reference: deployed jump clients are
	// never swept.
	kindJumpClient
)

type sweepType struct {
	name     string
	endpoint string
	query    map[string]string
	kind     sweepKind
	// phase is the type's place in the deletion order: contents before their
	// containers, and account groups before the policies they might use.
	// neverSwept types are listed only for their references.
	phase int
	// groupField and jumpointField name the item's jump group and jumpoint
	// references, which oracle calls asset_group_id and gateway_id.
	groupField, groupTypeField, jumpointField string
}

const neverSwept = -1

func jumpItemType(name string, item api.APIResource) sweepType {
	return sweepType{name: name, endpoint: item.Endpoint(), kind: kindJumpItem, phase: 3,
		groupField: "jump_group_id", groupTypeField: "jump_group_type", jumpointField: "jumpoint_id"}
}

// sweepTypes in phase order, which classify and deleteEligible walk them in;
// TestSweepOrder holds them there.
var sweepTypes = []sweepType{
	// Personal accounts are hidden unless asked for, and an account policy's
	// delete re-points them too.
	{name: "vault_account", endpoint: api.VaultAccount{}.Endpoint(), query: map[string]string{"include_personal": "true"}, kind: kindAccount, phase: 0},
	{name: "vault_account_group", endpoint: api.VaultAccountGroup{}.Endpoint(), kind: kindAccountGroup, phase: 1},
	{name: "vault_account_policy", endpoint: api.VaultAccountPolicy{}.Endpoint(), kind: kindPolicy, phase: 2},
	jumpItemType("shell_jump", api.ShellJump{}),
	jumpItemType("remote_rdp", api.RemoteRDP{}),
	jumpItemType("remote_vnc", api.RemoteVNC{}),
	jumpItemType("protocol_tunnel_jump", api.ProtocolTunnelJump{}),
	jumpItemType("postgresql_tunnel_jump", api.PostgreSQLTunnelJump{}),
	jumpItemType("mysql_tunnel_jump", api.MySQLTunnelJump{}),
	jumpItemType("mongodb_tunnel_jump", api.MongodbTunnelJump{}),
	{name: "oracle_tunnel_jump", endpoint: api.OracleTunnelJump{}.Endpoint(), kind: kindJumpItem, phase: 3,
		groupField: "asset_group_id", groupTypeField: "asset_group_type", jumpointField: "gateway_id"},
	jumpItemType("network_tunnel_jump", api.NetworkTunnelJump{}),
	jumpItemType("web_jump", api.WebJump{}),
	{name: "jump_client_installer", endpoint: api.JumpClientInstaller{}.Endpoint(), kind: kindInstaller, phase: 3,
		groupField: "jump_group_id", groupTypeField: "jump_group_type"},
	{name: "jump_group", endpoint: api.JumpGroup{}.Endpoint(), kind: kindJumpGroup, phase: 4},
	{name: "jumpoint", endpoint: api.Jumpoint{}.Endpoint(), kind: kindJumpoint, phase: 4},
	// The provider has no jump client model.
	{name: "jump_client", endpoint: "jump-client", kind: kindJumpClient, phase: neverSwept, groupField: "jump_group_id", groupTypeField: "jump_group_type"},
}

// sweepRef is one way objects of another type can point at a container. The
// classify pass checks it against the collected lists; each delete checks it
// again against the appliance as it is then.
type sweepRef struct {
	from string // the referencing type
	// field is from's field that names the container: its id, or the
	// container's own key field (code_name, name) when key is set.
	field string
	key   string
	// unchecked, when set, is the reason to keep the container if a row of from
	// lacks field: an appliance that doesn't return it can't show the
	// reference is absent.
	unchecked string
	// filter: from's list endpoint filters on field, so the live check asks for
	// just the referencing rows. Any row it returns is a reference.
	filter bool
	verb   string // the reason's wording: "contains", "used by", "criteria of"

	// sharedJumpGroups, when set, replaces field: it reads the jump groups an
	// object's jump item association names in criteria.shared_jump_groups.
	// standalone marks from as accounts: only those that can hold their own
	// association are read (holdsOwnAssociation).
	sharedJumpGroups func(c *api.APIClient, id int) ([]int, error)
	standalone       bool
}

// holdsOwnAssociation says whether account r can have a jump item association
// of its own. VaultAccountJumpItemAssociationApiController::store refuses
// personal, x509_ca and windows_local accounts (and AccountRepo never stores one
// for the last two), and an account in any group but Default uses its group's.
func holdsOwnAssociation(r sweepRow) bool {
	if personal, _ := r["personal"].(bool); personal {
		return false
	}
	switch r.str("type") {
	case "x509_ca", "windows_local":
		return false
	}
	g, _ := r.num("account_group_id")
	return g <= 1
}

// references lists what may point at a container of type st.
func references(st *sweepType) []sweepRef {
	switch st.kind {
	case kindAccountGroup:
		// Not filtered: with account_group_id the appliance also drops x509_ca
		// accounts (VaultAccountApiController::index), which the group holds too.
		return []sweepRef{{from: "vault_account", field: "account_group_id", verb: "contains"}}
	case kindPolicy:
		return []sweepRef{
			{from: "vault_account", field: "account_policy", key: "code_name", verb: "used by"},
			{from: "vault_account_group", field: "account_policy", key: "code_name", verb: "used by"},
		}
	case kindJumpGroup, kindJumpoint:
		var refs []sweepRef
		for _, from := range typesOfKind(kindJumpItem, kindInstaller, kindJumpClient) {
			field := from.groupField
			if st.kind == kindJumpoint {
				field = from.jumpointField
			}
			if field != "" {
				refs = append(refs, sweepRef{from: from.name, field: field, filter: true, verb: "used by"})
			}
		}
		if st.kind == kindJumpoint {
			// A jump client or installer names its Jump Zone Proxy jumpoint by
			// name. The API calls the field jumpoint_proxy (proxy_push_agent_name
			// internally) and can't filter on it.
			for _, from := range []string{"jump_client_installer", "jump_client"} {
				refs = append(refs, sweepRef{from: from, field: "jumpoint_proxy", key: "name",
					unchecked: "proxy-push refs unchecked", verb: "used by"})
			}
		}
		if st.kind == kindJumpGroup {
			refs = append(refs,
				sweepRef{from: "vault_account_group", verb: "criteria of", sharedJumpGroups: accountGroupSharedJumpGroups},
				sweepRef{from: "vault_account", verb: "criteria of", sharedJumpGroups: accountSharedJumpGroups, standalone: true})
		}
		return refs
	}
	return nil
}

func accountGroupSharedJumpGroups(c *api.APIClient, id int) ([]int, error) {
	jia, err := api.GetItemEndpoint[api.AccountGroupJumpItemAssociation](c, api.AccountGroupJumpItemAssociation{ID: &id}.Endpoint())
	if err != nil || jia.Criteria == nil {
		return nil, err
	}
	return jia.Criteria.SharedJumpGroups, nil
}

func accountSharedJumpGroups(c *api.APIClient, id int) ([]int, error) {
	jia, err := api.GetItemEndpoint[api.AccountJumpItemAssociation](c, api.AccountJumpItemAssociation{ID: &id}.Endpoint())
	if err != nil || jia.Criteria == nil {
		return nil, err
	}
	return jia.Criteria.SharedJumpGroups, nil
}

// --- matchers ---------------------------------------------------------------
//
// Every match is an anchored, full-string regular expression. A fixture name is
// generic ("This is a Name"), so only the marker beside it says the object is a
// test's, and a Contains check would match a real object that happens to
// mention one.

// fixtureNames are the name values tests give fixtures: the names they pass,
// and the variables.tf defaults they leave in place. TestSweepCoversFixtures
// renders every fixture against the real test bindings, so a new name fails
// there rather than leaking silently.
var fixtureNames = []string{
	"This is a Name",
	"This is a Name2",
	"This is a Renamed Name",
	"fun_policy",
	"fun_account",
	"fun_jump",
}

var (
	fixtureNameAlt = func() string {
		q := make([]string, len(fixtureNames))
		for i, n := range fixtureNames {
			q[i] = regexp.QuoteMeta(n)
		}
		return "(?:" + strings.Join(q, "|") + ")"
	}()

	accountGroupNameRE = regexp.MustCompile(`^` + fixtureNameAlt + ` ` + markerToken + `(?: (?:Nothing|GP|JIA))?$`)
	toggleGroupNameRE  = regexp.MustCompile(`^This is a Name ` + markerToken + ` Toggle$`)
	policyNameRE       = regexp.MustCompile(`^` + fixtureNameAlt + ` ` + markerToken + `(?: False| Mixed)?$`)
	accountNameRE      = regexp.MustCompile(`^(?:Group UP|Standalone UP(?: GP| JIA| Both)?|Group Key|Standalone Key(?: GP| JIA| Both| Any| None)?|Standalone CA Key|Standalone CA|Group TOKEN|Standalone TOKEN(?: GP| JIA| Both)?) ` +
		fixtureNameAlt + ` ` + markerToken + `$`)
	jumpGroupOrPointNameRE = regexp.MustCompile(`^Test (JG|JP)( GP)? ` + markerToken + `$`)
	itemNameRE             = regexp.MustCompile(`^` + fixtureNameAlt + `(?:-secondary)?$`)
	itemTagRE              = regexp.MustCompile(`^` + markerToken + `(?:-updated)?$`)
)

// sweepAccountTypesWithUsername are the account types whose username the list
// returns. Token accounts have none.
var sweepAccountTypesWithUsername = map[string]bool{"username_password": true, "ssh": true, "ssh_ca": true}

// matchFixture returns the marker a fixture object carries, or ok == false if it
// is not a fixture object. groups maps fixture jump group ids to their markers,
// for the jump items and installers that must sit in one.
func matchFixture(st *sweepType, row sweepRow, groups map[int]string) (marker string, ok bool) {
	name := row.str("name")
	switch st.kind {
	case kindAccountGroup:
		if row.id() == 1 { // the Default group
			return "", false
		}
		m := accountGroupNameRE.FindStringSubmatch(name)
		if m == nil {
			m = toggleGroupNameRE.FindStringSubmatch(name)
		}
		if m == nil || row.str("description") != m[1] {
			return "", false
		}
		return m[1], true

	case kindPolicy:
		m := policyNameRE.FindStringSubmatch(name)
		if m == nil {
			return "", false
		}
		switch row.str("code_name") {
		case m[1], m[1] + "_false", m[1] + "_mixed":
			return m[1], true
		}
		return "", false

	case kindAccount:
		m := accountNameRE.FindStringSubmatch(name)
		if m == nil {
			return "", false
		}
		if sweepAccountTypesWithUsername[row.str("type")] {
			switch row.str("username") {
			case m[1], m[1] + "any", m[1] + "none":
			default:
				return "", false
			}
		}
		return m[1], true

	case kindJumpGroup, kindJumpoint:
		abbrev := "JG"
		if st.kind == kindJumpoint {
			abbrev = "JP"
		}
		m := jumpGroupOrPointNameRE.FindStringSubmatch(name)
		code := row.str("code_name")
		if m == nil || m[1] != abbrev || code == sentinelCodeName {
			return "", false
		}
		want := "example_" + m[3]
		if m[2] != "" {
			want = "gp_" + want
		}
		if code != want {
			return "", false
		}
		return m[3], true

	case kindJumpItem, kindInstaller:
		if !itemNameRE.MatchString(name) {
			return "", false
		}
		m := itemTagRE.FindStringSubmatch(row.str("tag"))
		if m == nil {
			return "", false
		}
		// The jump group id is only meaningful for a shared group: a personal
		// group's id is a different namespace.
		group, ok := row.num(st.groupField)
		if !ok || row.str(st.groupTypeField) != "shared" || groups[group] != m[1] {
			return "", false
		}
		return m[1], true
	}
	return "", false
}

// --- the sweep --------------------------------------------------------------

type candidate struct {
	st     *sweepType
	row    sweepRow
	id     int
	marker string
	// decision is "eligible", "legacy-candidate" or "skip".
	decision string
	reason   string
}

type sweeper struct {
	c    *api.APIClient
	set  sweepSettings
	logf func(format string, args ...any)

	rows   map[string][]sweepRow
	absent map[string]bool
	failed map[string]error

	// groups maps each fixture jump group's id to its marker.
	groups     map[int]string
	candidates map[string][]*candidate
	eligible   map[sweepKey]bool

	// inventoryJIA memoises, per referencing type, the jump item associations
	// the classify pass read: one read serves every jump group.
	inventoryJIA map[string]jiaRefs
}

// jiaRefs maps each jump group named in criteria.shared_jump_groups to one
// object naming it.
type jiaRefs struct {
	by  map[int]int
	err error
}

func typeByName(name string) *sweepType {
	for i := range sweepTypes {
		if sweepTypes[i].name == name {
			return &sweepTypes[i]
		}
	}
	return nil
}

func typesOfKind(kinds ...sweepKind) []*sweepType {
	var out []*sweepType
	for i := range sweepTypes {
		for _, k := range kinds {
			if sweepTypes[i].kind == k {
				out = append(out, &sweepTypes[i])
			}
		}
	}
	return out
}

// runSweep runs one sweep. Lines go to logf: one per fixture candidate, and
// per-type counts. Nothing that failed to match a fixture is ever named.
func runSweep(c *api.APIClient, set sweepSettings, logf func(string, ...any)) error {
	s := &sweeper{c: c, set: set, logf: logf,
		rows: map[string][]sweepRow{}, absent: map[string]bool{}, failed: map[string]error{},
		groups: map[int]string{}, candidates: map[string][]*candidate{},
		eligible: map[sweepKey]bool{}, inventoryJIA: map[string]jiaRefs{}}

	if set.mode == sweepDelete {
		if err := s.checkSentinel(); err != nil {
			return err
		}
	}

	if set.mode == sweepGPSnapshot {
		s.collect(gpRelationTypes()...)
		return s.gpSnapshot()
	}
	s.collect(sweepTypes...)
	s.classify()
	over := s.report()

	if set.mode == sweepDryRun {
		if len(over) > 0 {
			logf("sweep WARNING: more than E2E_SWEEP_MAX=%d would be deleted for %s; delete mode would refuse", set.max, strings.Join(over, ", "))
		}
		return s.writeCandidates()
	}
	if len(over) > 0 {
		return fmt.Errorf("refusing to delete: more than E2E_SWEEP_MAX=%d eligible for %s", set.max, strings.Join(over, ", "))
	}
	return s.deleteEligible()
}

// checkSentinel requires the appliance-side opt-in: a jump group the sweeper
// will never delete, whose comments name this host. A restored backup or a
// cloned appliance on another host carries the group but not a matching host.
func (s *sweeper) checkSentinel() error {
	groups, err := api.ListAllItemsEndpoint(s.c, api.JumpGroup{}.Endpoint(), map[string]string{"code_name": sentinelCodeName}, rowKey)
	if err != nil {
		return fmt.Errorf("could not look up the %s jump group: %w", sentinelCodeName, err)
	}
	found := 0
	for _, g := range groups {
		if g.str("code_name") != sentinelCodeName {
			continue
		}
		found++
		if normaliseHost(g.str("comments")) != s.set.host {
			return fmt.Errorf("the %s jump group's comments do not name this host; refusing to delete", sentinelCodeName)
		}
	}
	if found != 1 {
		return fmt.Errorf("delete mode needs exactly one jump group with code_name %s on the appliance (found %d); see CONTRIBUTING.md", sentinelCodeName, found)
	}
	return nil
}

// collect lists each type in full before anything is decided. A 404 on an
// inventory list means this appliance has no such type (RS has no PRA-only
// items). Any other failure, an incomplete list included, means no deletions
// for that type or for any type whose checks rely on it.
func (s *sweeper) collect(types ...sweepType) {
	for _, st := range types {
		rows, err := api.ListAllItemsEndpoint(s.c, st.endpoint, st.query, rowKey)
		switch {
		case api.IsNotFound(err):
			s.absent[st.name] = true
		case err != nil:
			s.failed[st.name] = err
		default:
			s.rows[st.name] = rows
		}
	}
	for _, g := range s.rows["jump_group"] {
		if m, ok := matchFixture(typeByName("jump_group"), g, nil); ok {
			s.groups[g.id()] = m
		}
	}
}

// dependsOn lists the types whose lists st's reference checks read.
func dependsOn(st *sweepType) []*sweepType {
	var deps []*sweepType
	seen := map[string]bool{}
	for _, ref := range references(st) {
		if !seen[ref.from] {
			seen[ref.from] = true
			deps = append(deps, typeByName(ref.from))
		}
	}
	return deps
}

// classify decides each fixture candidate, type by type in phase order, so the
// reference checks of a container see the final decisions on its contents.
func (s *sweeper) classify() {
	for i := range sweepTypes {
		st := &sweepTypes[i]
		if st.phase == neverSwept {
			continue
		}
		for _, row := range s.rows[st.name] {
			marker, ok := matchFixture(st, row, s.groups)
			if !ok {
				continue
			}
			cand := &candidate{st: st, row: row, id: row.id(), marker: marker}
			cand.decision, cand.reason = s.ageDecision(st, cand.id, marker)
			if cand.decision == "eligible" {
				if dep := s.failedDependency(st); dep != "" {
					cand.decision, cand.reason = "skip", "list-failed:"+dep
				} else if reason := s.blocked(cand, false); reason != "" {
					cand.decision, cand.reason = "skip", reason
				} else {
					s.eligible[sweepKey{st.name, cand.id}] = true
				}
			}
			s.candidates[st.name] = append(s.candidates[st.name], cand)
		}
	}
}

// ageDecision applies the age rule for timestamped markers and the review rule
// for legacy ones.
func (s *sweeper) ageDecision(st *sweepType, id int, marker string) (decision, reason string) {
	kind, at := parseMarker(marker)
	switch kind {
	case markerTimestamped:
		age := s.set.now.Sub(at)
		switch {
		case age < -futureSkew:
			return "skip", "marker-in-future"
		case age < s.set.minAge:
			return "skip", "too-young"
		}
		return "eligible", ""
	case markerLegacy:
		if s.set.legacy[sweepKey{st.name, id}] {
			return "eligible", "listed"
		}
		return "legacy-candidate", "no-creation-time"
	}
	return "skip", "marker-unusable"
}

func (s *sweeper) failedDependency(st *sweepType) string {
	if s.failed[st.name] != nil {
		return st.name
	}
	for _, dep := range dependsOn(st) {
		if s.failed[dep.name] != nil {
			return dep.name
		}
	}
	return ""
}

// blocked returns why something still points at the container cand, or "". The
// classify pass (live false) reads the collected lists and ignores what this
// run will delete; a delete (live true) asks the appliance as it is now. In a
// live check a 404 keeps the container: it may be a wrong path, and a wrong
// path must not remove a safety check. Types the inventory found absent are not
// queried.
func (s *sweeper) blocked(cand *candidate, live bool) string {
	for _, ref := range references(cand.st) {
		from := typeByName(ref.from)
		if live && s.absent[from.name] {
			continue
		}
		var by int
		var err error
		if ref.sharedJumpGroups != nil {
			by, err = s.namedInCriteria(ref, from, cand.id, live)
			if err != nil {
				return "jump-item-association-check-failed"
			}
		} else {
			by, err = s.referencing(ref, from, cand, live)
			if errors.Is(err, errRefUnchecked) {
				return ref.unchecked
			}
			if err != nil {
				return "reference-check-failed " + from.name
			}
		}
		if by != 0 {
			return fmt.Sprintf("%s %s %d", ref.verb, from.name, by)
		}
	}
	return ""
}

// rowsOf is from's rows: as collected, less those this run deletes, or listed
// now with filter added to from's own query.
func (s *sweeper) rowsOf(from *sweepType, live bool, filter map[string]string) ([]sweepRow, error) {
	if !live {
		var kept []sweepRow
		for _, r := range s.rows[from.name] {
			if !s.eligible[sweepKey{from.name, r.id()}] {
				kept = append(kept, r)
			}
		}
		return kept, nil
	}
	query := maps.Clone(from.query)
	if len(filter) > 0 {
		if query == nil {
			query = map[string]string{}
		}
		maps.Copy(query, filter)
	}
	return api.ListAllItemsEndpoint(s.c, from.endpoint, query, rowKey)
}

// errRefUnchecked: a row of from lacks the field a reference with unchecked set
// reads.
var errRefUnchecked = errors.New("reference field not returned")

// referencing returns the id of a row of from whose ref.field names cand, or 0.
func (s *sweeper) referencing(ref sweepRef, from *sweepType, cand *candidate, live bool) (int, error) {
	var filter map[string]string
	if live && ref.filter {
		filter = map[string]string{ref.field: strconv.Itoa(cand.id)}
	}
	rows, err := s.rowsOf(from, live, filter)
	if err != nil {
		return 0, err
	}
	for _, r := range rows {
		if _, present := r[ref.field]; ref.unchecked != "" && !present {
			return 0, errRefUnchecked
		}
		var match bool
		switch {
		case filter != nil:
			match = true
		case ref.key != "":
			match = r.str(ref.field) == cand.row.str(ref.key)
		default:
			v, ok := r.num(ref.field)
			match = ok && v == cand.id
		}
		if match {
			return r.id(), nil
		}
	}
	return 0, nil
}

// namedInCriteria returns the id of a row of from whose jump item association
// names jump group id, or 0. The classify pass reads each association once.
func (s *sweeper) namedInCriteria(ref sweepRef, from *sweepType, id int, live bool) (int, error) {
	refs, ok := s.inventoryJIA[from.name]
	if live || !ok {
		refs = s.readSharedJumpGroups(ref, from, live)
		if !live {
			s.inventoryJIA[from.name] = refs
		}
	}
	return refs.by[id], refs.err
}

func (s *sweeper) readSharedJumpGroups(ref sweepRef, from *sweepType, live bool) jiaRefs {
	rows, err := s.rowsOf(from, live, nil)
	if err != nil {
		return jiaRefs{err: err}
	}
	by := map[int]int{}
	read, notFound := 0, 0
	for _, r := range rows {
		if ref.standalone && !holdsOwnAssociation(r) {
			continue
		}
		read++
		groups, err := ref.sharedJumpGroups(s.c, r.id())
		if api.IsNotFound(err) {
			// This object has no association.
			notFound++
			continue
		}
		if err != nil {
			// A body that does not decode is an error too, so it keeps every
			// jump group.
			return jiaRefs{err: err}
		}
		for _, g := range groups {
			by[g] = r.id()
		}
	}
	if !ref.standalone && read > 0 && notFound == read {
		// Every account group is created with an association (AccountGroupRepo),
		// so a 404 for every group is more likely a wrong path, which must not
		// remove the check. Most accounts have none, so for accounts every read
		// 404ing is normal.
		return jiaRefs{err: fmt.Errorf("every %s association read was a 404", from.name)}
	}
	return jiaRefs{by: by}
}

// report prints the candidate lines and per-type counts, and returns the types
// whose eligible count is over the cap.
func (s *sweeper) report() []string {
	var over []string
	for i := range sweepTypes {
		st := &sweepTypes[i]
		if st.phase == neverSwept {
			continue
		}
		excluded := map[string]int{}
		eligible, legacy := 0, 0
		for _, c := range s.candidates[st.name] {
			// Fixture candidates only: the name matched a fixture pattern.
			s.logf("sweep %s %d %q %s %s", st.name, c.id, c.row.str("name"), c.decision, c.reason)
			switch c.decision {
			case "eligible":
				eligible++
			case "legacy-candidate":
				legacy++
			default:
				excluded[strings.SplitN(c.reason, " ", 2)[0]]++
			}
		}
		var parts []string
		for reason, n := range excluded {
			parts = append(parts, fmt.Sprintf("%s:%d", reason, n))
		}
		sort.Strings(parts)
		state := "listed"
		switch {
		case s.absent[st.name]:
			state = "absent"
		case s.failed[st.name] != nil:
			state = "list-failed"
		}
		// No total: how many objects a type has, personal accounts included,
		// says something about the appliance that CI logs, which are public,
		// should not.
		s.logf("sweep-total %s %s matched=%d eligible=%d legacy-candidate=%d excluded=[%s]",
			st.name, state, len(s.candidates[st.name]), eligible, legacy, strings.Join(parts, " "))
		if eligible > s.set.max {
			over = append(over, st.name)
		}
	}
	for name, err := range s.failed {
		s.logf("sweep-list-failed %s: %s", name, failureClass(err))
	}
	return over
}

// failureClass names what kind of failure an API call or api.NewClient had,
// and nothing else. The error itself can carry the response body, the token
// URL, or, for a transport error, the request URL with the appliance's host.
func failureClass(err error) string {
	var status *api.StatusError
	var token *oauth2.RetrieveError
	var unstable *api.ErrUnstableList
	var transport *url.Error
	switch {
	case errors.As(err, &status):
		return "status " + strconv.Itoa(status.Status)
	case errors.As(err, &token) && token.Response != nil:
		return "token request status " + strconv.Itoa(token.Response.StatusCode)
	case errors.As(err, &unstable):
		switch unstable.Cause {
		case api.ListOverlapped:
			return "unstable list: pages overlapped"
		case api.ListPageCap:
			return "unstable list: page limit"
		}
		return "unstable list: changed while read"
	case errors.As(err, &transport):
		if transport.Op == "parse" {
			return "invalid host"
		}
		return "transport error"
	}
	return "other error"
}

// writeCandidates writes the legacy candidates for human review, in the format
// parseLegacyIDs reads. The listed-at time is when listing started.
func (s *sweeper) writeCandidates() error {
	if s.set.candidatesOut == "" {
		return nil
	}
	var b strings.Builder
	b.WriteString(listedAtPrefix + s.set.now.UTC().Format(time.RFC3339) + "\n")
	for i := range sweepTypes {
		for _, c := range s.candidates[sweepTypes[i].name] {
			if c.decision == "legacy-candidate" {
				fmt.Fprintf(&b, "%s\t%d\t# %s\n", c.st.name, c.id, c.row.str("name"))
			}
		}
	}
	return os.WriteFile(s.set.candidatesOut, []byte(b.String()), 0o600)
}

// deleteEligible deletes in phase order. Each delete re-reads the object,
// re-matches it, and re-runs the container's reference checks against the live
// appliance: anything that changed since collection keeps the object.
func (s *sweeper) deleteEligible() error {
	var errs []error
	for i := range sweepTypes {
		st := &sweepTypes[i]
		for _, cand := range s.candidates[st.name] {
			if !s.eligible[sweepKey{st.name, cand.id}] {
				continue
			}
			ok := func(item *sweepRow) error {
				marker, matched := matchFixture(st, *item, s.groups)
				if !matched || marker != cand.marker {
					return errors.New("no-longer-matches")
				}
				if decision, _ := s.ageDecision(st, cand.id, marker); decision != "eligible" {
					return errors.New("no-longer-eligible")
				}
				if reason := s.blocked(cand, true); reason != "" {
					return errors.New(reason)
				}
				return nil
			}
			outcome, err := deleteIfStill(s.c, st.endpoint, cand.id, ok)
			switch {
			case err != nil:
				s.logf("sweep %s %d delete-failed", st.name, cand.id)
				errs = append(errs, fmt.Errorf("%s %d: %w", st.name, cand.id, err))
			case outcome.refused != nil:
				s.logf("sweep %s %d kept-at-delete %s", st.name, cand.id, outcome.refused)
			default:
				s.logf("sweep %s %d %s", st.name, cand.id, outcome.result)
			}
		}
	}
	return errors.Join(errs...)
}

// gpRelation is one member type of a group policy.
type gpRelation struct {
	path, field, typ string
	endpoint         func(gp string) string
}

var gpRelations = []gpRelation{
	{"jump-group", "jump_group_id", "jump_group", func(gp string) string { return api.GroupPolicyJumpGroup{}.WithGroupPolicyID(gp).Endpoint() }},
	{"jumpoint", "jumpoint_id", "jumpoint", func(gp string) string { return api.GroupPolicyJumpoint{}.WithGroupPolicyID(gp).Endpoint() }},
	{"vault-account-group", "account_group_id", "vault_account_group", func(gp string) string {
		return api.GroupPolicyVaultAccountGroup{}.WithGroupPolicyID(gp).Endpoint()
	}},
	{"vault-account", "account_id", "vault_account", func(gp string) string { return api.GroupPolicyVaultAccount{}.WithGroupPolicyID(gp).Endpoint() }},
}

// gpRelationTypes are the types gp-snapshot mode collects: the members'.
func gpRelationTypes() []sweepType {
	types := make([]sweepType, len(gpRelations))
	for i, rel := range gpRelations {
		types[i] = *typeByName(rel.typ)
	}
	return types
}

// gpSnapshot records, read-only, every member of the group policy the fixtures
// write memberships on (data.sra_group_policy_list.gp.items[0]): ids and
// whether each is a fixture object, never a name.
func (s *sweeper) gpSnapshot() error {
	gps, err := api.ListAllItemsEndpoint(s.c, api.GroupPolicy{}.Endpoint(), nil, rowKey)
	if err != nil {
		return fmt.Errorf("listing group policies: %w", err)
	}
	if len(gps) == 0 {
		return errors.New("no group policies")
	}
	gp := strconv.Itoa(gps[0].id())
	for _, rel := range gpRelations {
		key := func(r *sweepRow) string { return fmt.Sprint((*r)[rel.field]) }
		members, err := api.ListAllItemsEndpoint(s.c, rel.endpoint(gp), nil, key)
		if err != nil {
			return fmt.Errorf("listing %s members: %w", rel.path, err)
		}
		if s.failed[rel.typ] != nil {
			return fmt.Errorf("listing %s: %w", rel.typ, s.failed[rel.typ])
		}
		byID := map[int]sweepRow{}
		for _, r := range s.rows[rel.typ] {
			byID[r.id()] = r
		}
		ids := []int{}
		for _, m := range members {
			if id, ok := m.num(rel.field); ok {
				ids = append(ids, id)
			}
		}
		sort.Ints(ids)
		fixture := 0
		for _, id := range ids {
			class := "non-fixture"
			if row, ok := byID[id]; ok {
				if _, matched := matchFixture(typeByName(rel.typ), row, s.groups); matched {
					class = "fixture"
					fixture++
				}
			}
			s.logf("gp-snapshot %s %d %s", rel.path, id, class)
		}
		s.logf("gp-snapshot-total %s members=%d fixture=%d non-fixture=%d", rel.path, len(ids), fixture, len(ids)-fixture)
	}
	return nil
}

// deleteOutcome says what deleteIfStill did when it returned no error: refused
// is why ok kept the object, or nil if it was deleted or already gone.
type deleteOutcome struct {
	refused error
	result  string // "deleted" or "already-gone"
}

// deleteIfStill re-reads object id at endpoint, asks ok whether it may still be
// deleted, and deletes it. A 404 on either call means it is already gone, which
// is the state the caller wants. Both the import guard and the sweeper use it,
// so neither deletes on what an object looked like earlier.
func deleteIfStill[I any](c *api.APIClient, endpoint string, id int, ok func(*I) error) (deleteOutcome, error) {
	path := fmt.Sprintf("%s/%d", endpoint, id)
	item, err := api.GetItemEndpoint[I](c, path)
	switch {
	case api.IsNotFound(err):
		return deleteOutcome{result: "already-gone"}, nil
	case err != nil:
		return deleteOutcome{}, fmt.Errorf("re-reading before delete: %w", err)
	}
	if refused := ok(item); refused != nil {
		return deleteOutcome{refused: refused}, nil
	}
	switch err := api.DeleteItemEndpoint(c, path); {
	case err == nil:
		return deleteOutcome{result: "deleted"}, nil
	case api.IsNotFound(err):
		return deleteOutcome{result: "already-gone"}, nil
	default:
		return deleteOutcome{}, err
	}
}
