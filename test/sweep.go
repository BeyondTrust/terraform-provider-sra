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
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"terraform-provider-sra/api"
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
		known[st.name] = sweepPhase(st.kind) >= 0
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

// sweepRow is one object as the list endpoints return it. The Endpoint method
// exists only so api.GetItemEndpoint will decode into it.
type sweepRow map[string]any

func (sweepRow) Endpoint() string { return "" }

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
	// groupField and jumpointField name the item's jump group and jumpoint
	// references, which oracle calls asset_group_id and gateway_id.
	groupField, groupTypeField, jumpointField string
}

func jumpItemType(name, path string) sweepType {
	return sweepType{name: name, endpoint: "jump-item/" + path, kind: kindJumpItem,
		groupField: "jump_group_id", groupTypeField: "jump_group_type", jumpointField: "jumpoint_id"}
}

// sweepTypes in deletion order: contents before their containers, and
// account groups before the policies they might use.
var sweepTypes = []sweepType{
	// Personal accounts are hidden unless asked for, and an account policy's
	// delete re-points them too.
	{name: "vault_account", endpoint: "vault/account", query: map[string]string{"include_personal": "true"}, kind: kindAccount},
	{name: "vault_account_group", endpoint: "vault/account-group", kind: kindAccountGroup},
	{name: "vault_account_policy", endpoint: "vault/account-policy", kind: kindPolicy},
	jumpItemType("shell_jump", "shell-jump"),
	jumpItemType("remote_rdp", "remote-rdp"),
	jumpItemType("remote_vnc", "remote-vnc"),
	jumpItemType("protocol_tunnel_jump", "protocol-tunnel-jump"),
	jumpItemType("postgresql_tunnel_jump", "psql-tunnel-jump"),
	jumpItemType("mysql_tunnel_jump", "mysql-tunnel-jump"),
	jumpItemType("mongodb_tunnel_jump", "mongodb-tunnel-jump"),
	{name: "oracle_tunnel_jump", endpoint: "jump-item/oracle-tunnel-jump", kind: kindJumpItem,
		groupField: "asset_group_id", groupTypeField: "asset_group_type", jumpointField: "gateway_id"},
	jumpItemType("network_tunnel_jump", "network-tunnel-jump"),
	jumpItemType("web_jump", "web-jump"),
	{name: "jump_client_installer", endpoint: "jump-client/installer", kind: kindInstaller,
		groupField: "jump_group_id", groupTypeField: "jump_group_type"},
	{name: "jump_group", endpoint: "jump-group", kind: kindJumpGroup},
	{name: "jumpoint", endpoint: "jumpoint", kind: kindJumpoint},
	{name: "jump_client", endpoint: "jump-client", kind: kindJumpClient, groupField: "jump_group_id", groupTypeField: "jump_group_type"},
}

// sweepPhase is a type's place in the deletion order.
func sweepPhase(k sweepKind) int {
	switch k {
	case kindAccount:
		return 0
	case kindAccountGroup:
		return 1
	case kindPolicy:
		return 2
	case kindJumpItem, kindInstaller:
		return 3
	case kindJumpGroup, kindJumpoint:
		return 4
	}
	return -1 // never deleted
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

const markerToken = `(e2e[0-9a-z]{12}2?|[0-9a-z]{6}2?)`

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
	jumpGroupNameRE = regexp.MustCompile(`^Test JG( GP)? ` + markerToken + `$`)
	jumpointNameRE  = regexp.MustCompile(`^Test JP( GP)? ` + markerToken + `$`)
	itemNameRE      = regexp.MustCompile(`^` + fixtureNameAlt + `(?:-secondary)?$`)
	itemTagRE       = regexp.MustCompile(`^` + markerToken + `(?:-updated)?$`)
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
		re := jumpGroupNameRE
		if st.kind == kindJumpoint {
			re = jumpointNameRE
		}
		m := re.FindStringSubmatch(name)
		code := row.str("code_name")
		if m == nil || code == sentinelCodeName {
			return "", false
		}
		want := "example_" + m[2]
		if m[1] != "" {
			want = "gp_" + want
		}
		if code != want {
			return "", false
		}
		return m[2], true

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
	kind   markerKind
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
	deleted    map[sweepKey]bool
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
		eligible: map[sweepKey]bool{}, deleted: map[sweepKey]bool{}}

	if set.mode == sweepDelete {
		if err := s.checkSentinel(); err != nil {
			return err
		}
	}

	s.collect()
	if set.mode == sweepGPSnapshot {
		return s.gpSnapshot()
	}
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
	groups, err := api.ListAllItemsEndpoint(s.c, "jump-group", map[string]string{"code_name": sentinelCodeName}, rowKey)
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

// collect lists every type in full before anything is decided. A 404 on an
// inventory list means this appliance has no such type (RS has no PRA-only
// items). Any other failure, an incomplete list included, means no deletions
// for that type or for any type whose checks rely on it.
func (s *sweeper) collect() {
	for i := range sweepTypes {
		st := &sweepTypes[i]
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

// dependsOn lists the types whose lists a type's safety checks read.
func dependsOn(st *sweepType) []*sweepType {
	switch st.kind {
	case kindAccountGroup:
		return typesOfKind(kindAccount)
	case kindPolicy:
		return typesOfKind(kindAccount, kindAccountGroup)
	case kindJumpItem, kindInstaller:
		return typesOfKind(kindJumpGroup)
	case kindJumpGroup:
		return typesOfKind(kindJumpItem, kindInstaller, kindJumpClient, kindAccount, kindAccountGroup)
	case kindJumpoint:
		return typesOfKind(kindJumpItem)
	}
	return nil
}

func (s *sweeper) classify() {
	var byPhase [5][]*sweepType
	for i := range sweepTypes {
		if p := sweepPhase(sweepTypes[i].kind); p >= 0 {
			byPhase[p] = append(byPhase[p], &sweepTypes[i])
		}
	}

	for _, phase := range byPhase {
		var containers []*candidate
		for _, st := range phase {
			for _, row := range s.rows[st.name] {
				marker, ok := matchFixture(st, row, s.groups)
				if !ok {
					continue
				}
				cand := &candidate{st: st, row: row, id: row.id(), marker: marker}
				cand.kind, cand.decision, cand.reason = s.ageDecision(st, cand.id, marker)
				if cand.decision == "eligible" {
					if dep := s.failedDependency(st); dep != "" {
						cand.decision, cand.reason = "skip", "list-failed:"+dep
					}
				}
				if cand.decision == "eligible" {
					containers = append(containers, cand)
				}
				s.candidates[st.name] = append(s.candidates[st.name], cand)
			}
		}

		var jia map[int]string
		var jiaErr error
		for _, cand := range containers {
			reason := s.blockedByInventory(cand)
			if reason == "" && cand.st.kind == kindJumpGroup {
				if jia == nil && jiaErr == nil {
					jia, jiaErr = s.sharedJumpGroupReferences()
				}
				reason = jiaReason(jia, jiaErr, cand.id)
			}
			if reason != "" {
				cand.decision, cand.reason = "skip", reason
				continue
			}
			s.eligible[sweepKey{cand.st.name, cand.id}] = true
		}
	}
}

// ageDecision applies the age rule for timestamped markers and the review rule
// for legacy ones.
func (s *sweeper) ageDecision(st *sweepType, id int, marker string) (markerKind, string, string) {
	kind, _, _, at := parseMarker(marker)
	switch kind {
	case markerTimestamped:
		age := s.set.now.Sub(at)
		switch {
		case age < -futureSkew:
			return kind, "skip", "marker-in-future"
		case age < s.set.minAge:
			return kind, "skip", "too-young"
		}
		return kind, "eligible", ""
	case markerLegacy:
		if s.set.legacy[sweepKey{st.name, id}] {
			return kind, "eligible", "listed"
		}
		return kind, "legacy-candidate", "no-creation-time"
	}
	return kind, "skip", "marker-unusable"
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

func (s *sweeper) willDelete(typ string, id int) bool {
	return s.eligible[sweepKey{typ, id}] || s.deleted[sweepKey{typ, id}]
}

// blockedByInventory is a container's reference check against the collected
// lists: anything still pointing at it that this run will not also delete.
func (s *sweeper) blockedByInventory(cand *candidate) string {
	switch cand.st.kind {
	case kindAccountGroup:
		for _, a := range s.rows["vault_account"] {
			if g, _ := a.num("account_group_id"); g == cand.id && !s.willDelete("vault_account", a.id()) {
				return fmt.Sprintf("contains vault_account %d", a.id())
			}
		}
	case kindPolicy:
		code := cand.row.str("code_name")
		for _, typ := range []string{"vault_account", "vault_account_group"} {
			for _, r := range s.rows[typ] {
				if r.str("account_policy") == code && !s.willDelete(typ, r.id()) {
					return fmt.Sprintf("used by %s %d", typ, r.id())
				}
			}
		}
	case kindJumpGroup, kindJumpoint:
		for _, st := range typesOfKind(kindJumpItem, kindInstaller, kindJumpClient) {
			field := st.groupField
			if cand.st.kind == kindJumpoint {
				field = st.jumpointField
			}
			if field == "" {
				continue
			}
			for _, r := range s.rows[st.name] {
				if v, ok := r.num(field); ok && v == cand.id && !s.willDelete(st.name, r.id()) {
					return fmt.Sprintf("used by %s %d", st.name, r.id())
				}
			}
		}
	}
	return ""
}

// sharedJumpGroupReferences maps each jump group named in a jump item
// association's criteria.shared_jump_groups to one object naming it, across
// every account group and standalone account the run will keep. Accounts in a
// group other than Default inherit the group's association, so their own is not
// read.
func (s *sweeper) sharedJumpGroupReferences() (map[int]string, error) {
	refs := map[int]string{}
	read := func(typ, endpoint string, id int) error {
		item, err := api.GetItemEndpoint[sweepRow](s.c, fmt.Sprintf("%s/%d/jump-item-association", endpoint, id))
		if api.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if c, ok := (*item)["criteria"].(map[string]any); ok {
			if groups, ok := c["shared_jump_groups"].([]any); ok {
				for _, g := range groups {
					if f, ok := g.(float64); ok {
						refs[int(f)] = fmt.Sprintf("%s %d", typ, id)
					}
				}
			}
		}
		return nil
	}
	for _, g := range s.rows["vault_account_group"] {
		if !s.willDelete("vault_account_group", g.id()) {
			if err := read("vault_account_group", "vault/account-group", g.id()); err != nil {
				return nil, err
			}
		}
	}
	for _, a := range s.rows["vault_account"] {
		if g, _ := a.num("account_group_id"); g <= 1 && !s.willDelete("vault_account", a.id()) {
			if err := read("vault_account", "vault/account", a.id()); err != nil {
				return nil, err
			}
		}
	}
	return refs, nil
}

func jiaReason(refs map[int]string, err error, id int) string {
	if err != nil {
		return "jump-item-association-check-failed"
	}
	if by, ok := refs[id]; ok {
		return "criteria of " + by
	}
	return ""
}

// report prints the candidate lines and per-type counts, and returns the types
// whose eligible count is over the cap.
func (s *sweeper) report() []string {
	var over []string
	for i := range sweepTypes {
		st := &sweepTypes[i]
		if sweepPhase(st.kind) < 0 {
			continue
		}
		total := len(s.rows[st.name])
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
		s.logf("sweep-total %s %s total=%d pages=%d matched=%d eligible=%d legacy-candidate=%d excluded=[%s]",
			st.name, state, total, (total+99)/100, len(s.candidates[st.name]), eligible, legacy, strings.Join(parts, " "))
		if eligible > s.set.max {
			over = append(over, st.name)
		}
	}
	for name, err := range s.failed {
		// The error text is the API's status and body for a list call, or the
		// unstable-list counts; neither carries an object's fields.
		s.logf("sweep-list-failed %s: %v", name, err)
	}
	return over
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
// re-matches it, and re-runs the container's reference queries against the
// live appliance: anything that changed since collection keeps the object.
func (s *sweeper) deleteEligible() error {
	var errs []error
	for phase := 0; phase <= 4; phase++ {
		var jia map[int]string
		var jiaErr error
		jiaRead := false
		for i := range sweepTypes {
			st := &sweepTypes[i]
			if sweepPhase(st.kind) != phase {
				continue
			}
			for _, cand := range s.candidates[st.name] {
				if !s.eligible[sweepKey{st.name, cand.id}] {
					continue
				}
				if st.kind == kindJumpGroup && !jiaRead {
					jia, jiaErr = s.sharedJumpGroupReferences()
					jiaRead = true
				}
				ok := func(item *sweepRow) error {
					marker, matched := matchFixture(st, *item, s.groups)
					if !matched || marker != cand.marker {
						return errors.New("no-longer-matches")
					}
					if _, decision, _ := s.ageDecision(st, cand.id, marker); decision != "eligible" {
						return errors.New("no-longer-eligible")
					}
					if reason := s.blockedLive(cand); reason != "" {
						return errors.New(reason)
					}
					if st.kind == kindJumpGroup {
						if reason := jiaReason(jia, jiaErr, cand.id); reason != "" {
							return errors.New(reason)
						}
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
					s.deleted[sweepKey{st.name, cand.id}] = true
					s.logf("sweep %s %d %s", st.name, cand.id, outcome.result)
				}
			}
		}
	}
	return errors.Join(errs...)
}

// blockedLive is blockedByInventory against the appliance as it is now. In a
// reference check a 404 keeps the container: it may be a wrong path, and a
// wrong path must not remove a safety check. (Types the inventory found absent
// are not queried.)
func (s *sweeper) blockedLive(cand *candidate) string {
	type query struct {
		st     *sweepType
		filter map[string]string
	}
	var queries []query
	switch cand.st.kind {
	case kindAccountGroup:
		queries = append(queries, query{typeByName("vault_account"),
			map[string]string{"account_group_id": strconv.Itoa(cand.id), "include_personal": "true"}})
	case kindPolicy:
		code := cand.row.str("code_name")
		for _, typ := range []string{"vault_account", "vault_account_group"} {
			st := typeByName(typ)
			rows, err := api.ListAllItemsEndpoint(s.c, st.endpoint, st.query, rowKey)
			if err != nil {
				return "reference-check-failed " + typ
			}
			for _, r := range rows {
				if r.str("account_policy") == code {
					return fmt.Sprintf("used by %s %d", typ, r.id())
				}
			}
		}
	case kindJumpGroup, kindJumpoint:
		for _, st := range typesOfKind(kindJumpItem, kindInstaller, kindJumpClient) {
			field := st.groupField
			if cand.st.kind == kindJumpoint {
				field = st.jumpointField
			}
			if field == "" || s.absent[st.name] {
				continue
			}
			queries = append(queries, query{st, map[string]string{field: strconv.Itoa(cand.id)}})
		}
	}
	for _, q := range queries {
		rows, err := api.ListAllItemsEndpoint(s.c, q.st.endpoint, q.filter, rowKey)
		if err != nil {
			return "reference-check-failed " + q.st.name
		}
		if len(rows) > 0 {
			if cand.st.kind == kindAccountGroup {
				return fmt.Sprintf("contains %s %d", q.st.name, rows[0].id())
			}
			return fmt.Sprintf("used by %s %d", q.st.name, rows[0].id())
		}
	}
	return ""
}

// gpSnapshot records, read-only, every member of the group policy the fixtures
// write memberships on (data.sra_group_policy_list.gp.items[0]): ids and
// whether each is a fixture object, never a name.
func (s *sweeper) gpSnapshot() error {
	gps, err := api.ListAllItemsEndpoint(s.c, "group-policy", nil, rowKey)
	if err != nil {
		return fmt.Errorf("listing group policies: %w", err)
	}
	if len(gps) == 0 {
		return errors.New("no group policies")
	}
	gp := gps[0].id()
	for _, rel := range []struct{ path, field, typ string }{
		{"jump-group", "jump_group_id", "jump_group"},
		{"jumpoint", "jumpoint_id", "jumpoint"},
		{"vault-account-group", "account_group_id", "vault_account_group"},
		{"vault-account", "account_id", "vault_account"},
	} {
		key := func(r *sweepRow) string { return fmt.Sprint((*r)[rel.field]) }
		members, err := api.ListAllItemsEndpoint(s.c, fmt.Sprintf("group-policy/%d/%s", gp, rel.path), nil, key)
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
func deleteIfStill[I api.APIResource](c *api.APIClient, endpoint string, id int, ok func(*I) error) (deleteOutcome, error) {
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
	switch err := api.DeleteItemEndpoint[I](c, path); {
	case err == nil:
		return deleteOutcome{result: "deleted"}, nil
	case api.IsNotFound(err):
		return deleteOutcome{result: "already-gone"}, nil
	default:
		return deleteOutcome{}, err
	}
}
