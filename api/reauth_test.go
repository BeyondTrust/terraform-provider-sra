package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The appliance retains a bounded number of concurrent tokens per credential and
// evicts the oldest. oauth2's ReuseTokenSource re-mints only on expiry, so an
// evicted token keeps being sent and the request 401s while the client still
// considers itself authenticated. These cover the single bounded retry that
// answers that, and the boundaries on it.

// newReauthServer returns a server that rejects any token other than the most
// recently issued one — the eviction behaviour, in miniature — along with counters
// for tokens minted and API calls served.
func newReauthServer(t *testing.T) (*httptest.Server, *atomic.Int64, *atomic.Int64) {
	t.Helper()

	var minted, calls atomic.Int64
	var mu sync.Mutex
	current := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/oauth2/token") {
			n := minted.Add(1)
			mu.Lock()
			current = fmt.Sprintf("token-%d", n)
			tok := current
			mu.Unlock()
			_, _ = fmt.Fprintf(w, `{"token_type":"Bearer","expires_in":3600,"access_token":%q}`, tok)
			return
		}

		calls.Add(1)
		mu.Lock()
		want := "Bearer " + current
		mu.Unlock()
		if r.Header.Get("Authorization") != want {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Access token is invalid"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	return srv, &minted, &calls
}

func TestNewClientMintsOneToken(t *testing.T) {
	srv, minted, _ := newReauthServer(t)

	id, secret := "id", "secret"
	_, err := NewClient(srv.URL, &id, &secret)
	require.NoError(t, err)

	assert.Equal(t, int64(1), minted.Load(),
		"construction should validate the credentials and reuse that token, not mint a second one")
}

// TestDoRequestRetriesOnceAfterTokenEviction is the behaviour this exists for: a
// token that stopped working is replaced and the request succeeds, rather than the
// caller seeing a bare 401.
func TestDoRequestRetriesOnceAfterTokenEviction(t *testing.T) {
	srv, minted, calls := newReauthServer(t)

	id, secret := "id", "secret"
	c, err := NewClient(srv.URL, &id, &secret)
	require.NoError(t, err)
	require.Equal(t, int64(1), minted.Load())

	// Evict the client's token by minting a newer one out of band, exactly as
	// another process authenticating with the same credentials would.
	evict(t, srv)
	require.Equal(t, int64(2), minted.Load())

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/config/v1/thing", strings.NewReader(`{"a":1}`))
	require.NoError(t, err)

	body, err := c.doRequest(req)
	require.NoError(t, err, "the request should succeed after re-authenticating")
	assert.JSONEq(t, `{"ok":true}`, string(body))
	assert.Equal(t, int64(2), calls.Load(), "exactly one retry: the 401 and the replay")
	assert.Equal(t, int64(3), minted.Load(), "one fresh token for the retry")
}

// TestDoRequestReplaysTheBody guards the part most likely to break silently: the
// first attempt consumes the body, so a naive replay sends an empty one and the
// appliance sees a different request.
func TestDoRequestReplaysTheBody(t *testing.T) {
	var seen []string
	var mu sync.Mutex
	var minted atomic.Int64
	current := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/oauth2/token") {
			n := minted.Add(1)
			mu.Lock()
			current = fmt.Sprintf("token-%d", n)
			tok := current
			mu.Unlock()
			_, _ = fmt.Fprintf(w, `{"token_type":"Bearer","expires_in":3600,"access_token":%q}`, tok)
			return
		}
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		mu.Lock()
		seen = append(seen, string(b))
		want := "Bearer " + current
		mu.Unlock()
		if r.Header.Get("Authorization") != want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	id, secret := "id", "secret"
	c, err := NewClient(srv.URL, &id, &secret)
	require.NoError(t, err)
	evict(t, srv)

	const payload = `{"username":"someone","account_group_id":7}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/config/v1/thing", strings.NewReader(payload))
	require.NoError(t, err)

	_, err = c.doRequest(req)
	require.NoError(t, err)

	require.Len(t, seen, 2, "the 401 and the replay")
	assert.Equal(t, payload, seen[0])
	assert.Equal(t, payload, seen[1], "the replay must send the same body, not an empty one")
}

// TestDoRequestDoesNotRetryForever is the bound. With credentials the appliance
// always rejects, the replay 401s too and the error stands — a revoked credential
// must not become a hot loop against the token endpoint.
func TestDoRequestDoesNotRetryForever(t *testing.T) {
	var minted, calls atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/oauth2/token") {
			minted.Add(1)
			_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"never-valid"}`))
			return
		}
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Access token is invalid"}`))
	}))
	defer srv.Close()

	id, secret := "id", "secret"
	c, err := NewClient(srv.URL, &id, &secret)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/config/v1/thing", nil)
	require.NoError(t, err)

	_, err = c.doRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status: 401", "the original error shape is preserved")
	assert.Equal(t, int64(2), calls.Load(), "one attempt and one replay, no more")
	assert.LessOrEqual(t, minted.Load(), int64(2), "at most one extra token minted")
}

// TestDoRequestWithoutCredentialsDoesNotRetry covers the hand-built clients the
// tests use: an APIClient literal has no oauth config, so a 401 is returned as-is.
// This is the pre-existing behaviour and must not change.
func TestDoRequestWithoutCredentialsDoesNotRetry(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &APIClient{RootURL: srv.URL, BaseURL: srv.URL, HTTPClient: srv.Client()}
	require.False(t, c.canReauthenticate())

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/thing", nil)
	require.NoError(t, err)

	_, err = c.doRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status: 401")
	assert.Equal(t, int64(1), calls.Load(), "no retry without credentials to retry with")
}

// TestConcurrentRequestsReauthenticateOnce covers the shape the provider actually
// uses: group policy membership operations fan out across goroutines sharing one
// client. If each 401 refreshed independently they would evict each other in turn.
func TestConcurrentRequestsReauthenticateOnce(t *testing.T) {
	srv, minted, _ := newReauthServer(t)

	id, secret := "id", "secret"
	c, err := NewClient(srv.URL, &id, &secret)
	require.NoError(t, err)
	evict(t, srv)

	before := minted.Load()

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, rerr := http.NewRequest(http.MethodGet, srv.URL+"/api/config/v1/thing", nil)
			if rerr != nil {
				errs[i] = rerr
				return
			}
			_, errs[i] = c.doRequest(req)
		}(i)
	}
	wg.Wait()

	for i, e := range errs {
		assert.NoError(t, e, "goroutine %d", i)
	}
	assert.LessOrEqual(t, minted.Load()-before, int64(2),
		"concurrent 401s should collapse into about one refresh, not one each")
}

// evict mints a token out of band, which the server then treats as the only valid
// one — standing in for another process authenticating with the same credentials.
func evict(t *testing.T, srv *httptest.Server) {
	t.Helper()

	res, err := srv.Client().Post(srv.URL+"/oauth2/token", "application/x-www-form-urlencoded",
		strings.NewReader("grant_type=client_credentials"))
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
}
