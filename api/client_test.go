package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestNewClient(t *testing.T) {
	testClientID := "jean_valjean"
	testClientSecret := "24601"
	badTokenError := `{"error":"Nope"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			auth := r.Header.Get("Authorization")
			creds := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", testClientID, testClientSecret)))
			w.Header().Set("Content-Type", "application/json")
			if fmt.Sprintf("Basic %s", creds) == auth {
				w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
			} else {
				w.WriteHeader(http.StatusTeapot)
				w.Write([]byte(badTokenError))
			}
		}
	}))
	defer ts.Close()

	{
		c, err := NewClient(ts.URL, &testClientID, &testClientSecret)
		assert.Nil(t, err)
		assert.Equal(t, ts.URL, c.RootURL)
		assert.Equal(t, ts.URL+"/api/config/v1", c.BaseURL)
	}

	{
		wrongSecret := "🤬"
		c, err := NewClient(ts.URL, &testClientID, &wrongSecret)
		assert.Nil(t, c)
		assert.NotNil(t, err)
		oauthErr := err.(*oauth2.RetrieveError)
		assert.Equal(t, http.StatusTeapot, oauthErr.Response.StatusCode)
		assert.Equal(t, []byte(badTokenError), oauthErr.Body)
	}
}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	errorString := `{"error":"Nope"}`
	contentString := `{"content":24601}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(r.URL.Path, "error") {
				w.WriteHeader(http.StatusTeapot)
				_, err := w.Write([]byte(errorString))
				assert.Nil(t, err)
			} else if strings.HasSuffix(r.URL.Path, "no-content") {
				w.WriteHeader(http.StatusNoContent)
				_, err := w.Write([]byte(""))
				assert.Nil(t, err)
			} else if strings.HasSuffix(r.URL.Path, "content") {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(contentString))
				assert.Nil(t, err)
			} else {
				assert.Fail(t, "Bad request")
			}
		}
	}))
	defer ts.Close()

	clientID := "id"
	clientSecret := "🤐"
	c, err := NewClient(ts.URL, &clientID, &clientSecret)
	assert.Nil(t, err)

	{
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.RootURL, "error"), nil)
		assert.Nil(t, err)
		body, err := c.doRequest(req)
		assert.Nil(t, body)
		assert.Equal(t, fmt.Sprintf("status: %d, body: %s", http.StatusTeapot, errorString), err.Error())
	}

	{
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.RootURL, "no-content"), nil)
		assert.Nil(t, err)
		body, err := c.doRequest(req)
		assert.Nil(t, body)
		assert.Nil(t, err)
	}

	{
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", c.RootURL, "content"), nil)
		assert.Nil(t, err)
		body, err := c.doRequest(req)
		assert.Nil(t, err)
		assert.Equal(t, []byte(contentString), body)
	}
}
