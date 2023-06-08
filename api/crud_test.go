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
	Suffix string
}

func (t testAPIResource) Endpoint() string {
	return fmt.Sprintf("test-resource/%s", t.Suffix)
}

func TestGet(t *testing.T) {
	t.Parallel()
	// errorString := `{"error":"Nope"}`
	contentString := `{"Suffix":"the_sewers"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "oauth2/token") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"secret_access_granted"}`))
		} else {
			assert.Equal(t, "SRA-Terraform-Plugin", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")

			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "test-resource/") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(contentString))
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
		resp, err := Get[testAPIResource](c)
		assert.Nil(t, err)
		assert.Equal(t, "the_sewers", resp.Suffix)
	}

}
