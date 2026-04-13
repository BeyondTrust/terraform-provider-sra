# Extract GP Membership & JIA Helpers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ~1200 lines of copy-pasted GP membership and JIA CRUD code across 6 resource files with shared generic helpers, and genericize the 4 identical `DiffGP*Lists` functions into one.

**Architecture:** A single generic `DiffGPLists` function replaces 4 type-specific diff functions. Three GP membership helper functions (`CreateGPMemberships`, `ReadGPMemberships`, `UpdateGPMemberships`) in a new file handle the shared flow with type parameters and callbacks for the varying pieces. Three JIA helper functions handle the vault account JIA pattern shared by SSH/UserPass/Token.

**Tech Stack:** Go 1.26 generics, `github.com/deckarep/golang-set/v2`, terraform-plugin-framework

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `api/set_helpers.go` | Rewrite | One generic `DiffGPLists` replaces 4 functions; keep key types |
| `api/set_helpers_test.go` | Rewrite | Test generic function with all 4 GP types |
| `bt/rs/gp_membership.go` | **Create** | `CreateGPMemberships`, `ReadGPMemberships`, `UpdateGPMemberships` |
| `bt/rs/jia_account.go` | **Create** | `CreateAccountJIA`, `ReadAccountJIA`, `UpdateAccountJIA` |
| `bt/rs/vault_ssh_account.go` | Simplify | Replace inline JIA/GP with helper calls |
| `bt/rs/vault_user_pass_account.go` | Simplify | Replace inline JIA/GP with helper calls |
| `bt/rs/vault_token_account.go` | Simplify | Replace inline JIA/GP with helper calls |
| `bt/rs/vault_account_group.go` | Simplify | Replace inline GP with helper calls (JIA stays inline) |
| `bt/rs/jump_group.go` | Simplify | Replace inline GP with helper calls |
| `bt/rs/jumpoint.go` | Simplify | Replace inline GP with helper calls |

---

### Task 1: Generic DiffGPLists Function

Replace the four `DiffGP*Lists` functions with one generic function. Update tests.

**Files:**
- Modify: `api/set_helpers.go`
- Modify: `api/set_helpers_test.go`

- [ ] **Step 1: Rewrite `api/set_helpers.go`**

Replace the entire file with:

```go
package api

import mapset "github.com/deckarep/golang-set/v2"

// DiffGPLists computes the three-way diff (toAdd, toRemove, noChange) between
// a plan list and a state list of group policy memberships. The toKey and
// fromKey functions convert between the pointer-containing API types and a
// comparable value type suitable for set operations.
func DiffGPLists[T any, K comparable](
	planList []T,
	stateList []T,
	toKey func(T) K,
	fromKey func(K) T,
) (toAdd, toRemove, noChange mapset.Set[T]) {
	planKeys := make([]K, len(planList))
	for i, item := range planList {
		planKeys[i] = toKey(item)
	}
	stateKeys := make([]K, len(stateList))
	for i, item := range stateList {
		stateKeys[i] = toKey(item)
	}

	planSet := mapset.NewSet(planKeys...)
	stateSet := mapset.NewSet(stateKeys...)

	addKeys := planSet.Difference(stateSet)
	removeKeys := stateSet.Difference(planSet)
	unchangedKeys := planSet.Intersect(stateSet)

	toAdd = mapset.NewSet[T]()
	for k := range addKeys.Iter() {
		toAdd.Add(fromKey(k))
	}
	toRemove = mapset.NewSet[T]()
	for k := range removeKeys.Iter() {
		toRemove.Add(fromKey(k))
	}
	noChange = mapset.NewSet[T]()
	for k := range unchangedKeys.Iter() {
		noChange.Add(fromKey(k))
	}

	return toAdd, toRemove, noChange
}

// Key types for set operations (comparable, no pointers)

type gpAccountKey struct {
	GroupPolicyID string
	Role          string
}

type gpAccountGroupKey struct {
	GroupPolicyID string
	Role          string
}

type gpJumpGroupKey struct {
	GroupPolicyID  string
	JumpItemRoleID int
	JumpPolicyID   int
}

type gpJumpointKey struct {
	GroupPolicyID string
}

// Convenience wrappers that provide the toKey/fromKey for each GP type.
// These preserve the existing public API so callers don't need to change yet.

func DiffGPAccountLists(planList []GroupPolicyVaultAccount, stateList []GroupPolicyVaultAccount) (mapset.Set[GroupPolicyVaultAccount], mapset.Set[GroupPolicyVaultAccount], mapset.Set[GroupPolicyVaultAccount]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyVaultAccount) gpAccountKey {
			return gpAccountKey{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
		},
		func(k gpAccountKey) GroupPolicyVaultAccount {
			id := k.GroupPolicyID
			return GroupPolicyVaultAccount{GroupPolicyID: &id, Role: k.Role}
		},
	)
}

func DiffGPAccountGroupLists(planList []GroupPolicyVaultAccountGroup, stateList []GroupPolicyVaultAccountGroup) (mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup], mapset.Set[GroupPolicyVaultAccountGroup]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyVaultAccountGroup) gpAccountGroupKey {
			return gpAccountGroupKey{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
		},
		func(k gpAccountGroupKey) GroupPolicyVaultAccountGroup {
			id := k.GroupPolicyID
			return GroupPolicyVaultAccountGroup{GroupPolicyID: &id, Role: k.Role}
		},
	)
}

func DiffGPJumpItemLists(planList []GroupPolicyJumpGroup, stateList []GroupPolicyJumpGroup) (mapset.Set[GroupPolicyJumpGroup], mapset.Set[GroupPolicyJumpGroup], mapset.Set[GroupPolicyJumpGroup]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyJumpGroup) gpJumpGroupKey {
			return gpJumpGroupKey{GroupPolicyID: *g.GroupPolicyID, JumpItemRoleID: g.JumpItemRoleID, JumpPolicyID: *g.JumpPolicyID}
		},
		func(k gpJumpGroupKey) GroupPolicyJumpGroup {
			gpID := k.GroupPolicyID
			jpID := k.JumpPolicyID
			return GroupPolicyJumpGroup{GroupPolicyID: &gpID, JumpItemRoleID: k.JumpItemRoleID, JumpPolicyID: &jpID}
		},
	)
}

func DiffGPJumpointLists(planList []GroupPolicyJumpoint, stateList []GroupPolicyJumpoint) (mapset.Set[GroupPolicyJumpoint], mapset.Set[GroupPolicyJumpoint], mapset.Set[GroupPolicyJumpoint]) {
	return DiffGPLists(planList, stateList,
		func(g GroupPolicyJumpoint) gpJumpointKey {
			return gpJumpointKey{GroupPolicyID: *g.GroupPolicyID}
		},
		func(k gpJumpointKey) GroupPolicyJumpoint {
			id := k.GroupPolicyID
			return GroupPolicyJumpoint{GroupPolicyID: &id}
		},
	)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./api/... -run TestDiffGP -v -count=1`
Expected: All 4 existing tests PASS — the convenience wrappers preserve the old API.

- [ ] **Step 3: Add a direct test for the generic function**

Add to `api/set_helpers_test.go`:

```go
func TestDiffGPLists_Generic(t *testing.T) {
	t.Parallel()

	type item struct {
		Key   string
		Value int
	}
	type itemKey struct {
		Key string
	}

	toKey := func(i item) itemKey { return itemKey{Key: i.Key} }
	fromKey := func(k itemKey) item { return item{Key: k.Key} }

	plan := []item{{Key: "add", Value: 1}, {Key: "keep", Value: 2}}
	state := []item{{Key: "remove", Value: 3}, {Key: "keep", Value: 4}}

	toAdd, toRemove, noChange := DiffGPLists(plan, state, toKey, fromKey)

	assert.Equal(t, 1, toAdd.Cardinality())
	assert.Equal(t, 1, toRemove.Cardinality())
	assert.Equal(t, 1, noChange.Cardinality())

	assert.Equal(t, "add", toAdd.ToSlice()[0].Key)
	assert.Equal(t, "remove", toRemove.ToSlice()[0].Key)
	assert.Equal(t, "keep", noChange.ToSlice()[0].Key)
}
```

- [ ] **Step 4: Run all tests**

Run: `go test ./api/... -v -count=1`
Expected: All PASS

- [ ] **Step 5: Commit**

```
git add api/set_helpers.go api/set_helpers_test.go
git commit -m "refactor: genericize DiffGPLists, replace 4 type-specific functions"
```

---

### Task 2: GP Membership Helpers

Create `bt/rs/gp_membership.go` with the three shared GP membership CRUD functions.

**Files:**
- Create: `bt/rs/gp_membership.go`

- [ ] **Step 1: Create the helper file**

Create `bt/rs/gp_membership.go`. The implementer should:

1. Read the current GP membership code from `bt/rs/jumpoint.go` (Create, Read, Update methods — the simplest GP-only resource) to understand the exact flow.
2. Read the same from `bt/rs/vault_ssh_account.go` (Create lines 201-276, Read lines 349-402, Update lines 497-622) to confirm it's the same pattern with different types.
3. Extract three generic functions that take callbacks for the varying parts.

The GP membership flow uses these varying pieces across the 6 resource files:

| Resource | GP Type | Entity ID Setter | Diff Function |
|----------|---------|------------------|---------------|
| vault_ssh_account | `GroupPolicyVaultAccount` | `m.AccountID = &id` | `DiffGPAccountLists` |
| vault_user_pass_account | `GroupPolicyVaultAccount` | `m.AccountID = &id` | `DiffGPAccountLists` |
| vault_token_account | `GroupPolicyVaultAccount` | `m.AccountID = &id` | `DiffGPAccountLists` |
| vault_account_group | `GroupPolicyVaultAccountGroup` | `m.AccountGroupID = &id` | `DiffGPAccountGroupLists` |
| jump_group | `GroupPolicyJumpGroup` | `m.JumpGroupID = &id` | `DiffGPJumpItemLists` |
| jumpoint | `GroupPolicyJumpoint` | `m.JumpointID = &id` | `DiffGPJumpointLists` |

The helper functions should have these signatures:

```go
// GPMembership constrains the GP types that can be used with the helpers.
type GPMembership interface {
	api.APIResource
	GetGroupPolicyID() *string
	SetGroupPolicyID(id *string)
}

func CreateGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
	setEntityID func(*T, int),
	mu *sync.Mutex,
)

func ReadGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
)

func UpdateGPMemberships[T GPMembership](
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	entityID int,
	setEntityID func(*T, int),
	diffFunc func([]T, []T) (mapset.Set[T], mapset.Set[T], mapset.Set[T]),
	mu *sync.Mutex,
)
```

**IMPORTANT implementation notes:**

- The `GPMembership` interface needs `GetGroupPolicyID()` and `SetGroupPolicyID()` methods. These don't exist on the API types yet — the implementer will need to add them to `api/models.go` for all 4 GP types (`GroupPolicyVaultAccount`, `GroupPolicyVaultAccountGroup`, `GroupPolicyJumpGroup`, `GroupPolicyJumpoint`). Each type already has a `GroupPolicyID *string` field.

- All GP types also already implement `api.APIResource` (they have an `Endpoint()` method).

- The `ReadGPMemberships` helper reads from state, calls `GetItemEndpoint` for each membership by building the endpoint as `fmt.Sprintf("%s/%d", m.Endpoint(), entityID)`, sets the GroupPolicyID from the original, and writes back to state. On API error, it logs and skips (doesn't fail) — this matches the current behavior.

- The provision step at the end of Create and Update is always the same: for each unique GroupPolicyID that was changed, create a `GroupPolicyProvision` and call `api.CreateItem`.

- The implementer should look at the exact current code in the resource files to ensure the helper matches the existing behavior precisely. The Diagnostics should be checked after each state mutation and returned early if errors occurred.

- [ ] **Step 2: Add GetGroupPolicyID/SetGroupPolicyID to API types**

In `api/models.go`, add these methods for each GP type:

For `GroupPolicyVaultAccount`:
```go
func (g *GroupPolicyVaultAccount) GetGroupPolicyID() *string { return g.GroupPolicyID }
func (g *GroupPolicyVaultAccount) SetGroupPolicyID(id *string) { g.GroupPolicyID = id }
```

Same pattern for `GroupPolicyVaultAccountGroup`, `GroupPolicyJumpGroup`, `GroupPolicyJumpoint`.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: Compiles (new file, not yet called)

- [ ] **Step 4: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 5: Commit**

```
git add bt/rs/gp_membership.go api/models.go
git commit -m "feat: add generic GP membership CRUD helpers"
```

---

### Task 3: JIA Account Helpers

Create `bt/rs/jia_account.go` with the three shared JIA CRUD functions for vault accounts (SSH, UserPass, Token).

**Files:**
- Create: `bt/rs/jia_account.go`

- [ ] **Step 1: Create the helper file**

The implementer should:

1. Read `bt/rs/vault_ssh_account.go` Create (lines 146-199 — the `updateJIA` closure), Read (lines 289-345 — the `readJIA` closure), and Update (lines 415-493 — the `updateJIA` closure).
2. Extract three functions: `CreateAccountJIA`, `ReadAccountJIA`, `UpdateAccountJIA`.

These handle the `api.AccountJumpItemAssociation` type. The three vault account files (SSH, UserPass, Token) use identical JIA code. The `vault_account_group.go` file uses a different type (`AccountGroupJumpItemAssociation`) with different behavior (update-only, never delete) — that stays inline.

Signatures:

```go
func CreateAccountJIA(
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	state *tfsdk.State,
	diags *diag.Diagnostics,
	accountID int,
)

func ReadAccountJIA(
	ctx context.Context,
	client *api.APIClient,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	accountID int,
)

func UpdateAccountJIA(
	ctx context.Context,
	client *api.APIClient,
	plan tfsdk.Plan,
	reqState tfsdk.State,
	respState *tfsdk.State,
	diags *diag.Diagnostics,
	accountID int,
)
```

The implementer must reproduce the exact current behavior from the vault_ssh_account.go closures, including:
- Create: null → return, unknown → set null in state and return, otherwise deserialize + CreateItem + set in state
- Read: get from state, build endpoint with ID, GetItemEndpoint, handle null/empty cases
- Update: compare plan vs state for null/unknown, decide CREATE/DELETE/UPDATE, set result in state

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Compiles

- [ ] **Step 3: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 4: Commit**

```
git add bt/rs/jia_account.go
git commit -m "feat: add JIA account CRUD helpers"
```

---

### Task 4: Simplify vault_ssh_account.go

Replace the inline JIA and GP code with helper calls.

**Files:**
- Modify: `bt/rs/vault_ssh_account.go`

- [ ] **Step 1: Read current file and rewrite Create/Read/Update**

The implementer should:

1. Read `bt/rs/vault_ssh_account.go` in full
2. Replace the Create method body (after `r.apiResource.Create` and ID extraction) with calls to `CreateAccountJIA` and `CreateGPMemberships`
3. Replace the Read method body (after `r.apiResource.Read` and ID extraction) with calls to `ReadAccountJIA` and `ReadGPMemberships`
4. Replace the Update method body (after `r.apiResource.Update` and ID extraction) with calls to `UpdateAccountJIA` and `UpdateGPMemberships`

The Create method should look approximately like:
```go
func (r *vaultSSHAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", "Could not parse resource ID: "+err.Error())
		return
	}

	CreateAccountJIA(ctx, r.ApiClient, req.Plan, &resp.State, &resp.Diagnostics, id)
	if resp.Diagnostics.HasError() {
		return
	}

	CreateGPMemberships[api.GroupPolicyVaultAccount](ctx, r.ApiClient, req.Plan, &resp.State, &resp.Diagnostics, id,
		func(m *api.GroupPolicyVaultAccount, entityID int) { m.AccountID = &entityID },
		&accountMembershipMutex,
	)
}
```

Same pattern for Read and Update, using the corresponding helper functions and passing the appropriate diff function for Update:
```go
UpdateGPMemberships[api.GroupPolicyVaultAccount](ctx, r.ApiClient, req.Plan, req.State, &resp.State, &resp.Diagnostics, id,
	func(m *api.GroupPolicyVaultAccount, entityID int) { m.AccountID = &entityID },
	api.DiffGPAccountLists,
	&accountMembershipMutex,
)
```

Also fix the unchecked `strconv.Atoi` calls while touching this file.

Remove unused imports (`json`, `mapset`, `basetypes`, etc.) that were only needed by the inline code.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Compiles

- [ ] **Step 3: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 4: Commit**

```
git add bt/rs/vault_ssh_account.go
git commit -m "refactor: simplify vault_ssh_account.go using JIA and GP helpers"
```

---

### Task 5: Simplify vault_user_pass_account.go and vault_token_account.go

These are identical to SSH account. Apply the same transformation.

**Files:**
- Modify: `bt/rs/vault_user_pass_account.go`
- Modify: `bt/rs/vault_token_account.go`

- [ ] **Step 1: Read and rewrite vault_user_pass_account.go**

Same pattern as Task 4. The implementer should read the file, replace Create/Read/Update with helper calls using `api.GroupPolicyVaultAccount`, `api.DiffGPAccountLists`, `func(m *api.GroupPolicyVaultAccount, entityID int) { m.AccountID = &entityID }`, and `&accountMembershipMutex`.

Fix unchecked `strconv.Atoi` calls. Remove unused imports.

- [ ] **Step 2: Read and rewrite vault_token_account.go**

Same transformation. Same types, same mutex, same diff function.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: Compiles

- [ ] **Step 4: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 5: Commit**

```
git add bt/rs/vault_user_pass_account.go bt/rs/vault_token_account.go
git commit -m "refactor: simplify vault_user_pass_account and vault_token_account using helpers"
```

---

### Task 6: Simplify vault_account_group.go

Replace inline GP code with helper calls. JIA stays inline (unique update-only pattern).

**Files:**
- Modify: `bt/rs/vault_account_group.go`

- [ ] **Step 1: Read and rewrite**

The implementer should:

1. Read `bt/rs/vault_account_group.go` in full
2. Replace the GP portions of Create/Read/Update with helper calls using `api.GroupPolicyVaultAccountGroup`, `api.DiffGPAccountGroupLists`, `func(m *api.GroupPolicyVaultAccountGroup, entityID int) { m.AccountGroupID = &entityID }`, and `&agMembershipMutex`
3. Leave the JIA portions (AccountGroupJumpItemAssociation) inline — they have unique behavior (update-only, never delete, different endpoint structure)
4. Fix unchecked `strconv.Atoi` calls
5. Remove unused imports

- [ ] **Step 2: Verify build and tests**

Run: `go build ./... && go test ./api/... ./bt/... -v -count=1`
Expected: Compiles, all PASS

- [ ] **Step 3: Commit**

```
git add bt/rs/vault_account_group.go
git commit -m "refactor: simplify vault_account_group.go GP code using helpers"
```

---

### Task 7: Simplify jump_group.go and jumpoint.go

Replace inline GP code with helper calls. These have no JIA.

**Files:**
- Modify: `bt/rs/jump_group.go`
- Modify: `bt/rs/jumpoint.go`

- [ ] **Step 1: Rewrite jump_group.go**

Replace GP portions of Create/Read/Update with helper calls using:
- `api.GroupPolicyJumpGroup`
- `func(m *api.GroupPolicyJumpGroup, entityID int) { m.JumpGroupID = &entityID }`
- `api.DiffGPJumpItemLists`
- `&jgMembershipMutex`

Fix unchecked `strconv.Atoi` calls. Remove unused imports.

- [ ] **Step 2: Rewrite jumpoint.go**

Replace GP portions of Create/Read/Update with helper calls using:
- `api.GroupPolicyJumpoint`
- `func(m *api.GroupPolicyJumpoint, entityID int) { m.JumpointID = &entityID }`
- `api.DiffGPJumpointLists`
- `&jpMembershipMutex`

Fix unchecked `strconv.Atoi` calls. Remove unused imports.

- [ ] **Step 3: Verify build and tests**

Run: `go build ./... && go test ./api/... ./bt/... -v -count=1`
Expected: Compiles, all PASS

- [ ] **Step 4: Commit**

```
git add bt/rs/jump_group.go bt/rs/jumpoint.go
git commit -m "refactor: simplify jump_group and jumpoint GP code using helpers"
```

---

### Task 8: Final Verification

Run the complete build, vet, and test suite. Count lines removed.

**Files:** None (verification only)

- [ ] **Step 1: Full build and vet**

Run: `go build ./... && go vet ./...`
Expected: Clean

- [ ] **Step 2: Run all tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 3: Count the impact**

Run: `git diff --stat origin/refactor/api-layer-safety-and-correctness...HEAD`
Expected: Significant net line reduction across the 6 simplified resource files

- [ ] **Step 4: Verify no remaining copy-paste patterns**

Run: `grep -l "needsProvision := mapset.NewSet\[string\]" bt/rs/*.go`
Expected: Only `bt/rs/gp_membership.go` should contain this pattern (the helper). It should NOT appear in the 6 resource files anymore.

Run: `grep -l "api.GroupPolicyProvision" bt/rs/*.go`
Expected: Only `bt/rs/gp_membership.go`
