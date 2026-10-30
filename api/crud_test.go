package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"terraform-provider-sra/api/apimock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testAPIResource struct {
	ID       *int `json:"-"`
	Location string
}

func (t testAPIResource) Endpoint() string {
	return fmt.Sprintf("test-resource/%s", t.Location)
}

func TestGet(t *testing.T) {
	t.Parallel()

	contentString := `{"Location":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "test-resource/") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		resp, err := Get[testAPIResource](c)
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}
}

func TestPost(t *testing.T) {
	t.Parallel()

	contentString := `{"Location":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "test-resource/the_barricade/post") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "test-resource/the_barricade/error") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		test := testAPIResource{nil, "the_barricade"}
		resp, err := Post(c, "post", test, false)
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}

	{
		test := testAPIResource{nil, "the_barricade"}
		resp, err := Post(c, "post", test, true)
		assert.Nil(t, err)
		assert.Nil(t, resp)
	}

	{
		test := testAPIResource{nil, "the_barricade"}
		resp, err := Post(c, "error", test, true)
		assert.Equal(t, "status: 400, body: error", err.Error())
		assert.Nil(t, resp)
	}
}

func TestListItems(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "test-resource/") {
				w.WriteHeader(http.StatusOK)
				var err error
				if r.URL.Query().Get("name") == "Cosette" {
					_, err = w.Write([]byte(`[{"Location":"the_apartment"}]`))
				} else {
					_, err = w.Write([]byte(`[{"Location":"the_sewers"}]`))
				}
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		resp, err := ListItems[testAPIResource](c)
		assert.Nil(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "the_sewers", resp[0].Location)
	}

	{
		resp, err := ListItems[testAPIResource](c, map[string]string{"name": "Cosette"})
		assert.Nil(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "the_apartment", resp[0].Location)
	}
}

// B3: ListItemsEndpoint must decode either the array or single-object shape
// the group-policy membership endpoints are known to return, and — when the
// body is genuinely an array but one element fails to decode — report that
// array-decode error rather than masking it behind the single-object
// fallback's unrelated "cannot unmarshal array into Go value" message.
func TestListItemsEndpoint(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
			return
		}

		assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "valid-array"):
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[{"Location":"the_sewers"}]`))
			assert.Nil(t, err)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "single-object"):
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"Location":"the_barricade"}`))
			assert.Nil(t, err)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "bad-element"):
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[{"Location":123}]`))
			assert.Nil(t, err)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "bad-nested-object"):
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[{"Location":{"deep":1}}]`))
			assert.Nil(t, err)
		default:
			assert.Fail(t, "Bad request", r.URL)
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		resp, err := ListItemsEndpoint[testAPIResource](c, "valid-array")
		assert.Nil(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "the_sewers", resp[0].Location)
	}

	{
		resp, err := ListItemsEndpoint[testAPIResource](c, "single-object")
		assert.Nil(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "the_barricade", resp[0].Location)
	}

	{
		resp, err := ListItemsEndpoint[testAPIResource](c, "bad-element")
		assert.Nil(t, resp)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Location", "should report the array-decode error naming the field that failed")
		}
	}

	// Finding 8: an array element that is itself object-shaped where a scalar
	// field was expected also reports UnmarshalTypeError.Value == "object" —
	// the same value the top-level single-object case reports. Without also
	// checking Field (empty only at the root), this used to take the
	// single-object fallback and mask the array-decode error.
	{
		resp, err := ListItemsEndpoint[testAPIResource](c, "bad-nested-object")
		assert.Nil(t, resp)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Location", "should report the array-decode error naming the field that failed, not the single-object fallback's error")
		}
	}
}

func TestGetItem(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "test-resource//1") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"Location":"the_sewers"}`))
				assert.Nil(t, err)
			} else if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "test-resource/") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"Location":"the_barricade"}`))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		id := 1
		resp, err := GetItem[testAPIResource](c, &id)
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}

	{
		resp, err := GetItem[testAPIResource](c, nil)
		assert.Nil(t, err)
		assert.Equal(t, "the_barricade", resp.Location)
	}
}

func TestGetItemEndpoint(t *testing.T) {
	t.Parallel()

	contentString := `{"Location":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "something_random") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "error") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		resp, err := GetItemEndpoint[testAPIResource](c, "something_random")
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}

	{
		resp, err := GetItemEndpoint[testAPIResource](c, "error")
		assert.Equal(t, "status: 400, body: error", err.Error())
		assert.Nil(t, resp)
	}
}

func TestCreateItem(t *testing.T) {
	t.Parallel()

	contentString := `{"Location":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "test-resource/the_barricade") {
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "test-resource/the_sewers") {
				w.WriteHeader(http.StatusNoContent)
				_, err := w.Write([]byte(""))
				assert.Nil(t, err)
			} else if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "test-resource/error") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		test := testAPIResource{nil, "the_barricade"}
		resp, err := CreateItem(c, test)
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}

	{
		test := testAPIResource{nil, "the_sewers"}
		resp, err := CreateItem(c, test)
		assert.Nil(t, err)
		assert.Nil(t, resp)
	}

	{
		test := testAPIResource{nil, "error"}
		resp, err := CreateItem(c, test)
		assert.Equal(t, "status: 400, body: error", err.Error())
		assert.Nil(t, resp)
	}
}

func TestUpdateItem(t *testing.T) {
	t.Parallel()

	contentString := `{"Location":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "test-resource/the_barricade/1") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "test-resource/error/2") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		id := 1
		test := testAPIResource{&id, "the_barricade"}
		resp, err := UpdateItem(c, test)
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}

	{
		id := 2
		test := testAPIResource{&id, "error"}
		resp, err := UpdateItem(c, test)
		assert.Equal(t, "status: 400, body: error", err.Error())
		assert.Nil(t, resp)
	}
}

func TestUpdateItemEndpoint(t *testing.T) {
	t.Parallel()

	contentString := `{"Location":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "update") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "error") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		test := testAPIResource{nil, "the_barricade"}
		resp, err := UpdateItemEndpoint(c, test, "update")
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Location)
	}

	{
		test := testAPIResource{nil, "error"}
		resp, err := UpdateItemEndpoint(c, test, "error")
		assert.Equal(t, "status: 400, body: error", err.Error())
		assert.Nil(t, resp)
	}
}

func TestDeleteItem(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "test-resource//1") {
				w.WriteHeader(http.StatusNoContent)
				_, err := w.Write([]byte(""))
				assert.Nil(t, err)
			} else if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "test-resource/") {
				w.WriteHeader(http.StatusNoContent)
				_, err := w.Write([]byte(""))
				assert.Nil(t, err)
			} else if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "test-resource//2") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		id := 1
		err := DeleteItem[testAPIResource](c, &id)
		assert.Nil(t, err)
	}

	{
		err := DeleteItem[testAPIResource](c, nil)
		assert.Nil(t, err)
	}

	{
		id := 2
		err := DeleteItem[testAPIResource](c, &id)
		assert.Equal(t, "status: 400, body: error", err.Error())
	}
}

func TestDeleteItemEndpoint(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			assert.Nil(t, err)
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "delete") {
				w.WriteHeader(http.StatusNoContent)
				_, err := w.Write([]byte(""))
				assert.Nil(t, err)
			} else if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "error") {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("error"))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request", r.URL)
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	c.SetTestLogger(t)
	assert.Nil(t, err)

	{
		err := DeleteItemEndpoint(c, "delete")
		assert.Nil(t, err)
	}

	{
		err := DeleteItemEndpoint(c, "error")
		assert.Equal(t, "status: 400, body: error", err.Error())
	}
}

// IsNotFound decides whether a resource gets dropped from Terraform state, so a
// false positive silently discards a record of something that still exists.
//
// It used to be strings.Contains(err.Error(), "status: 404") against a message
// that interpolates the response body, so any error whose body mentioned that
// text answered yes. The appliance returns JSON bodies it does not promise the
// shape of, and echoes request content in some error paths, so this is reachable
// rather than theoretical.
func TestIsNotFoundReadsTheStatusNotTheBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"a real 404", &StatusError{Status: http.StatusNotFound, Body: `{"message":"Not Found"}`}, true},
		{"a 500 whose body quotes a 404", &StatusError{Status: http.StatusInternalServerError,
			Body: `{"message":"upstream said status: 404, body: gone"}`}, false},
		{"a 422 echoing a submitted field", &StatusError{Status: http.StatusUnprocessableEntity,
			Body: `{"errors":{"name":["status: 404 is not a valid name"]}}`}, false},
		{"a wrapped 404", fmt.Errorf("reading item: %w",
			&StatusError{Status: http.StatusNotFound, Body: ""}), true},
		{"a transport error", errors.New("dial tcp: connection refused"), false},
		{"no error at all", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsNotFound(tc.err))
		})
	}
}

// The message is part of the interface: it reaches operators through provider
// diagnostics, and both the 401 recovery tests and the CRUD error tests assert
// on its exact text.
func TestStatusErrorKeepsTheEstablishedMessage(t *testing.T) {
	err := &StatusError{Status: http.StatusTeapot, Body: "short and stout"}
	assert.Equal(t, "status: 418, body: short and stout", err.Error(),
		"the format must stay identical to what fmt.Errorf produced before")
}

// HasStatus is what lets a caller branch on a code other than 404 — the vault
// secret data source tolerates a 422 from check-in this way.
func TestHasStatusMatchesOnlyTheGivenCode(t *testing.T) {
	err := &StatusError{Status: http.StatusUnprocessableEntity, Body: "cannot check in"}
	assert.True(t, HasStatus(err, http.StatusUnprocessableEntity))
	assert.False(t, HasStatus(err, http.StatusNotFound))
	assert.False(t, HasStatus(errors.New("status: 422, body: not a StatusError"), http.StatusUnprocessableEntity),
		"a lookalike string must not satisfy a typed check")
}

// --- ListAllItems -----------------------------------------------------------

type pagedResource struct {
	ID   *int   `json:"id,omitempty"`
	Name string `json:"name"`
}

func (pagedResource) Endpoint() string { return "paged" }

// memberRow stands in for the group policy membership types: no ID, and a zero
// value whose Endpoint() would dereference a nil group policy ID.
type memberRow struct {
	JumpGroupID *int `json:"jump_group_id"`
}

func (m memberRow) Endpoint() string { panic("membership endpoints are built by the caller") }

func memberKey(m *memberRow) string { return fmt.Sprint(*m.JumpGroupID) }

func pagedRows(n int) []apimock.Row {
	rows := make([]apimock.Row, n)
	for i := range rows {
		rows[i] = apimock.Row{"id": i + 1, "name": fmt.Sprintf("item %d", i+1)}
	}
	return rows
}

// mockClient is a client of srv that retries an unstable list without pausing.
func mockClient(srv *apimock.Server) *APIClient {
	return &APIClient{RootURL: srv.URL, BaseURL: srv.URL + apimock.BasePath, HTTPClient: srv.Client(), ListRetryBackoff: []time.Duration{0, 0}}
}

func ids(items []pagedResource) []int {
	out := make([]int, len(items))
	for i, it := range items {
		out[i] = *it.ID
	}
	return out
}

func seq(from, to int) []int {
	out := []int{}
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

func TestListAllItemsReadsEveryPage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		rows, calls int
	}{
		{0, 1}, {99, 1}, {100, 1}, {101, 2}, {200, 2}, {250, 3},
	} {
		t.Run(fmt.Sprint(tc.rows), func(t *testing.T) {
			t.Parallel()
			srv := apimock.New()
			defer srv.Close()
			srv.Rows("paged", pagedRows(tc.rows)...)

			items, err := ListAllItems[pagedResource](mockClient(srv), nil)
			require.NoError(t, err)
			assert.Equal(t, seq(1, tc.rows), ids(items), "every row, once, in order")
			assert.Equal(t, tc.calls, srv.Calls(http.MethodGet, "paged"),
				"bounded by Last-Page; never asks past the end, where the appliance would repeat the last page")
		})
	}
}

func TestListAllItemsSendsFiltersWithPaging(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("paged", pagedRows(150)...)

	_, err := ListAllItems[pagedResource](mockClient(srv), map[string]string{"name": "item 7"})
	require.NoError(t, err)
	assert.Equal(t, []string{"GET paged?current_page=1&name=item+7&per_page=100"}, srv.Requests())
}

// The appliance clamps a page past the end to the last page. Without the paging
// headers there is no Last-Page to stop at, so exactly 100 rows must not loop:
// page 2 repeats page 1, adds nothing, and ends the pass.
func TestListAllItemsWithoutHeadersStopsOnRepeatOrShortPage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		rows, calls int
	}{
		{100, 2}, {150, 2}, {250, 3},
	} {
		t.Run(fmt.Sprint(tc.rows), func(t *testing.T) {
			t.Parallel()
			srv := apimock.New()
			defer srv.Close()
			srv.Set("paged", &apimock.Collection{Rows: pagedRows(tc.rows), NoHeaders: true})

			items, err := ListAllItems[pagedResource](mockClient(srv), nil)
			require.NoError(t, err)
			assert.Equal(t, seq(1, tc.rows), ids(items))
			assert.Equal(t, tc.calls, srv.Calls(http.MethodGet, "paged"))
		})
	}
}

// Without headers, overlapping pages dedup to the first occurrence and keep the
// order rows were first read in, so items[0] is what an unpaged GET returned.
func TestListAllItemsDedupKeepsFirstOccurrenceInOrder(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	// Page p returns ids (p-1)*50+1 .. (p-1)*50+100, capped at 150: every page
	// half-overlaps the previous one.
	srv.Handle(http.MethodGet, "paged", func(w http.ResponseWriter, r *http.Request) {
		p, _ := strconv.Atoi(r.URL.Query().Get("current_page"))
		rows := []apimock.Row{}
		for id := (p-1)*50 + 1; id <= min((p-1)*50+100, 150); id++ {
			rows = append(rows, apimock.Row{"id": id})
		}
		_ = json.NewEncoder(w).Encode(rows)
	})

	items, err := ListAllItems[pagedResource](mockClient(srv), nil)
	require.NoError(t, err)
	assert.Equal(t, seq(1, 150), ids(items))
	assert.Equal(t, 3, srv.Calls(http.MethodGet, "paged"))
}

// Membership rows have no id. Keyed on the member's own id they stay distinct;
// keyed on a missing id they would collapse into one.
func TestListAllItemsEndpointUsesTheGivenKey(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("group-policy/7/jump-group",
		apimock.Row{"jump_group_id": 3}, apimock.Row{"jump_group_id": 1}, apimock.Row{"jump_group_id": 2})

	items, err := ListAllItemsEndpoint(mockClient(srv), "group-policy/7/jump-group", nil, memberKey)
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, 3, *items[0].JumpGroupID)
}

func TestListAllItemsRequiresAnIDField(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()

	_, err := ListAllItems[memberRow](mockClient(srv), nil)
	require.ErrorContains(t, err, "use ListAllItemsEndpoint")
	assert.Empty(t, srv.Requests())
}

// The 201-row case: a delete between page 1 and page 2 shifts every later row
// back one place, so OFFSET 100 skips the row that moved onto page 1. Total and
// Last-Page change, and the pass is thrown away rather than returned short.
func TestListAllItemsRestartsWhenAPageReportsADifferentTotal(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		mutate func(c *apimock.Collection)
		calls  int
	}{
		// The unstable pass stops at page 2; the stable one reads 200 or 202 rows.
		{"delete", func(c *apimock.Collection) { c.Rows = c.Rows[1:] }, 2 + 2},
		{"growth", func(c *apimock.Collection) { c.Rows = append(c.Rows, apimock.Row{"id": 1000}) }, 2 + 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := apimock.New()
			defer srv.Close()
			mutated := false
			srv.Set("paged", &apimock.Collection{
				Rows: pagedRows(201),
				OnPage: func(c *apimock.Collection, page int) {
					if page == 2 && !mutated {
						mutated = true
						tc.mutate(c)
					}
				},
			})

			items, err := ListAllItems[pagedResource](mockClient(srv), nil)
			require.NoError(t, err)
			assert.Len(t, items, len(srv.Collection("paged").Rows), "the stable second pass, complete")
			assert.Equal(t, tc.calls, srv.Calls(http.MethodGet, "paged"))
		})
	}
}

func TestListAllItemsGivesUpAfterThreeUnstablePasses(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Set("paged", &apimock.Collection{
		Rows: pagedRows(201),
		OnPage: func(c *apimock.Collection, page int) {
			if page == 2 {
				c.Rows = c.Rows[1:]
			}
		},
	})

	items, err := ListAllItems[pagedResource](mockClient(srv), nil)
	assert.Nil(t, items, "never a partial list")
	var unstable *ErrUnstableList
	require.ErrorAs(t, err, &unstable)
	assert.Equal(t, ListChanged, unstable.Cause)
	assert.Equal(t, 3, unstable.Attempts)
	assert.Equal(t, 6, srv.Calls(http.MethodGet, "paged"), "three passes of two pages")
	assert.Contains(t, unstable.Describe("vault_account"),
		"The list of vault_account objects changed while it was being read, in 3 consecutive attempts")
	assert.Contains(t, unstable.Describe("vault_account"), "add filters")
}

// Jump groups sort on a non-unique name, and OFFSET can split ties differently
// per page: a row reappears on page 2 and another is never read. The headers
// never change, so only the unique count against Total gives it away.
func TestListAllItemsEndpointReportsOverlappingPages(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	rows := make([]apimock.Row, 200)
	for i := range rows {
		rows[i] = apimock.Row{"jump_group_id": i + 1}
	}
	srv.Set("group-policy/7/jump-group", &apimock.Collection{
		Rows: rows,
		OnPage: func(c *apimock.Collection, page int) {
			// Page 1 ends with row 100; page 2 sees the tie the other way round
			// and starts with it again.
			switch page {
			case 1:
				c.Rows[99], c.Rows[100] = apimock.Row{"jump_group_id": 100}, apimock.Row{"jump_group_id": 101}
			case 2:
				c.Rows[99], c.Rows[100] = apimock.Row{"jump_group_id": 101}, apimock.Row{"jump_group_id": 100}
			}
		},
	})

	_, err := ListAllItemsEndpoint(mockClient(srv), "group-policy/7/jump-group", nil, memberKey)
	var unstable *ErrUnstableList
	require.ErrorAs(t, err, &unstable)
	assert.Equal(t, ListOverlapped, unstable.Cause)
	assert.Equal(t, 3, unstable.Attempts)
	assert.Equal(t, 199, unstable.Got)
	assert.Equal(t, 200, unstable.Total)
	assert.Contains(t, unstable.Describe("jump_group"), "overlapping pages for jump_group")
}

func TestListAllItemsStopsAtThePageCap(t *testing.T) {
	t.Parallel()

	t.Run("header", func(t *testing.T) {
		t.Parallel()
		srv := apimock.New()
		defer srv.Close()
		srv.Handle(http.MethodGet, "paged", func(w http.ResponseWriter, r *http.Request) {
			apimock.WritePageClaiming(w, pagedRows(100), 100001, 1001)
		})

		_, err := ListAllItems[pagedResource](mockClient(srv), nil)
		var unstable *ErrUnstableList
		require.ErrorAs(t, err, &unstable)
		assert.Equal(t, ListPageCap, unstable.Cause)
		assert.Equal(t, 1, srv.Calls(http.MethodGet, "paged"), "known from page 1; not read, not retried")
		assert.Contains(t, unstable.Error(), "1000-page limit")
	})

	t.Run("no header", func(t *testing.T) {
		t.Parallel()
		srv := apimock.New()
		defer srv.Close()
		// An endpoint that never runs out of new rows.
		srv.Handle(http.MethodGet, "paged", func(w http.ResponseWriter, r *http.Request) {
			p, _ := strconv.Atoi(r.URL.Query().Get("current_page"))
			rows := make([]apimock.Row, 100)
			for i := range rows {
				rows[i] = apimock.Row{"id": p*100 + i}
			}
			_ = json.NewEncoder(w).Encode(rows)
		})

		_, err := ListAllItems[pagedResource](mockClient(srv), nil)
		var unstable *ErrUnstableList
		require.ErrorAs(t, err, &unstable)
		assert.Equal(t, ListPageCap, unstable.Cause)
		assert.Equal(t, 1000, srv.Calls(http.MethodGet, "paged"))
	})
}

func TestListAllItemsPassesThroughAPIErrors(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Set("paged", &apimock.Collection{Status: http.StatusForbidden})

	_, err := ListAllItems[pagedResource](mockClient(srv), nil)
	assert.True(t, HasStatus(err, http.StatusForbidden))
	assert.Equal(t, 1, srv.Calls(http.MethodGet, "paged"), "an API error is not an unstable list; no retry")
}
