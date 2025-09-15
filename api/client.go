package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2/clientcredentials"
)

type APIClient struct {
	RootURL    string
	BaseURL    string
	HTTPClient *http.Client
	t          *testing.T
	logCtx     *context.Context
	mu         sync.Mutex
}

func (c *APIClient) SetTest(t *testing.T) {
	if t == nil {
		return
	}
	if c == nil {
		// Receiver is nil (caller may have passed a nil client). Log to the test
		// so the caller still sees the context but avoid a panic.
		t.Logf("🧪 Set testing context for APIClient (nil receiver)")
		return
	}
	c.t = t
	t.Logf("🧪 Set testing context for APIClient")
}

func (c *APIClient) SetLogContext(ctx *context.Context) {
	if ctx == nil {
		return
	}
	if c == nil {
		// No client instance to store the context on; still log the action.
		tflog.Debug(*ctx, "Set logging context for APIClient (nil receiver)")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logCtx = ctx
	tflog.Debug(*c.logCtx, "Set logging context for APIClient")
}

func (c *APIClient) LogString(format string, args ...any) {
	if c.t != nil {
		c.t.Logf(format, args...)
	}
	if c.logCtx != nil {
		c.mu.Lock()
		tflog.Debug(*c.logCtx, fmt.Sprintf(format, args...))
		c.mu.Unlock()
	}
}

func NewClient(host string, client_id *string, client_secret *string) (*APIClient, error) {
	hostURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	if hostURL.Scheme == "" {
		hostURL.Scheme = "https"
	}

	config := clientcredentials.Config{
		ClientID:     *client_id,
		ClientSecret: *client_secret,
		TokenURL:     hostURL.String() + "/oauth2/token",
	}
	ctx := context.Background()
	c := APIClient{
		HTTPClient: config.Client(ctx),
		RootURL:    hostURL.String(),
		BaseURL:    hostURL.String() + "/api/config/v1",
	}

	_, err = config.Token(ctx)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *APIClient) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "SRA-Terraform-Plugin")
	req.Header.Set("Accept", "application/json")

	if c.t != nil || c.logCtx != nil {
		// DEBUG: print request body so tests can show the exact JSON payload sent to the API
		var urlStr string = "<nil>"
		if req.URL != nil {
			urlStr = req.URL.String()
		}
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				c.LogString("➡️ doRequest payload: <error reading body: %v>", err)
			} else {
				// restore the Body so it can be read by the HTTP client
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				c.LogString("➡️ doRequest payload [%s %s]: %s", req.Method, urlStr, string(bodyBytes))
			}
		} else {
			c.LogString("➡️ doRequest payload [%s %s]: <empty body>", req.Method, urlStr)
		}
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	return body, err
}
