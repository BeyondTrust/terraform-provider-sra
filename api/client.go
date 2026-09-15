package api

import (
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
	Product    string
	HTTPClient *http.Client
	testLogger Logger
	logCtx     *context.Context
	mu         sync.Mutex
}

func (c *APIClient) IsRS() bool {
	return c.Product == ProductRS
}

func (c *APIClient) IsPRA() bool {
	return c.Product == ProductPRA
}

func (c *APIClient) ProductName() string {
	return c.Product
}

func (c *APIClient) SetTestLogger(l Logger) {
	if l == nil {
		return
	}
	if c == nil {
		// Receiver is nil (caller may have passed a nil client). Log to the test
		// so the caller still sees the context but avoid a panic.
		l.Logf("🧪 Set testing context for APIClient (nil receiver)")
		return
	}
	c.testLogger = l
	l.Logf("🧪 Set testing context for APIClient")
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
	if c.testLogger != nil {
		c.testLogger.Logf(format, args...)
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

	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.testLogger != nil || c.logCtx != nil {
		// Shape, not content — see the logging policy in api/logging.go.
		var urlStr = "<nil>"
		if req.URL != nil {
			urlStr = req.URL.String()
		}
		switch {
		case req.Body == nil:
			c.LogString("➡️ doRequest [%s %s]: <empty body>", req.Method, urlStr)
		case req.ContentLength > 0:
			// http.NewRequest sets ContentLength for every body type this package
			// builds (nil or *strings.Reader), so the size is already known. Reading
			// the body here to measure it — and rewrapping it so the transport could
			// still send it — copied every request payload for the sake of one number.
			c.LogString("➡️ doRequest [%s %s]: %d byte body", req.Method, urlStr, req.ContentLength)
		default:
			// An opaque io.Reader body leaves ContentLength at 0. No such caller
			// exists today; say so rather than reporting a misleading zero.
			c.LogString("➡️ doRequest [%s %s]: <unknown size body>", req.Method, urlStr)
		}
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

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
