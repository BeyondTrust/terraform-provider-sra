package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2"
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

	// oauth is retained so that a 401 can be answered by minting a fresh token
	// rather than surfacing as a hard error mid-apply. See doRequest.
	//
	// A zero value means "this client was built by hand, without credentials" —
	// the case for every APIClient literal in the tests. Re-authentication is then
	// skipped and a 401 is returned as-is, which is the pre-existing behaviour.
	oauth clientcredentials.Config

	// authMu guards HTTPClient and authGen. authGen increments on every successful
	// re-authentication, so concurrent callers that all received a 401 against the
	// same token refresh once between them rather than once each.
	authMu  sync.Mutex
	authGen uint64
}

// httpClient returns the client to send with, and the auth generation it belongs
// to. The generation is what a caller passes back to reauthenticate to say "the
// token I used was this one".
func (c *APIClient) httpClient() (*http.Client, uint64) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	return c.HTTPClient, c.authGen
}

// canReauthenticate reports whether this client holds the credentials needed to
// mint a new token.
func (c *APIClient) canReauthenticate() bool {
	return c.oauth.ClientID != "" && c.oauth.TokenURL != ""
}

// reauthenticate mints a fresh token and installs a client that uses it, unless
// another caller has already done so since the request that got the 401 went out.
//
// Returns whether the caller should replay. False means the credentials
// themselves are not working, so replaying would just produce the same 401 — and
// looping on that would turn a revoked credential into a hot loop against the
// appliance's token endpoint.
func (c *APIClient) reauthenticate(usedGen uint64) bool {
	c.authMu.Lock()
	defer c.authMu.Unlock()

	if c.authGen != usedGen {
		// Someone else refreshed after our request was sent. Their client is
		// current, so a replay is all that is needed.
		return true
	}

	ctx := context.Background()
	tok, err := c.oauth.Token(ctx)
	if err != nil {
		return false
	}

	// Seed the new source with the token just minted so this does not immediately
	// mint a second one.
	c.HTTPClient = oauth2.NewClient(ctx, oauth2.ReuseTokenSource(tok, c.oauth.TokenSource(ctx)))
	c.authGen++
	return true
}

// replayable returns a copy of req that can be sent again. The original's body
// has been consumed by the first attempt, so it cannot simply be resent.
func replayable(req *http.Request) (*http.Request, bool) {
	clone := req.Clone(req.Context())
	if req.Body == nil {
		return clone, true
	}
	if req.GetBody == nil {
		// net/http populates GetBody for the body types this package builds
		// (*strings.Reader). An opaque io.Reader has no rewind, so there is nothing
		// to replay and the 401 stands.
		return nil, false
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false
	}
	clone.Body = body
	return clone, true
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

	// Mint once and reuse. config.Token validates the credentials eagerly, which
	// is what turns a bad client_id into a clear diagnostic at provider
	// configuration rather than a confusing failure on first use; config.Client
	// would mint a second token lazily on the first request. Seeding the source
	// with the token already in hand keeps that validation and spends one token.
	tok, err := config.Token(ctx)
	if err != nil {
		return nil, err
	}

	c := APIClient{
		HTTPClient: oauth2.NewClient(ctx, oauth2.ReuseTokenSource(tok, config.TokenSource(ctx))),
		RootURL:    hostURL.String(),
		BaseURL:    hostURL.String() + "/api/config/v1",
		oauth:      config,
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

	body, status, usedGen, err := c.send(req)
	if err != nil {
		return nil, err
	}

	// A 401 here does not necessarily mean the credentials are wrong. The
	// appliance retains a bounded number of concurrent tokens per credential and
	// evicts the oldest, and oauth2's ReuseTokenSource only re-mints on expiry —
	// so a token can stop working while the client still considers it valid. Left
	// alone that surfaces as a bare "status: 401" mid-apply, after objects have
	// been created, with nothing in the message to act on.
	//
	// Retry exactly once. If the replay also 401s, the credentials really are not
	// working and the error stands; looping would turn a revoked credential into a
	// hot loop against the token endpoint.
	if status == http.StatusUnauthorized && c.canReauthenticate() {
		if replay, ok := replayable(req); ok && c.reauthenticate(usedGen) {
			c.LogString("🔑 doRequest [%s]: got 401, re-authenticated and replaying once", req.Method)
			body, status, _, err = c.send(replay)
			if err != nil {
				return nil, err
			}
		}
	}

	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, &StatusError{Status: status, Body: string(body)}
	}

	if status == http.StatusNoContent {
		return nil, nil
	}

	return body, nil
}

// send performs one round trip and reads the response. It returns the auth
// generation the request was sent under so the caller can tell reauthenticate
// which token failed.
func (c *APIClient) send(req *http.Request) (body []byte, status int, usedGen uint64, err error) {
	client, gen := c.httpClient()

	res, err := client.Do(req)
	if err != nil {
		return nil, 0, gen, err
	}
	defer func() { _ = res.Body.Close() }()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, gen, err
	}

	return body, res.StatusCode, gen, nil
}
