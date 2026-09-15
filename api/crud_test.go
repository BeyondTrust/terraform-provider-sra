package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		err := DeleteItemEndpoint[testAPIResource](c, "delete")
		assert.Nil(t, err)
	}

	{
		err := DeleteItemEndpoint[testAPIResource](c, "error")
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
	assert.Equal(t, "status: 418, body: short and stout", err.Error())
	assert.Equal(t, fmt.Sprintf("status: %d, body: %s", http.StatusTeapot, "short and stout"), err.Error(),
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
