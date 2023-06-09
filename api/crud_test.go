package api

import (
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
