package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type APIResource interface {
	Endpoint() string
}

// StatusError is a non-2xx response from the API.
//
// The status code is carried as a field rather than left to be recovered from
// the message, because the message embeds the response body: a body that happens
// to contain "status: 404" would otherwise read as a 404, and the branches that
// ask are the ones that drop a resource from Terraform state.
//
// Error() keeps the format the API layer has always produced. Diagnostics
// surface it to operators verbatim, so the wording is part of the interface.
type StatusError struct {
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("status: %d, body: %s", e.Status, e.Body)
}

// HasStatus reports whether err is an API status error carrying the given code.
func HasStatus(err error, status int) bool {
	var statusErr *StatusError
	return errors.As(err, &statusErr) && statusErr.Status == status
}

// IsNotFound reports whether err represents a 404 from the API. Callers use it
// to treat a resource as deleted (e.g. remove it from Terraform state) rather
// than surfacing a hard error.
func IsNotFound(err error) bool {
	return HasStatus(err, http.StatusNotFound)
}

func Get[I APIResource](c *APIClient) (*I, error) {
	var item I
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.RootURL, item.Endpoint()), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func Post[I APIResource](c *APIClient, path string, item I, ignoreReturn bool) (*I, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s/%s", c.BaseURL, item.Endpoint(), path), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if ignoreReturn {
		return nil, nil
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func ListItems[I APIResource](c *APIClient, query ...map[string]string) ([]I, error) {
	var tmp I
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, tmp.Endpoint()), nil)
	if err != nil {
		return nil, err
	}

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query[0] {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	items := []I{}
	err = json.Unmarshal(resp, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

const (
	// listPageSize is the appliance's per-page cap
	// (ConfigurationBaseController::__construct sets maxPerPage to 100).
	listPageSize = 100
	// listMaxPages bounds one listing pass at 100,000 rows.
	listMaxPages = 1000
	// listAttempts is how many full passes ListAllItemsEndpoint makes before
	// giving up on a list that keeps changing under it.
	listAttempts = 3

	paginationLastPageHeader = "X-BT-Pagination-Last-Page"
	paginationTotalHeader    = "X-BT-Pagination-Total"
)

// listRetryBackoff is the pause before each retry, so the retries do not all
// land in the same burst of writes that broke the previous pass. A variable so
// the package's own tests can shorten it.
var listRetryBackoff = []time.Duration{time.Second, 2 * time.Second}

// UnstableListCause says why ListAllItemsEndpoint could not return a complete list.
type UnstableListCause int

const (
	// ListChanged: the reported total or last page moved between pages, or the
	// rows read did not add up to the total, so objects were created, deleted or
	// reordered during the read.
	ListChanged UnstableListCause = iota
	// ListOverlapped: the headers held steady but pages repeated rows. The
	// appliance sorts some indexes on a non-unique column (jump groups by name),
	// and OFFSET paging can split ties differently per page.
	ListOverlapped
	// ListPageCap: the list did not end within listMaxPages pages.
	ListPageCap
)

// ErrUnstableList means a paginated list could not be read completely. Callers
// must treat the list as unknown, never as the rows that were read: a partial
// list reaching for_each, import discovery or the E2E sweeper reads as
// "these objects do not exist".
type ErrUnstableList struct {
	Endpoint string
	Cause    UnstableListCause
	Attempts int
	Got      int // unique rows read in the last pass
	Total    int // the total the appliance reported on page 1 of that pass
}

func (e *ErrUnstableList) Error() string {
	return e.Describe(e.Endpoint)
}

// Describe renders the error with the advice that fits its cause. what names the
// object type in terms the practitioner knows (a data source passes its own type
// rather than the API path).
func (e *ErrUnstableList) Describe(what string) string {
	switch e.Cause {
	case ListOverlapped:
		return fmt.Sprintf("The appliance returned overlapping pages for %s in %d consecutive attempts (last: %d unique of %d reported); narrow the list with a filter such as `name` or `code_name`.",
			what, e.Attempts, e.Got, e.Total)
	case ListPageCap:
		return fmt.Sprintf("Listing %s stopped at the %d-page limit after reading %d objects without reaching the end of the list; add filters so fewer objects match.",
			what, listMaxPages, e.Got)
	default:
		return fmt.Sprintf("The list of %s objects changed while it was being read, in %d consecutive attempts (last: %d unique of %d reported). Creating, deleting or updating objects of this type causes this; for vault accounts that includes credential check-outs and password rotations. Re-run; if it keeps failing, add filters (for example `account_group_id`, `name`, `tag`, `jump_group_id`) so fewer objects match.",
			what, e.Attempts, e.Got, e.Total)
	}
}

// ListAllItems returns every item of type I matching query, across all pages.
// Rows are deduplicated on their ID field. Membership types have no ID and a
// zero value whose Endpoint() cannot be called; use ListAllItemsEndpoint.
func ListAllItems[I APIResource](c *APIClient, query map[string]string) ([]I, error) {
	if f, ok := reflect.TypeFor[I]().FieldByName("ID"); !ok || f.Type != reflect.TypeFor[*int]() {
		return nil, fmt.Errorf("ListAllItems: %s has no ID *int field to deduplicate on; use ListAllItemsEndpoint", reflect.TypeFor[I]())
	}
	var tmp I
	return ListAllItemsEndpoint(c, tmp.Endpoint(), query, idKey[I])
}

// idKey is ListAllItems' dedup key: the item's ID, or "" when the API omitted it.
func idKey[I any](item *I) string {
	id := reflect.ValueOf(item).Elem().FieldByName("ID")
	if id.IsNil() {
		return ""
	}
	return strconv.Itoa(*id.Interface().(*int))
}

// ListAllItemsEndpoint pages through endpoint and returns every row, in the
// order first read, deduplicated on key (the first occurrence wins, so items[0]
// is the same row a single unpaged GET would have put first).
//
// The appliance's paging is OFFSET over indexes that mostly have no ORDER BY,
// and it clamps a page past the end to the last page instead of returning an
// empty one (PaginatingControllerTrait::applyPagination). So:
//   - the loop is bounded by the X-BT-Pagination-Last-Page header from page 1,
//     never by "a short or empty page", which the clamp never produces;
//   - a pass is unstable if a later page reports a different Total or Last-Page
//     than page 1 did, or if the unique rows read do not add up to Total. Either
//     means rows moved under the read and some may have been skipped. An
//     unstable pass is restarted from page 1, up to listAttempts passes in all,
//     and then *ErrUnstableList is returned rather than a partial list.
//
// Without the headers (an endpoint that does not paginate), a pass stops at the
// first page that is short or adds no new rows, and no stability check is
// possible.
func ListAllItemsEndpoint[I any](c *APIClient, endpoint string, query map[string]string, key func(*I) string) ([]I, error) {
	var unstable *ErrUnstableList
	for attempt := 1; attempt <= listAttempts; attempt++ {
		if attempt > 1 {
			c.LogString("🔁 ListAllItems [%s]: attempt %d of %d was unstable (%d unique of %d reported), retrying",
				endpoint, attempt-1, listAttempts, unstable.Got, unstable.Total)
			time.Sleep(listRetryBackoff[attempt-2])
		}

		items, err := listPass(c, endpoint, query, key)
		if !errors.As(err, &unstable) {
			return items, err
		}
		unstable.Attempts = attempt
		if unstable.Cause == ListPageCap {
			// Reading it again cannot make it shorter.
			return nil, unstable
		}
	}
	return nil, unstable
}

// listPass makes one pass over every page of endpoint.
func listPass[I any](c *APIClient, endpoint string, query map[string]string, key func(*I) string) ([]I, error) {
	items := []I{}
	seen := map[string]bool{}
	overlapped := false
	lastPage, total, paged := 0, 0, false

	for page := 1; ; page++ {
		if page > listMaxPages {
			return nil, &ErrUnstableList{Endpoint: endpoint, Cause: ListPageCap, Got: len(items), Total: total}
		}

		rows, header, err := listPage[I](c, endpoint, query, page)
		if err != nil {
			return nil, err
		}

		l, lOK := intHeader(header, paginationLastPageHeader)
		t, tOK := intHeader(header, paginationTotalHeader)
		if page == 1 {
			lastPage, total, paged = l, t, lOK && tOK
			if paged && lastPage > listMaxPages {
				return nil, &ErrUnstableList{Endpoint: endpoint, Cause: ListPageCap, Total: total}
			}
		} else if paged && (l != lastPage || t != total || !lOK || !tOK) {
			return nil, &ErrUnstableList{Endpoint: endpoint, Cause: ListChanged, Got: len(items), Total: total}
		}

		added := 0
		for i := range rows {
			k := key(&rows[i])
			if seen[k] {
				overlapped = true
				continue
			}
			seen[k] = true
			items = append(items, rows[i])
			added++
		}

		if paged {
			if page >= lastPage {
				break
			}
		} else if len(rows) < listPageSize || added == 0 {
			break
		}
	}

	if paged && len(items) != total {
		cause := ListChanged
		if overlapped {
			cause = ListOverlapped
		}
		return nil, &ErrUnstableList{Endpoint: endpoint, Cause: cause, Got: len(items), Total: total}
	}
	return items, nil
}

// listPage fetches one page of endpoint.
func listPage[I any](c *APIClient, endpoint string, query map[string]string, page int) ([]I, http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	if err != nil {
		return nil, nil, err
	}

	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	q.Set("per_page", strconv.Itoa(listPageSize))
	q.Set("current_page", strconv.Itoa(page))
	req.URL.RawQuery = q.Encode()

	body, header, err := c.doRequestWithHeaders(req)
	if err != nil {
		return nil, nil, err
	}

	rows := []I{}
	if body != nil {
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, nil, err
		}
	}
	return rows, header, nil
}

func intHeader(h http.Header, name string) (int, bool) {
	v, err := strconv.Atoi(h.Get(name))
	return v, err == nil
}

func GetItem[I APIResource](c *APIClient, id *int) (*I, error) {
	var item I
	endpoint := item.Endpoint()
	if id != nil {
		endpoint = fmt.Sprintf("%s/%d", endpoint, *id)
	}

	return GetItemEndpoint[I](c, endpoint)
}

func GetItemEndpoint[I APIResource](c *APIClient, endpoint string) (*I, error) {
	var item I
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// ListItemsEndpoint performs a GET against a specific endpoint and returns the
// result as a slice. The group-policy membership read endpoints document a
// single JSON object in the spec, but the appliance has been observed
// returning a JSON array for at least one of them — so this decodes whichever
// shape the endpoint actually returns (a lone object becomes a one-element
// slice). Returns an empty slice on a 204/no-content response.
func ListItemsEndpoint[I APIResource](c *APIClient, endpoint string) ([]I, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}

	items := []I{}
	arrErr := json.Unmarshal(body, &items)
	if arrErr == nil {
		return items, nil
	}

	// Only fall back to single-object decode when the top-level value isn't an
	// array at all (encoding/json reports that as an UnmarshalTypeError whose
	// Value is "object" and, because the error is at the root, Field is
	// empty). If the body is an array but one of its elements fails to
	// decode, arrErr is the more useful diagnostic; the single-object decode
	// below would otherwise mask it behind a confusing "cannot unmarshal
	// object into Go value of type ..." error. Value alone is not enough to
	// tell the two apart: an array element that is itself object-shaped where
	// a scalar field was expected also reports Value == "object", but with
	// Field naming that element's field — checking Field == "" excludes it.
	var typeErr *json.UnmarshalTypeError
	if !errors.As(arrErr, &typeErr) || typeErr.Value != "object" || typeErr.Field != "" {
		return nil, arrErr
	}

	// Endpoint returned a single object rather than an array; decode it as one.
	var single I
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []I{single}, nil
}

// CreateItem POSTs item to its endpoint.
//
// It returns (nil, nil) when the API answers 204 No Content: the item WAS
// created, there is simply no body to decode. Callers that dereference or
// store the result must handle a nil item — writing it straight into Terraform
// state produces a null attribute and an "inconsistent result after apply".
// Echo the request back instead; the server accepted it.
//
// Note the asymmetry with UpdateItemEndpoint, which does not check for an
// empty body: a 204 on PATCH surfaces as "unexpected end of JSON input"
// rather than (nil, nil).
func CreateItem[I APIResource](c *APIClient, item I) (*I, error) {
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	// Endpoint and size only — see the logging policy in api/logging.go.
	c.LogString("✅ CreateItem [%s]: %d byte payload", item.Endpoint(), len(rb))

	var newItem I
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", c.BaseURL, item.Endpoint()), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if body == nil {
		// success, but no content (204)
		return nil, nil
	}

	err = json.Unmarshal(body, &newItem)
	if err != nil {
		return nil, err
	}

	return &newItem, nil
}

func UpdateItem[I APIResource](c *APIClient, item I) (*I, error) {
	itemObj := reflect.ValueOf(item)
	id := itemObj.FieldByName("ID").Elem().Int()
	endpoint := fmt.Sprintf("%s/%d", item.Endpoint(), id)

	return UpdateItemEndpoint(c, item, endpoint)
}
func UpdateItemEndpoint[I APIResource](c *APIClient, item I, endpoint string) (*I, error) {
	rb, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var newItem I
	err = json.Unmarshal(body, &newItem)
	if err != nil {
		return nil, err
	}

	return &newItem, nil
}

func DeleteItem[I APIResource](c *APIClient, id *int) error {
	var tmp I
	endpoint := tmp.Endpoint()
	if id != nil {
		endpoint = fmt.Sprintf("%s/%d", endpoint, *id)
	}

	return DeleteItemEndpoint[I](c, endpoint)
}
func DeleteItemEndpoint[I APIResource](c *APIClient, endpoint string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
