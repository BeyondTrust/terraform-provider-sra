# API Layer Safety & Correctness Refactor — Design Spec

## Goal

Fix safety, correctness, and maintainability issues in the `api/` package and its immediate consumers (`bt/provider.go`, `bt/rs/api_resource.go`, `bt/ds/api_data_source.go`). No changes to individual resource or data source implementations.

## Motivation

The code review identified several categories of issues:

- **Unsafe pointer usage** in `model_transforms.go` — fragile, bypasses Go's type system
- **Panics in production code** in `model_transforms.go` — crashes the Terraform provider
- **Global mutable product state** in `product.go` — not concurrency-safe, prevents multi-config
- **Nil-client-after-error** in `provider.go` — uses client before checking error
- **Silent `strconv.Atoi` errors** — corrupted state silently becomes resource ID 0
- **Missing `Content-Type` header** — POST/PATCH send JSON without declaring it
- **`testing.T` in production code** — production binary links the test framework
- **Dead code** — ~140 lines of commented-out FilterRules code
- **Stale `x/exp/slices` import** — stdlib `slices` available since Go 1.21
- **Error message typos** and naming convention violations in `provider.go`

## Scope

### In scope

All changes in the `api/` package, plus minimal touch points in:
- `bt/provider.go` (client configuration, product detection)
- `bt/rs/api_resource.go` (generic CRUD, passes product to transforms)
- `bt/ds/api_data_source.go` (generic Read, passes product to transforms)

### Out of scope (follow-up work)

- Resource-level JIA/GP membership code duplication
- `set_helpers.go` genericization
- FilterRules conversion rewrite (only deleting the dead commented code here)
- Any new features

---

## Design

### 1. Product State Moves Onto `APIClient`

**Current:** `api/product.go` has a package-level `var product = ProductPRA` with `SetProductIsRS()`, `IsRS()`, `IsPRA()`, `ProductName()` as free functions. Not concurrency-safe. Prevents multiple provider configurations.

**New:** `APIClient` gains a `Product string` field. Methods `IsRS()`, `IsPRA()`, `ProductName()` move to `APIClient`. The `SetProductIsRS()` function and package-level `var product` are deleted.

**Impact on transforms:** `CopyTFtoAPI` and `CopyAPItoTF` currently read the global via `product` variable directly (not even through the accessor). They'll take a `product string` parameter instead:

```go
func CopyTFtoAPI(ctx context.Context, tfObj reflect.Value, apiObj reflect.Value, product string)
func CopyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value, apiType reflect.Type, product string)
```

**Impact on `IsProductAllowed`:** Currently takes `(ctx, interface{})`. Will take `(ctx, interface{}, product string)`.

**Impact on callers:** `bt/rs/api_resource.go` and `bt/ds/api_data_source.go` already have `r.ApiClient` / `d.apiClient`. They pass `r.ApiClient.Product` to the transform functions.

**Impact on `bt/provider.go`:** Instead of calling `api.SetProductIsRS(mechs.IsRS())`, sets `c.Product = api.ProductRS` or `c.Product = api.ProductPRA`.

**Impact on tests:** Test functions that call `SetProductIsRS()` will pass the product string directly to the transform functions instead.

### 2. Unsafe Pointers Replaced With `reflect.Value.Set()`

**Current:** `CopyAPItoTF` uses patterns like:
```go
*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(...)
```

**New:** Replace with the safe reflect API:
```go
tfObj.Field(i).Set(reflect.ValueOf(types.StringValue(...)))
```

This applies to all type cases in `CopyAPItoTF`: String, Int64, Bool, Set (null and valued), and ID.

The `CopyTFtoAPI` direction doesn't use unsafe pointers (it uses `field.SetString()`, `field.SetInt()`, etc.) so it doesn't need changes.

### 3. Panics Replaced With Returned Errors

**Current:** `CopyAPItoTF` panics on:
- Unhandled slice element types (line 741)
- Error converting Go set to TF object (line 748)
- Unknown encoded type in struct (line 753)

`CopyTFtoAPI` has a `default` branch that logs and continues (already safe — no panic).

**New:** `CopyAPItoTF` returns `error`:

```go
func CopyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value, apiType reflect.Type, product string) error
```

Callers check the error and add it to `resp.Diagnostics`. The three panic sites become `return fmt.Errorf(...)`.

`CopyTFtoAPI` signature stays the same (void return) since its default branch already handles unknown types gracefully.

### 4. `testing.T` Removed From `APIClient`

**Current:** `APIClient` has `t *testing.T` and imports `testing`. Production binary links the test framework.

**New:** Replace with a logger interface:

```go
type Logger interface {
    Logf(format string, args ...any)
}
```

`APIClient.t *testing.T` becomes `APIClient.testLogger Logger`. `SetTest(t *testing.T)` becomes `SetTestLogger(l Logger)`. `*testing.T` satisfies `Logger` so callers don't change.

The `import "testing"` is removed from `client.go`.

### 5. Fix `doRequest` Content-Type

**Current:** Sets `Accept: application/json` but not `Content-Type` on requests with bodies.

**New:** Add `Content-Type: application/json` when `req.Body != nil`:

```go
if req.Body != nil {
    req.Header.Set("Content-Type", "application/json")
}
```

### 6. Fix Provider Configuration Error Handling

**Current (`bt/provider.go:155-176`):**
```go
c, err := api.NewClient(host, &client_id, &client_secret)
c.SetLogContext(&ctx)     // uses c before checking err
if err != nil { ... }
mechs, err := api.Get[api.MechList](c)  // shadows first err
if err != nil { ... }
```

**New:**
```go
c, err := api.NewClient(host, &clientID, &clientSecret)
if err != nil {
    resp.Diagnostics.AddError(...)
    return
}
c.SetLogContext(&ctx)

mechs, mechErr := api.Get[api.MechList](c)
if mechErr != nil {
    resp.Diagnostics.AddError(...)
    return
}
c.Product = productFromMechs(mechs)
```

Key changes: error checked before using client, separate error variables, early return pattern.

### 7. Fix Silent `strconv.Atoi` Errors

**Current:** Throughout `api_resource.go`:
```go
id, _ := strconv.Atoi(tfId.ValueString())
```

**New:** Check the error and add to diagnostics:
```go
id, err := strconv.Atoi(tfId.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Invalid resource ID", ...)
    return
}
```

This applies to all `strconv.Atoi` calls in `bt/rs/api_resource.go` (Read, Update, Delete). The individual resource files (`vault_ssh_account.go`, `jump_group.go`, etc.) also have unchecked `Atoi` calls but those are out of scope for this refactor — they'll be addressed when we tackle the resource-level duplication.

### 8. Delete Dead Code

- Remove ~140 lines of commented-out FilterRules code in `model_transforms.go` (lines 260-404)
- Remove the `import "testing"` from `client.go` after the logger interface change

### 9. Replace `x/exp/slices` With Stdlib

**Current:** `model_transforms.go` imports `golang.org/x/exp/slices`.

**New:** Import `slices` from stdlib (available since Go 1.21, running Go 1.26).

After this change, run `go mod tidy` to remove the `x/exp` dependency if nothing else uses it.

### 10. Fix Provider Typos and Naming

- `provider.go:65`: Remove stray `q` character from markdown description
- `provider.go:129`: Change error title from "Missing BeyondTrust SRA API Username" to "Missing BeyondTrust SRA API Client ID"
- `provider.go:139`: Change error title from "Missing BeyondTrust SRA API Password" to "Missing BeyondTrust SRA API Client Secret"
- `provider.go:100-101`: Rename `client_id` → `clientID`, `client_secret` → `clientSecret` (Go naming convention)

---

## New Test Opportunities

The refactor enables several new tests that weren't possible or practical before:

### Product state on client (Section 1)
- **Test `APIClient.IsRS()` / `IsPRA()` / `ProductName()` methods** — verify the methods work on the client instance. The existing `product_test.go` tests the global functions; new tests verify the instance methods.
- **Test concurrent product state safety** — two clients with different products can coexist without interference (impossible to test with the global).

### CopyAPItoTF returns error (Section 3)
- **Test error return on unhandled type** — create a test model with an unhandled field type (e.g., `float64`) and verify `CopyAPItoTF` returns an error instead of panicking. Currently untestable because it panics.

### Provider error handling (Section 6)
- **Test that `NewClient` error is properly reported** — verify the provider returns a diagnostic error when credentials are bad, not a nil-pointer panic.

### Atoi error handling (Section 7)
- **Test that corrupted ID produces diagnostic error** — verify a non-numeric ID in state produces a clear error message, not silent operation on resource 0.

### Content-Type header (Section 5)
- **Test that POST/PATCH requests include Content-Type** — extend existing `crud_test.go` to verify the header.

---

## Testing Strategy

1. **All existing tests must pass** after each change
2. The model transform tests added in the pre-refactor coverage work are the safety net for sections 1-3
3. Transform test helpers will be updated to pass `product` as a parameter instead of calling `SetProductIsRS()`
4. New tests listed above will be added as part of the implementation tasks
5. Run `go build ./...` and `go vet ./...` after changes to verify compilation

## Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Product on client | Medium — threading through transforms | Compiler catches missing params; existing tests cover behavior |
| Unsafe pointer removal | Low — safe reflect API is equivalent | Pre-refactor tests lock down behavior for every type |
| Panic removal | Low — adding error returns | Callers already handle errors for other operations |
| Provider error ordering | Low — straightforward reorder | Manual verification against a real instance |
| Content-Type header | Low — additive | Unlikely to break anything; could fix subtle issues |
| `testing.T` removal | Low — interface swap | `*testing.T` satisfies the interface |
| Dead code deletion | None — commented out code | Git has history |
| Atoi error handling | Low — adding checks | Only affects corrupted state paths |
