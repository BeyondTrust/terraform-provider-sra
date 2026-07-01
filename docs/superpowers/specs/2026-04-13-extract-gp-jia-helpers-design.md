# Extract GP Membership & JIA Helpers — Design Spec

## Goal

Replace ~1200 lines of copy-pasted GroupPolicy membership and JumpItemAssociation CRUD code across 6 resource files with shared generic helpers. Also genericize the 4 structurally identical `DiffGP*Lists` functions into one.

## Motivation

The GP membership Create/Read/Update flow is repeated in `vault_ssh_account.go`, `vault_user_pass_account.go`, `vault_token_account.go`, `vault_account_group.go`, `jump_group.go`, and `jumpoint.go`. Each copy is ~200 lines and structurally identical — only the GP type, entity ID field, diff function, and mutex vary. A bug fix in one copy must be replicated to the other 5.

The JIA (JumpItemAssociation) CRUD is repeated identically in the 3 vault account files (SSH, UserPass, Token).

The `set_helpers.go` file has 4 functions (`DiffGPAccountLists`, `DiffGPAccountGroupLists`, `DiffGPJumpItemLists`, `DiffGPJumpointLists`) that are structurally identical — only the struct types and pointer fields differ.

## Scope

### In scope

- Generic diff function replacing 4 `DiffGP*Lists` functions in `api/set_helpers.go`
- GP membership Create/Read/Update helpers in `bt/rs/`
- JIA Create/Read/Update helpers for vault accounts in `bt/rs/`
- Fix `strconv.Atoi` errors in resource files touched (these were deferred from the prior refactor)
- Update `api/set_helpers_test.go` for the new generic function

### Out of scope

- Schema definitions (unchanged)
- ModifyPlan implementations (unchanged)
- Account-group JIA behavior (unique update-only pattern, only 1 file — not worth abstracting)
- FilterRules handling
- Any new features

---

## Design

### Part 1: Generic Diff Function

**File:** `api/set_helpers.go`

Replace all four `DiffGP*Lists` functions with one generic function:

```go
func DiffGPLists[T any, K comparable](
    planList []T,
    stateList []T,
    toKey func(T) K,
    fromKey func(K) T,
) (toAdd, toRemove, noChange mapset.Set[T])
```

The `toKey` function extracts a comparable value-type key from a pointer-containing struct. The `fromKey` function reconstructs the struct from a key. This eliminates the `noPointerGP*` intermediate types.

Each call site provides its own `toKey`/`fromKey` closures. For example, `GroupPolicyVaultAccount`:

```go
toKey := func(g api.GroupPolicyVaultAccount) noPointerGPAccount {
    return noPointerGPAccount{GroupPolicyID: *g.GroupPolicyID, Role: g.Role}
}
fromKey := func(k noPointerGPAccount) api.GroupPolicyVaultAccount {
    return api.GroupPolicyVaultAccount{GroupPolicyID: &k.GroupPolicyID, Role: k.Role}
}
```

The `noPointerGP*` key types stay in `set_helpers.go` since they're needed for the comparable constraint.

**Test:** `api/set_helpers_test.go` updated to call the new generic function with the same inputs and assert the same outputs.

### Part 2: GP Membership CRUD Helpers

**File:** `bt/rs/gp_membership.go` (new file)

Three exported functions that encapsulate the GP membership flow:

#### CreateGPMemberships

```go
func CreateGPMemberships[T api.APIResource](
    ctx context.Context,
    client *api.APIClient,
    req resource.CreateRequest,
    resp *resource.CreateResponse,
    entityID int,
    setEntityID func(*T, int),
    mu *sync.Mutex,
)
```

Flow:
1. Get `group_policy_memberships` from plan
2. If null, return
3. Lock mutex
4. For each membership: set entity ID, call `api.CreateItem`, collect result + GroupPolicyID
5. Provision each unique GroupPolicyID
6. Set results in state

#### ReadGPMemberships

```go
func ReadGPMemberships[T api.APIResource](
    ctx context.Context,
    client *api.APIClient,
    req resource.ReadRequest,
    resp *resource.ReadResponse,
    entityID int,
    getEndpoint func(T, int) string,
    setGPID func(*T, string),
)
```

Flow:
1. Get `group_policy_memberships` from state
2. If null, return
3. For each membership: build endpoint, GET, update with response
4. Set results in state

#### UpdateGPMemberships

```go
func UpdateGPMemberships[T api.APIResource, K comparable](
    ctx context.Context,
    client *api.APIClient,
    req resource.UpdateRequest,
    resp *resource.UpdateResponse,
    entityID int,
    setEntityID func(*T, int),
    getDeleteEndpoint func(T, int) string,
    diffFunc func([]T, []T) (mapset.Set[T], mapset.Set[T], mapset.Set[T]),
    mu *sync.Mutex,
)
```

Flow:
1. Get plan and state GP lists
2. If both null, return
3. Diff plan vs state
4. Lock mutex
5. For each toRemove: delete + collect GroupPolicyID
6. For each toAdd: create + collect GroupPolicyID
7. Combine noChange + added results
8. Provision each unique GroupPolicyID
9. Set results in state

The `setEntityID` callback handles the varying ID field (`m.AccountID = &id` vs `m.JumpGroupID = &id` etc.).

The `getDeleteEndpoint`/`getEndpoint` callbacks handle building the endpoint with the entity ID (e.g., `fmt.Sprintf("%s/%d", m.Endpoint(), entityID)`).

The `diffFunc` is the caller's partially applied `DiffGPLists[T, K]` call with its specific `toKey`/`fromKey`.

### Part 3: JIA Helpers for Vault Accounts

**File:** `bt/rs/jia_account.go` (new file)

Three functions for the vault account JIA pattern (used by SSH, UserPass, Token — identical in all 3):

#### CreateAccountJIA

```go
func CreateAccountJIA(
    ctx context.Context,
    client *api.APIClient,
    req resource.CreateRequest,
    resp *resource.CreateResponse,
    accountID int,
)
```

Flow:
1. Get `jump_item_association` from plan
2. If null, return; if unknown, set null in state and return
3. Deserialize to `api.AccountJumpItemAssociation`
4. Set ID, call `api.CreateItem`
5. Set result in state

#### ReadAccountJIA

```go
func ReadAccountJIA(
    ctx context.Context,
    client *api.APIClient,
    req resource.ReadRequest,
    resp *resource.ReadResponse,
    accountID int,
)
```

Flow:
1. Get `jump_item_association` from state
2. Build endpoint with account ID
3. GET from API
4. Handle null/empty cases
5. Set result in state

#### UpdateAccountJIA

```go
func UpdateAccountJIA(
    ctx context.Context,
    client *api.APIClient,
    req resource.UpdateRequest,
    resp *resource.UpdateResponse,
    accountID int,
)
```

Flow:
1. Get plan and state for `jump_item_association`
2. Compare plan vs state (both gone → return, state gone → create, plan gone → delete, else → update)
3. Set result in state

### Part 4: Simplify Resource Files

Each resource file's Create/Read/Update methods shrink dramatically:

**Before (vault_ssh_account.go Create, ~140 lines):**
```go
func (r *vaultSSHAccountResource) Create(...) {
    r.apiResource.Create(ctx, req, resp)
    // ... 50 lines of JIA handling
    // ... 70 lines of GP handling
}
```

**After (~15 lines):**
```go
func (r *vaultSSHAccountResource) Create(...) {
    r.apiResource.Create(ctx, req, resp)
    if resp.Diagnostics.HasError() { return }

    id := extractID(ctx, resp)
    CreateAccountJIA(ctx, r.ApiClient, req, resp, id)
    if resp.Diagnostics.HasError() { return }

    CreateGPMemberships(ctx, r.ApiClient, req, resp, id, setVaultAccountID, &accountMembershipMutex)
}
```

### strconv.Atoi cleanup

The resource files currently have unchecked `strconv.Atoi` calls for extracting the entity ID after Create/Read/Update. Since we're touching all these files anyway, add a shared helper:

```go
func extractID(resp *resource.CreateResponse) (int, bool) {
    var tfId types.String
    resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
    id, err := strconv.Atoi(tfId.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(...)
        return 0, false
    }
    return id, true
}
```

(With variants for Read/Update that read from different sources.)

---

## Files Changed Summary

| File | Action |
|------|--------|
| `api/set_helpers.go` | Rewrite: one generic `DiffGPLists` replaces 4 functions |
| `api/set_helpers_test.go` | Rewrite: test the generic function |
| `bt/rs/gp_membership.go` | **New:** CreateGPMemberships, ReadGPMemberships, UpdateGPMemberships |
| `bt/rs/jia_account.go` | **New:** CreateAccountJIA, ReadAccountJIA, UpdateAccountJIA |
| `bt/rs/vault_ssh_account.go` | Simplify: replace inline JIA/GP code with helper calls |
| `bt/rs/vault_user_pass_account.go` | Simplify: replace inline JIA/GP code with helper calls |
| `bt/rs/vault_token_account.go` | Simplify: replace inline JIA/GP code with helper calls |
| `bt/rs/vault_account_group.go` | Simplify: replace inline GP code with helper calls (JIA stays inline — unique pattern) |
| `bt/rs/jump_group.go` | Simplify: replace inline GP code with helper calls |
| `bt/rs/jumpoint.go` | Simplify: replace inline GP code with helper calls |

## Testing Strategy

1. All existing unit tests must pass after each change
2. The generic diff function gets the same test cases as the 4 functions it replaces
3. The GP/JIA helpers are integration-level code — they're tested by the existing E2E terratest suite
4. Run `go build ./...` and `go vet ./...` after every change
5. Final verification: run E2E tests against a real instance if available

## Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Generic diff function | Low | Same test cases, mechanical transformation |
| GP helpers | Medium | Same flow, just extracted; E2E tests cover |
| JIA helpers | Medium | Identical code in 3 files; E2E tests cover |
| Resource file simplification | Medium | Each file change is mechanical replacement of inline code with helper call |
