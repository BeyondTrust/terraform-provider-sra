# API Layer Safety & Correctness Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix unsafe pointers, panics, global product state, nil-client-after-error, missing Content-Type, silent Atoi errors, and dead code in the `api/` layer and its immediate consumers.

**Architecture:** Product state moves from a package global to `APIClient`. Transform functions take a `product` parameter. `CopyAPItoTF` returns `error` instead of panicking. Unsafe pointer casts replaced with `reflect.Value.Set()`. Logger interface replaces `*testing.T` on the client.

**Tech Stack:** Go 1.26, terraform-plugin-framework, testify/assert

---

## File Structure

| File | Action | Changes |
|------|--------|---------|
| `api/client.go` | Modify | Logger interface, remove `testing` import, add Content-Type header, add Product field + methods |
| `api/product.go` | Modify | Delete global state, keep `RestrictsProducts` interface and `IsProductAllowed` (takes product param) |
| `api/model_transforms.go` | Modify | Add product param to signatures, replace unsafe pointers with reflect.Set, return error from CopyAPItoTF, delete dead code, replace x/exp/slices |
| `bt/provider.go` | Modify | Fix error handling order, set client.Product, fix typos, fix naming |
| `bt/rs/api_resource.go` | Modify | Pass product to transforms/IsProductAllowed, check Atoi errors |
| `bt/ds/api_data_source.go` | Modify | Pass product to transforms/IsProductAllowed |
| `bt/ds/vault_ssh_account.go` | Modify | Pass product to CopyAPItoTF |
| `bt/rs/jump_group.go` | Modify | Replace `api.IsPRA()` with `r.ApiClient.IsPRA()` |
| `bt/rs/jump_client_installer.go` | Modify | Replace `api.IsPRA()`/`api.IsRS()` with client methods |
| `bt/rs/jumpoint.go` | Modify | Replace `api.IsRS()`/`api.IsPRA()` with client methods |
| `bt/rs/remote_rdp.go` | Modify | Replace `api.IsPRA()` with `r.ApiClient.IsPRA()` |
| `api/client_test.go` | Modify | Update SetTest → SetTestLogger |
| `api/crud_test.go` | Modify | Update SetTest → SetTestLogger, add Content-Type assertion |
| `api/model_transforms_test.go` | Modify | Pass product string to transforms instead of calling SetProductIsRS |
| `api/product_test.go` | Modify | Rewrite tests for client methods instead of globals |
| `test/setup.go` | Modify | Update SetTest → SetTestLogger |

---

### Task 1: Logger Interface and Content-Type Header

Replace `*testing.T` on `APIClient` with a logger interface so the production binary doesn't link the test framework. Also add the missing `Content-Type: application/json` header for requests with bodies.

**Files:**
- Modify: `api/client.go`
- Modify: `api/client_test.go`
- Modify: `api/crud_test.go`
- Modify: `test/setup.go`

- [ ] **Step 1: Define Logger interface and update APIClient in `api/client.go`**

Replace the `testing` import and `t *testing.T` field. The new code for the top of the file:

```go
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2/clientcredentials"
)

// Logger is satisfied by *testing.T and any other type with a Logf method.
type Logger interface {
	Logf(format string, args ...any)
}

type APIClient struct {
	RootURL    string
	BaseURL    string
	HTTPClient *http.Client
	testLogger Logger
	logCtx     *context.Context
	mu         sync.Mutex
}
```

Rename `SetTest` to `SetTestLogger` and update references to `c.t` → `c.testLogger`:

```go
func (c *APIClient) SetTestLogger(l Logger) {
	if l == nil {
		return
	}
	if c == nil {
		l.Logf("Set testing context for APIClient (nil receiver)")
		return
	}
	c.testLogger = l
	l.Logf("Set testing context for APIClient")
}
```

Update `SetLogContext` — change `c.t` → `c.testLogger`:
```go
func (c *APIClient) SetLogContext(ctx *context.Context) {
	if ctx == nil {
		return
	}
	if c == nil {
		tflog.Debug(*ctx, "Set logging context for APIClient (nil receiver)")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logCtx = ctx
	tflog.Debug(*c.logCtx, "Set logging context for APIClient")
}
```

Update `LogString` — change `c.t` → `c.testLogger`:
```go
func (c *APIClient) LogString(format string, args ...any) {
	if c.testLogger != nil {
		c.testLogger.Logf(format, args...)
	}
	if c.logCtx != nil {
		c.mu.Lock()
		tflog.Debug(*c.logCtx, fmt.Sprintf(format, args...))
		c.mu.Unlock()
	}
}
```

Update `doRequest` — change `c.t` → `c.testLogger` in the debug condition, AND add Content-Type header:
```go
func (c *APIClient) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "SRA-Terraform-Plugin")
	req.Header.Set("Accept", "application/json")
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.testLogger != nil || c.logCtx != nil {
		// ... rest of debug logging unchanged, just s/c.t/c.testLogger/
```

- [ ] **Step 2: Update test files to use SetTestLogger**

In `api/client_test.go`, `api/crud_test.go`: replace all `c.SetTest(t)` with `c.SetTestLogger(t)`.

In `test/setup.go`: replace `client.SetTest(t)` with `client.SetTestLogger(t)`.

- [ ] **Step 3: Add Content-Type assertion to an existing CRUD test**

In `api/crud_test.go`, in the `TestCreateItem` handler, add this assertion inside the handler where it checks `r.Method == http.MethodPost`:

```go
assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
```

- [ ] **Step 4: Run tests**

Run: `go test ./api/... -v -count=1`
Expected: All PASS

- [ ] **Step 5: Commit**

```
git add api/client.go api/client_test.go api/crud_test.go test/setup.go
git commit -m "refactor: replace testing.T with Logger interface, add Content-Type header"
```

---

### Task 2: Add Product Field and Methods to APIClient

Add product state to `APIClient` with `IsRS()`, `IsPRA()`, `ProductName()` methods. Keep the global functions alive temporarily for backward compatibility until Task 4 removes them.

**Files:**
- Modify: `api/client.go`
- Modify: `api/product.go`

- [ ] **Step 1: Add Product field to APIClient in `api/client.go`**

Add to the `APIClient` struct:

```go
type APIClient struct {
	RootURL    string
	BaseURL    string
	HTTPClient *http.Client
	Product    string
	testLogger Logger
	logCtx     *context.Context
	mu         sync.Mutex
}
```

Add methods after the struct:

```go
func (c *APIClient) IsRS() bool {
	return c.Product == ProductRS
}

func (c *APIClient) IsPRA() bool {
	return c.Product == ProductPRA
}

func (c *APIClient) ProductName() string {
	return c.Product
}
```

Note: `ProductRS` and `ProductPRA` constants are defined in `api/product.go` and accessible within the package.

- [ ] **Step 2: Run tests**

Run: `go test ./api/... -v -count=1`
Expected: All PASS (additive change, nothing broken)

- [ ] **Step 3: Commit**

```
git add api/client.go
git commit -m "refactor: add Product field and methods to APIClient"
```

---

### Task 3: Update Transform and Product Signatures

Change `CopyTFtoAPI`, `CopyAPItoTF`, and `IsProductAllowed` to take a `product string` parameter. Update all callers: `api_resource.go`, `api_data_source.go`, `vault_ssh_account.go` (datasource), and all tests.

**Files:**
- Modify: `api/model_transforms.go`
- Modify: `api/product.go`
- Modify: `bt/rs/api_resource.go`
- Modify: `bt/ds/api_data_source.go`
- Modify: `bt/ds/vault_ssh_account.go`
- Modify: `api/model_transforms_test.go`
- Modify: `api/product_test.go`

- [ ] **Step 1: Update `CopyTFtoAPI` signature and body in `api/model_transforms.go`**

Change the signature:
```go
func CopyTFtoAPI(ctx context.Context, tfObj reflect.Value, apiObj reflect.Value, product string) {
```

Inside the function, replace all references to the package-level `product` variable with the `product` parameter. The variable was already named `product` so the only change needed is removing the import dependency. Specifically, update the log line at line 86:
```go
tflog.Debug(ctx, fmt.Sprintf("🍻 🔥 copyTFtoAPI check product for %s [%s][%s][%v]", fieldName, prod, product, strings.EqualFold(prod, product)))
```
This already works since the parameter shadows the global.

- [ ] **Step 2: Update `CopyAPItoTF` signature and body in `api/model_transforms.go`**

Change the signature:
```go
func CopyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value, apiType reflect.Type, product string) {
```

Inside the function, the references to `product` and `IsRS()` need updating:
- Line 419: Change `IsRS()` to `product == ProductRS`:
  ```go
  tflog.Debug(ctx, fmt.Sprintf("🍺 copyAPItoTF source obj [%+v] [%v]", apiObj, product == ProductRS))
  ```
- All other uses of `product` inside the function already work via parameter shadowing.

- [ ] **Step 3: Update `IsProductAllowed` in `api/product.go`**

Change the signature:
```go
func IsProductAllowed(ctx context.Context, i interface{}, product string) bool {
```

Update the body — replace `IsRS()` with `product == ProductRS` and `IsPRA()` with `product == ProductPRA`:
```go
func IsProductAllowed(ctx context.Context, i interface{}, product string) bool {
	s, ok := i.(RestrictsProducts)

	if !ok {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not OK [%+v]\n", i))
		return true
	}

	if !s.AllowRS() && product == ProductRS {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not RS [%+v]\n", s))
		return false
	}
	if !s.AllowPRA() && product == ProductPRA {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not PRA [%+v]\n", s))
		return false
	}
	tflog.Trace(ctx, fmt.Sprintf("🌈 Yes, allowed [%+v]\n", s))
	return true
}
```

- [ ] **Step 4: Update `bt/rs/api_resource.go` callers**

Every call to `api.CopyTFtoAPI`, `api.CopyAPItoTF`, `api.IsProductAllowed`, `api.ProductName()` gains the product from `r.ApiClient`:

- `api.IsProductAllowed(ctx, item)` → `api.IsProductAllowed(ctx, item, r.ApiClient.Product)`
- `api.ProductName()` → `r.ApiClient.ProductName()`
- `api.CopyTFtoAPI(ctx, tfObj, apiObj)` → `api.CopyTFtoAPI(ctx, tfObj, apiObj, r.ApiClient.Product)`
- `api.CopyAPItoTF(ctx, ..., apiType)` → `api.CopyAPItoTF(ctx, ..., apiType, r.ApiClient.Product)`

There are 5 `IsProductAllowed` calls, 10 `ProductName` calls, 2 `CopyTFtoAPI` calls, and 3 `CopyAPItoTF` calls in this file. All are mechanical additions of the product parameter.

- [ ] **Step 5: Update `bt/ds/api_data_source.go` callers**

Same pattern:
- `api.IsProductAllowed(ctx, item)` → `api.IsProductAllowed(ctx, item, d.apiClient.Product)`
- `api.ProductName()` → `d.apiClient.ProductName()`
- `api.CopyAPItoTF(ctx, itemObj, itemStateObj, apiType)` → `api.CopyAPItoTF(ctx, itemObj, itemStateObj, apiType, d.apiClient.Product)`

- [ ] **Step 6: Update `bt/ds/vault_ssh_account.go` caller**

Line 161:
```go
api.CopyAPItoTF(ctx, apiObj, tfObj, apiType, d.apiClient.Product)
```

- [ ] **Step 7: Update `api/model_transforms_test.go`**

Replace every `SetProductIsRS(isRS)` / `SetProductIsRS(false)` / `SetProductIsRS(true)` with a local `product` variable and pass it to the transform calls.

Pattern for the existing `TestCopyTFtoAPI`:
```go
for _, isRS := range []bool{false, true} {
    product := ProductPRA
    if isRS {
        product = ProductRS
    }

    var apiObj testAPIModel
    apiElem := reflect.ValueOf(&apiObj).Elem()
    CopyTFtoAPI(ctx, tfElem, apiElem, product)
    // ... rest unchanged
```

Same pattern for `TestCopyAPItoTF` and all the new tests added in the pre-refactor coverage work. Every `CopyTFtoAPI(ctx, ...)` call gets `, product` appended. Every `CopyAPItoTF(ctx, ...)` call gets `, product` appended.

- [ ] **Step 8: Update `api/product_test.go`**

Update `TestProductRestriction` to pass product to `IsProductAllowed`:
```go
assert.False(t, IsProductAllowed(ctx, p, ProductRS))
assert.True(t, IsProductAllowed(ctx, r, ProductRS))
assert.True(t, IsProductAllowed(ctx, n, ProductRS))
assert.True(t, IsProductAllowed(ctx, a, ProductRS))

assert.True(t, IsProductAllowed(ctx, p, ProductPRA))
assert.False(t, IsProductAllowed(ctx, r, ProductPRA))
assert.True(t, IsProductAllowed(ctx, n, ProductPRA))
assert.True(t, IsProductAllowed(ctx, a, ProductPRA))
```

The `TestProductSetting` and `TestProductName` tests can remain using the globals for now (they'll be updated in Task 4).

- [ ] **Step 9: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 10: Commit**

```
git add api/model_transforms.go api/product.go bt/rs/api_resource.go bt/ds/api_data_source.go bt/ds/vault_ssh_account.go api/model_transforms_test.go api/product_test.go
git commit -m "refactor: add product parameter to transform and product-check functions"
```

---

### Task 4: Delete Global Product State

Remove the global `var product`, `SetProductIsRS()`, `IsRS()`, `IsPRA()`, `ProductName()` from `api/product.go`. Update all remaining callers to use client methods. Update `bt/provider.go` to set `c.Product` instead of calling the global setter.

**Files:**
- Modify: `api/product.go`
- Modify: `bt/provider.go`
- Modify: `bt/rs/jump_group.go`
- Modify: `bt/rs/jump_client_installer.go`
- Modify: `bt/rs/jumpoint.go`
- Modify: `bt/rs/remote_rdp.go`
- Modify: `api/product_test.go`

- [ ] **Step 1: Clean up `api/product.go`**

Remove the global variable and free functions. The file should contain only:

```go
package api

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const ProductRS = "RS"
const ProductPRA = "PRA"

type RestrictsProducts interface {
	AllowRS() bool
	AllowPRA() bool
}

func IsProductAllowed(ctx context.Context, i interface{}, product string) bool {
	s, ok := i.(RestrictsProducts)

	if !ok {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not OK [%+v]\n", i))
		return true
	}

	if !s.AllowRS() && product == ProductRS {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not RS [%+v]\n", s))
		return false
	}
	if !s.AllowPRA() && product == ProductPRA {
		tflog.Trace(ctx, fmt.Sprintf("🌈 Not PRA [%+v]\n", s))
		return false
	}
	tflog.Trace(ctx, fmt.Sprintf("🌈 Yes, allowed [%+v]\n", s))
	return true
}
```

Delete the `CtxKey` type (unused).

- [ ] **Step 2: Update `bt/provider.go`**

Replace lines 176-177:
```go
api.SetProductIsRS(mechs.IsRS())
tflog.Info(ctx, fmt.Sprintf("Detected product is RS? [%v]", api.IsRS()))
```
With:
```go
if mechs.IsRS() {
    c.Product = api.ProductRS
} else {
    c.Product = api.ProductPRA
}
tflog.Info(ctx, fmt.Sprintf("Detected product [%s]", c.Product))
```

- [ ] **Step 3: Update individual resource files**

In `bt/rs/jump_group.go` line 121:
```go
// Before: if api.IsPRA() {
// After:
if r.ApiClient.IsPRA() {
```

In `bt/rs/jump_client_installer.go` lines 80, 87:
```go
// Before: if api.IsPRA() {
// After:
if r.ApiClient.IsPRA() {
// Before: } else if api.IsRS() {
// After:
} else if r.ApiClient.IsRS() {
```

In `bt/rs/jumpoint.go` lines 133, 135:
```go
// Before: if api.IsRS() {
// After:
if r.ApiClient.IsRS() {
// Before: } else if api.IsPRA() && plan.ProtocolTunnelEnabled.IsUnknown() {
// After:
} else if r.ApiClient.IsPRA() && plan.ProtocolTunnelEnabled.IsUnknown() {
```

In `bt/rs/remote_rdp.go` line 178:
```go
// Before: if api.IsPRA() {
// After:
if r.ApiClient.IsPRA() {
```

- [ ] **Step 4: Rewrite `api/product_test.go`**

Delete `TestProductSetting` and `TestProductName` (they tested the globals). Replace with client method tests:

```go
func TestClientProductMethods(t *testing.T) {
	t.Parallel()

	c := &APIClient{Product: ProductRS}
	assert.True(t, c.IsRS())
	assert.False(t, c.IsPRA())
	assert.Equal(t, ProductRS, c.ProductName())

	c.Product = ProductPRA
	assert.False(t, c.IsRS())
	assert.True(t, c.IsPRA())
	assert.Equal(t, ProductPRA, c.ProductName())
}

func TestConcurrentProductSafety(t *testing.T) {
	t.Parallel()

	rs := &APIClient{Product: ProductRS}
	pra := &APIClient{Product: ProductPRA}

	// Two clients with different products should not interfere
	assert.True(t, rs.IsRS())
	assert.True(t, pra.IsPRA())
	assert.False(t, rs.IsPRA())
	assert.False(t, pra.IsRS())
}
```

Keep `TestProductRestriction` as-is (it was already updated in Task 3 to pass product as a parameter).

- [ ] **Step 5: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 6: Commit**

```
git add api/product.go bt/provider.go bt/rs/jump_group.go bt/rs/jump_client_installer.go bt/rs/jumpoint.go bt/rs/remote_rdp.go api/product_test.go
git commit -m "refactor: delete global product state, use APIClient.Product everywhere"
```

---

### Task 5: Replace Unsafe Pointers With reflect.Set()

Replace all `*(*types.X)(tfObj.Field(i).Addr().UnsafePointer()) = ...` patterns in `CopyAPItoTF` with the safe `tfObj.Field(i).Set(reflect.ValueOf(...))`.

**Files:**
- Modify: `api/model_transforms.go`

- [ ] **Step 1: Replace all unsafe pointer casts**

In the `CopyAPItoTF` function, apply these replacements throughout:

```go
// Before:
*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.StringValue(...)))

// Before:
*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringNull()
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.StringNull()))

// Before:
*(*types.Int64)(tfObj.Field(i).Addr().UnsafePointer()) = types.Int64Value(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.Int64Value(...)))

// Before:
*(*types.Int64)(tfObj.Field(i).Addr().UnsafePointer()) = types.Int64Null()
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.Int64Null()))

// Before:
*(*types.Bool)(tfObj.Field(i).Addr().UnsafePointer()) = types.BoolValue(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.BoolValue(...)))

// Before:
*(*types.Bool)(tfObj.Field(i).Addr().UnsafePointer()) = types.BoolNull()
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.BoolNull()))

// Before:
*(*types.Set)(tfObj.Field(i).Addr().UnsafePointer()) = types.SetNull(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.SetNull(...)))

// Before:
*(*types.Set)(tfObj.Field(i).Addr().UnsafePointer()) = types.SetValueMust(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.SetValueMust(...)))

// Before (in Set round-trip):
*(*types.Set)(tfObj.Field(i).Addr().UnsafePointer()), _ = types.SetValueFrom(...)
// After:
v, _ := types.SetValueFrom(...)
tfObj.Field(i).Set(reflect.ValueOf(v))

// Before:
*(*types.Object)(tfObj.Field(i).Addr().UnsafePointer()) = types.ObjectNull(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.ObjectNull(...)))

// Before:
*(*types.Object)(tfObj.Field(i).Addr().UnsafePointer()) = ov
// After:
tfObj.Field(i).Set(reflect.ValueOf(ov))

// Before (FilterRules list):
*(*types.List)(tfObj.Field(i).Addr().UnsafePointer()) = listVal
// After:
tfObj.Field(i).Set(reflect.ValueOf(listVal))

// Before (FilterRules string fallback):
*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(...)
// After:
tfObj.Field(i).Set(reflect.ValueOf(types.StringValue(...)))
```

Also update the FIXME comment at the top of the function to remove the unsafe pointer explanation, since it's no longer applicable.

- [ ] **Step 2: Run tests**

Run: `go test ./api/... -v -count=1`
Expected: All PASS — the pre-refactor tests verify every type path

- [ ] **Step 3: Commit**

```
git add api/model_transforms.go
git commit -m "refactor: replace unsafe pointer casts with reflect.Value.Set() in CopyAPItoTF"
```

---

### Task 6: Replace Panics With Error Returns

Change `CopyAPItoTF` to return `error`. Replace the three panic sites with error returns. Update all callers to check the error. Add a test for the error path.

**Files:**
- Modify: `api/model_transforms.go`
- Modify: `bt/rs/api_resource.go`
- Modify: `bt/ds/api_data_source.go`
- Modify: `bt/ds/vault_ssh_account.go`
- Modify: `api/model_transforms_test.go`

- [ ] **Step 1: Change `CopyAPItoTF` signature to return error**

```go
func CopyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value, apiType reflect.Type, product string) error {
```

At the very end of the function (after the for loop), add:
```go
	return nil
}
```

Replace the three panic sites:

Line ~741 (unhandled set element type):
```go
// Before:
panic("Unhandled set type: " + field.Index(j).Kind().String())
// After:
return fmt.Errorf("unhandled set element type for field %s: %s", tfObjField.Name, field.Index(j).Kind().String())
```

Line ~748 (error converting set):
```go
// Before:
panic("Error converting go set to TF object: " + err.Errors()[0].Detail())
// After:
return fmt.Errorf("error converting set for field %s: %s", tfObjField.Name, err.Errors()[0].Detail())
```

Line ~753 (unknown encoded type):
```go
// Before:
panic("Unknown encoded type in struct: " + field.Kind().String())
// After:
return fmt.Errorf("unknown encoded type for field %s: %s", tfObjField.Name, fieldKind.String())
```

- [ ] **Step 2: Update callers in `bt/rs/api_resource.go`**

There are 3 calls to `CopyAPItoTF`. Each needs error handling. Pattern:

```go
// Before:
api.CopyAPItoTF(ctx, newApiObj, tfObj, apiType, r.ApiClient.Product)
// After:
if err := api.CopyAPItoTF(ctx, newApiObj, tfObj, apiType, r.ApiClient.Product); err != nil {
    resp.Diagnostics.AddError(
        "Error converting API response",
        "Unexpected error converting API response to Terraform state: "+err.Error(),
    )
    return
}
```

Apply to all 3 calls (in Create, Read, and Update methods).

- [ ] **Step 3: Update caller in `bt/ds/api_data_source.go`**

In the `doFilteredRead` method:

```go
// Before:
api.CopyAPItoTF(ctx, itemObj, itemStateObj, apiType, d.apiClient.Product)
// After:
if err := api.CopyAPItoTF(ctx, itemObj, itemStateObj, apiType, d.apiClient.Product); err != nil {
    resp.Diagnostics.AddError(
        fmt.Sprintf("Error converting %s API response", d.printableName()),
        "Unexpected error converting API response to Terraform state: "+err.Error(),
    )
    return nil
}
```

- [ ] **Step 4: Update caller in `bt/ds/vault_ssh_account.go`**

```go
// Before:
api.CopyAPItoTF(ctx, apiObj, tfObj, apiType, d.apiClient.Product)
// After:
if err := api.CopyAPItoTF(ctx, apiObj, tfObj, apiType, d.apiClient.Product); err != nil {
    resp.Diagnostics.AddError(
        "Error converting SSH account API response",
        "Unexpected error converting API response to Terraform state: "+err.Error(),
    )
    return
}
```

- [ ] **Step 5: Update test calls in `api/model_transforms_test.go`**

Every call to `CopyAPItoTF` now returns an error. Update all test calls:

```go
// Before:
CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)
// After:
err := CopyAPItoTF(ctx, apiElem, tfElem, apiType, product)
assert.Nil(t, err)
```

Apply to all `CopyAPItoTF` calls in all test functions.

- [ ] **Step 6: Add test for error return on unhandled type**

Add new test model types and a test function:

```go
type testTFModelWithFloat struct {
	ID       types.String
	BadField types.Float64
}

type testAPIModelWithFloat struct {
	ID       *int
	BadField float64
}

func TestCopyAPItoTF_UnhandledTypeReturnsError(t *testing.T) {
	ctx := context.Background()

	id := 1
	apiObj := &testAPIModelWithFloat{
		ID:       &id,
		BadField: 3.14,
	}

	tfObj := &testTFModelWithFloat{
		ID:       types.StringUnknown(),
		BadField: types.Float64Unknown(),
	}

	apiElem := reflect.ValueOf(apiObj).Elem()
	apiType := reflect.TypeOf(apiObj).Elem()
	tfElem := reflect.ValueOf(tfObj).Elem()

	err := CopyAPItoTF(ctx, apiElem, tfElem, apiType, ProductPRA)
	assert.NotNil(t, err, "CopyAPItoTF should return error for unhandled float64 type")
	assert.Contains(t, err.Error(), "BadField")
}
```

Note: `types.Float64` is a real Terraform framework type but `float64` is not handled in the switch statement. This test verifies the error path instead of a panic.

- [ ] **Step 7: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS, including the new error path test

- [ ] **Step 8: Commit**

```
git add api/model_transforms.go api/model_transforms_test.go bt/rs/api_resource.go bt/ds/api_data_source.go bt/ds/vault_ssh_account.go
git commit -m "refactor: CopyAPItoTF returns error instead of panicking"
```

---

### Task 7: Fix Provider Error Handling and Typos

Fix the nil-client-after-error bug, error variable shadowing, error message typos, stray `q`, and snake_case variable naming in `bt/provider.go`.

**Files:**
- Modify: `bt/provider.go`

- [ ] **Step 1: Fix the Configure method**

Replace the entire Configure method body from the `tflog.Debug(ctx, "Creating BT API Client")` line through the end. The new version:

```go
	ctx = tflog.SetField(ctx, "bt_api_host", host)
	ctx = tflog.SetField(ctx, "bt_client_id", clientID)
	ctx = tflog.SetField(ctx, "bt_client_secret", clientSecret)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "bt_client_secret")

	tflog.Debug(ctx, "Creating BT API Client")
	c, err := api.NewClient(host, &clientID, &clientSecret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BeyondTrust SRA API Client",
			"An unexpected error occurred when creating the BeyondTrust SRA API Client. "+
				"Error: "+err.Error(),
		)
		return
	}
	c.SetLogContext(&ctx)

	mechs, err := api.Get[api.MechList](c)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to determine BeyondTrust Product",
			"An unexpected error occurred when querying the SRA Instance. "+
				"Error: "+err.Error(),
		)
		return
	}

	if mechs.IsRS() {
		c.Product = api.ProductRS
	} else {
		c.Product = api.ProductPRA
	}
	tflog.Info(ctx, fmt.Sprintf("Detected product [%s]", c.Product))

	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured BT API client", map[string]any{"success": true})
```

Note key changes: early return after `NewClient` error, `SetLogContext` after nil check, added periods and spaces in error messages before "Error:", early return after `Get` error.

- [ ] **Step 2: Fix variable naming — `client_id` → `clientID`, `client_secret` → `clientSecret`**

Lines 99-101:
```go
// Before:
host := os.Getenv("BT_API_HOST")
client_id := os.Getenv("BT_CLIENT_ID")
client_secret := os.Getenv("BT_CLIENT_SECRET")
// After:
host := os.Getenv("BT_API_HOST")
clientID := os.Getenv("BT_CLIENT_ID")
clientSecret := os.Getenv("BT_CLIENT_SECRET")
```

Update all references: `config.ClientId` → `clientID` assignments (line 108), `config.ClientSecret` → `clientSecret` assignments (line 112), the empty-string checks (lines 125, 135).

- [ ] **Step 3: Fix error message typos**

Line 128: `"Missing BeyondTrust SRA API Username"` → `"Missing BeyondTrust SRA API Client ID"`
Line 129: `"...client_id."` → `"...Client ID."`
Line 130: `"Set the username value"` → `"Set the client_id value"`

Line 138: `"Missing BeyondTrust SRA API Password"` → `"Missing BeyondTrust SRA API Client Secret"`
Line 139: `"...client_secret."` → `"...Client Secret."`
Line 140: `"Set the password value"` → `"Set the client_secret value"`

- [ ] **Step 4: Fix stray `q` in markdown**

Line 65 of the Schema markdown description:
```go
// Before (around line 65):
q
## Configuration
// After:
## Configuration
```

- [ ] **Step 5: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 6: Commit**

```
git add bt/provider.go
git commit -m "fix: provider error handling, typos, and naming conventions"
```

---

### Task 8: Fix strconv.Atoi Error Handling in api_resource.go

Add error checking to all `strconv.Atoi` calls in the generic resource CRUD methods.

**Files:**
- Modify: `bt/rs/api_resource.go`

- [ ] **Step 1: Fix Atoi in Read method**

```go
// Before (around line 171):
id, _ := strconv.Atoi(tfId.ValueString())
// After:
id, err := strconv.Atoi(tfId.ValueString())
if err != nil {
    resp.Diagnostics.AddError(
        "Invalid resource ID",
        fmt.Sprintf("Could not parse resource ID [%s] as integer: %s", tfId.ValueString(), err.Error()),
    )
    return
}
```

- [ ] **Step 2: Fix Atoi in Update method**

Same pattern around line 226:
```go
id, err := strconv.Atoi(tfId.ValueString())
if err != nil {
    resp.Diagnostics.AddError(
        "Invalid resource ID",
        fmt.Sprintf("Could not parse resource ID [%s] as integer: %s", tfId.ValueString(), err.Error()),
    )
    return
}
```

- [ ] **Step 3: Fix Atoi in Delete method**

Same pattern around line 269:
```go
id, err := strconv.Atoi(tfId.ValueString())
if err != nil {
    resp.Diagnostics.AddError(
        "Invalid resource ID",
        fmt.Sprintf("Could not parse resource ID [%s] as integer: %s", tfId.ValueString(), err.Error()),
    )
    return
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All PASS

- [ ] **Step 5: Commit**

```
git add bt/rs/api_resource.go
git commit -m "fix: check strconv.Atoi errors in generic resource CRUD"
```

---

### Task 9: Cleanup — Dead Code, x/exp/slices, Stray Comment

Delete the ~140 lines of commented-out FilterRules code, replace `golang.org/x/exp/slices` with stdlib `slices`, and remove the stale `convertToStruct` comment block.

**Files:**
- Modify: `api/model_transforms.go`
- Run: `go mod tidy`

- [ ] **Step 1: Delete commented-out FilterRules code**

Delete lines 260-404 (the entire `case reflect.Slice:` commented block inside `CopyTFtoAPI`). This is the block that starts with `// if tfObjField.Name == "FilterRules" {` and contains ~140 lines of old CIDR/port parsing logic.

- [ ] **Step 2: Delete stale `convertToStruct` comment**

Delete lines 45-53 (the commented-out `convertToStruct` function).

- [ ] **Step 3: Replace x/exp/slices import**

```go
// Before:
"golang.org/x/exp/slices"
// After:
"slices"
```

- [ ] **Step 4: Run go mod tidy**

Run: `go mod tidy`

This may remove `golang.org/x/exp` from `go.mod` if nothing else depends on it.

- [ ] **Step 5: Run tests and verify build**

Run: `go build ./... && go vet ./... && go test ./api/... ./bt/... -v -count=1`
Expected: All pass

- [ ] **Step 6: Commit**

```
git add api/model_transforms.go go.mod go.sum
git commit -m "cleanup: delete dead code, replace x/exp/slices with stdlib"
```

---

### Task 10: Final Verification and Coverage Check

Run the complete test suite, check coverage improvements, and verify the build.

**Files:**
- None (verification only)

- [ ] **Step 1: Full build and vet**

Run: `go build ./... && go vet ./...`
Expected: Clean output, no errors

- [ ] **Step 2: Run all unit tests**

Run: `go test ./api/... ./bt/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 3: Check coverage**

Run: `go test ./api/... -coverprofile=/tmp/api_refactor_coverage.out -count=1 && go tool cover -func=/tmp/api_refactor_coverage.out | grep -E "(CopyTFtoAPI|CopyAPItoTF|UnmarshalJSON|IsProductAllowed|doRequest|total)"`

Expected: Coverage should be higher than pre-refactor (59.4%) since we added new tests (concurrent product, error return path, Content-Type header).

- [ ] **Step 4: Verify no unsafe imports remain**

Run: `grep -r "UnsafePointer\|\"unsafe\"" api/`
Expected: No matches

- [ ] **Step 5: Verify no panics remain in model_transforms.go**

Run: `grep "panic(" api/model_transforms.go`
Expected: No matches

- [ ] **Step 6: Verify no testing import in client.go**

Run: `grep '"testing"' api/client.go`
Expected: No matches

- [ ] **Step 7: Verify no global product state**

Run: `grep "var product\|func SetProductIsRS\|func IsRS()\|func IsPRA()\|func ProductName()" api/product.go`
Expected: No matches
