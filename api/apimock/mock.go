// Package apimock is an in-process stand-in for the appliance's configuration
// API, for tests that need its paging behaviour rather than a canned response.
//
// It models what the real controllers do (ConfigurationBaseController,
// PaginatingControllerTrait):
//   - every collection GET is paged: per_page defaults to 100 and is capped at
//     100, and current_page is clamped into [1, last page], so asking past the
//     end returns the last page again rather than an empty one;
//   - every collection response carries X-BT-Pagination-Total and
//     X-BT-Pagination-Last-Page (and Current-Page, Per-Page);
//   - GET and DELETE of <collection>/<id> address a row by its "id".
//
// It deliberately imports only the standard library. api's own tests are
// package api, so an apimock -> api import would be an import cycle; callers
// build their client themselves:
//
//	srv := apimock.New()
//	defer srv.Close()
//	c := &api.APIClient{RootURL: srv.URL, BaseURL: srv.URL + apimock.BasePath, HTTPClient: srv.Client()}
package apimock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// BasePath is the configuration API prefix the provider puts on every request.
const BasePath = "/api/config/v1"

const maxPerPage = 100

// Row is one object in a collection, as the appliance would encode it.
type Row = map[string]any

// Collection is the data behind one collection endpoint.
type Collection struct {
	Rows []Row

	// Status, when non-zero, answers every GET of the collection (not of its
	// rows) with that status and no body.
	Status int

	// NoHeaders drops the X-BT-Pagination-* headers, for an endpoint that does
	// not page.
	NoHeaders bool

	// OnPage runs before a page is served, with the server lock held, so a test
	// can change Rows between pages: a delete mid-read, growth, or a reorder.
	OnPage func(c *Collection, page int)

	// Match decides whether a row satisfies the request's filter parameters
	// (everything except per_page and current_page). Nil means each filter must
	// equal the row field of the same name; a filter naming a field the row does
	// not have is ignored.
	Match func(row Row, filter url.Values) bool
}

// Server is an httptest server backed by Collections keyed on their path below
// BasePath ("vault/account", "group-policy/7/jump-group").
type Server struct {
	*httptest.Server

	mu          sync.Mutex
	collections map[string]*Collection
	handlers    map[string]http.HandlerFunc
	calls       map[string]int
	requests    []string
}

func New() *Server {
	s := &Server{
		collections: map[string]*Collection{},
		handlers:    map[string]http.HandlerFunc{},
		calls:       map[string]int{},
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	return s
}

// Set installs (or replaces) the collection at path and returns it.
func (s *Server) Set(path string, c *Collection) *Collection {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collections[path] = c
	return c
}

// Rows is shorthand for Set(path, &Collection{Rows: rows}).
func (s *Server) Rows(path string, rows ...Row) *Collection {
	return s.Set(path, &Collection{Rows: rows})
}

// Handle overrides one method and path (below BasePath, no query) with h. It
// takes precedence over any collection.
func (s *Server) Handle(method, path string, h http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method+" "+path] = h
}

// Calls reports how many requests the server saw for method and path (below
// BasePath, no query).
func (s *Server) Calls(method, path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[method+" "+path]
}

// CallsWithMethod reports how many requests used method, on any path.
func (s *Server) CallsWithMethod(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for k, v := range s.calls {
		if strings.HasPrefix(k, method+" ") {
			n += v
		}
	}
	return n
}

// Requests returns every request seen so far as "METHOD path?query", in order.
func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

// Collection returns the collection at path, for a test that inspects it after
// the fact (a DELETE removes the row).
func (s *Server) Collection(path string) *Collection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.collections[path]
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, BasePath), "/")

	s.mu.Lock()
	s.calls[r.Method+" "+path]++
	req := r.Method + " " + path
	if r.URL.RawQuery != "" {
		req += "?" + r.URL.RawQuery
	}
	s.requests = append(s.requests, req)
	h := s.handlers[r.Method+" "+path]
	s.mu.Unlock()

	if h != nil {
		h(w, r)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.collections[path]; ok {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.servePage(w, r, c)
		return
	}

	// <collection>/<id>
	i := strings.LastIndex(path, "/")
	if i < 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	c, ok := s.collections[path[:i]]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id := path[i+1:]
	for n, row := range c.Rows {
		if fmt.Sprint(row["id"]) != id {
			continue
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, nil, row)
		case http.MethodDelete:
			c.Rows = append(c.Rows[:n:n], c.Rows[n+1:]...)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// servePage answers one collection GET the way applyPagination does.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request, c *Collection) {
	if c.Status != 0 {
		w.WriteHeader(c.Status)
		return
	}

	q := r.URL.Query()
	perPage := clampedInt(q.Get("per_page"), maxPerPage, 1, maxPerPage)
	requested := clampedInt(q.Get("current_page"), 1, 1, int(^uint(0)>>1))

	if c.OnPage != nil {
		c.OnPage(c, requested)
	}

	filter := url.Values{}
	for k, v := range q {
		if k != "per_page" && k != "current_page" {
			filter[k] = v
		}
	}
	match := c.Match
	if match == nil {
		match = fieldsEqual
	}
	rows := []Row{}
	for _, row := range c.Rows {
		if match(row, filter) {
			rows = append(rows, row)
		}
	}

	total := len(rows)
	last := max((total+perPage-1)/perPage, 1)
	page := min(requested, last)
	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)

	var header http.Header
	if !c.NoHeaders {
		header = http.Header{}
		header.Set("X-BT-Pagination-Current-Page", strconv.Itoa(page))
		header.Set("X-BT-Pagination-Per-Page", strconv.Itoa(perPage))
		header.Set("X-BT-Pagination-Last-Page", strconv.Itoa(last))
		header.Set("X-BT-Pagination-Total", strconv.Itoa(total))
	}
	writeJSON(w, http.StatusOK, header, rows[start:end])
}

func fieldsEqual(row Row, filter url.Values) bool {
	for k := range filter {
		if v, ok := row[k]; ok && fmt.Sprint(v) != filter.Get(k) {
			return false
		}
	}
	return true
}

// clampedInt parses s, using def when it is absent or not a number, and clamps
// the result into [lo, hi].
func clampedInt(s string, def, lo, hi int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		v = def
	}
	return min(max(v, lo), hi)
}

func writeJSON(w http.ResponseWriter, status int, header http.Header, v any) {
	for k, vs := range header {
		w.Header()[k] = vs
	}
	w.Header().Set("Content-Type", "application/json")
	body, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
