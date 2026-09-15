package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// canary is a value that appears nowhere else. Every assertion below is "this
// string must not survive", so it has to be distinctive enough that a match is
// unambiguous and a miss is not hidden by an unrelated substring.
//
// It contains no character that json.Marshal escapes. That is load-bearing: a
// canary containing a backslash or a quote marshals to a different byte sequence,
// so a NotContains assertion against the raw Go string would pass even with the
// secret sitting in the output in escaped form. The escaped-quote case is covered
// separately below, by constructing the escape explicitly and looking for a tail
// marker that is itself JSON-stable.
const canary = "c4n4ry-Sup3rSecretValue-do-not-log"

// captureLogger records what the API client logs, standing in for the tflog sink
// that carries this output into `terraform apply`'s debug log.
type captureLogger struct{ lines []string }

func (c *captureLogger) Logf(format string, args ...any) {
	c.lines = append(c.lines, fmt.Sprintf(format, args...))
}
func (c *captureLogger) all() string { return strings.Join(c.lines, "\n") }

// newTestClient returns a client wired to a stub server and the logger capturing
// its output. Three tests here need exactly this pair.
func newTestClient(t *testing.T, body string) (*APIClient, *captureLogger) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	log := &captureLogger{}
	c := &APIClient{RootURL: srv.URL, BaseURL: srv.URL, HTTPClient: srv.Client()}
	c.SetTestLogger(log)
	return c, log
}

// redact applies every pattern, mirroring what tflog does with them.
func redact(s string) string {
	for _, re := range SensitiveValuePatterns() {
		s = re.ReplaceAllString(s, "***")
	}
	return s
}

// TestDoRequestDoesNotLogRequestBody is the regression guard for the request path.
//
// doRequest used to log the full outbound body. For the vault account resources
// that body is the credential the provider exists to protect, and this log lands
// in `TF_LOG=DEBUG` output, which is routinely attached to support tickets.
func TestDoRequestDoesNotLogRequestBody(t *testing.T) {
	c, log := newTestClient(t, `{"id":1}`)

	body, err := json.Marshal(VaultUsernamePasswordAccount{Username: "someone", Password: canary})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/vault/account", strings.NewReader(string(body)))
	require.NoError(t, err)

	_, err = c.doRequest(req)
	require.NoError(t, err)

	assert.NotContains(t, log.all(), canary,
		"the request body must not reach the log — it carries the account's password")
	assert.Contains(t, log.all(), "doRequest",
		"the request should still be logged; only its body is withheld")
}

// TestDoRequestStillLogsRequestShape pins the diagnostic value that was kept, so a
// later cleanup does not quietly reduce this to silence.
func TestDoRequestStillLogsRequestShape(t *testing.T) {
	c, log := newTestClient(t, `{}`)

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/vault/account", strings.NewReader(`{"a":1}`))
	require.NoError(t, err)
	_, err = c.doRequest(req)
	require.NoError(t, err)

	out := log.all()
	assert.Contains(t, out, http.MethodPost, "method is diagnostic and should be kept")
	assert.Contains(t, out, "/vault/account", "URL is diagnostic and should be kept")
	assert.Contains(t, out, "7 byte body", "body size is kept as a substitute for the body itself")
}

// TestCreateItemDoesNotLogPayload covers the second place the create payload was
// written out: CreateItem logged both the struct (%+v) and the marshalled body.
func TestCreateItemDoesNotLogPayload(t *testing.T) {
	c, log := newTestClient(t, `{"id":1,"username":"someone"}`)

	_, err := CreateItem(c, VaultUsernamePasswordAccount{Username: "someone", Password: canary})
	require.NoError(t, err)

	assert.NotContains(t, log.all(), canary,
		"neither the struct nor the marshalled payload may be logged")
}

// TestSensitiveValuePatternsRedactEveryCredentialField marshals each
// credential-bearing model with a canary in every secret field and asserts the
// patterns leave none behind.
//
// This is deliberately driven off the real structs rather than a hand-written list
// of key names: if a json tag is renamed, or a new credential field is added to one
// of these models, the canary survives and this fails.
func TestSensitiveValuePatternsRedactEveryCredentialField(t *testing.T) {
	pass := canary
	for _, tc := range []struct {
		name string
		item any
	}{
		{"username/password account", VaultUsernamePasswordAccount{Username: "someone", Password: canary}},
		{"ssh account", VaultSSHAccount{Username: "someone", PrivateKey: &pass, PrivateKeyPassphrase: &pass}},
		{"token account", VaultTokenAccount{Token: canary}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blob, err := json.Marshal(tc.item)
			require.NoError(t, err)
			require.Contains(t, string(blob), canary, "canary must be present before redaction, or this proves nothing")

			out := redact(string(blob))
			assert.NotContains(t, out, canary, "a credential field survived redaction: %s", out)
		})
	}
}

// TestSensitiveValuePatternsHandleEscapedQuotes covers the case that would silently
// truncate a match: a credential containing a quote. A naive `"[^"]*"` value pattern
// ends at the embedded quote and leaves the remainder of the secret in the output.
func TestSensitiveValuePatternsHandleEscapedQuotes(t *testing.T) {
	blob, err := json.Marshal(map[string]string{"password": `abc"def-` + canary})
	require.NoError(t, err)
	require.Contains(t, string(blob), `\"`, "the fixture must actually contain an escaped quote")

	out := redact(string(blob))
	assert.NotContains(t, out, canary, "the tail after an escaped quote must not survive: %s", out)
}

// TestSensitiveValuePatternsLeaveOtherFieldsAlone guards the other direction. An
// over-broad pattern that swallowed neighbouring fields would make debug logs
// useless and would be noticed late.
func TestSensitiveValuePatternsLeaveOtherFieldsAlone(t *testing.T) {
	blob, err := json.Marshal(map[string]any{
		"username":         "someone",
		"password":         canary,
		"account_group_id": 42,
	})
	require.NoError(t, err)

	out := redact(string(blob))
	assert.NotContains(t, out, canary)
	assert.Contains(t, out, `"username":"someone"`, "non-credential fields must remain readable")
	assert.Contains(t, out, `"account_group_id":42`, "non-credential fields must remain readable")
}

// TestSensitiveValuePatternsWorkThroughTflog checks the patterns survive the real
// tflog pipeline — that MaskLogRegexes accepts them and redacts both the message
// and structured fields.
//
// This covers the patterns, not the wiring: it builds its own context and applies
// the mask itself, so deleting the MaskLogRegexes call in bt/provider.go leaves
// this test green. TestConfigureWiresRedactionIntoTheAPIClient in package bt covers
// that, and does fail when the call is removed.
func TestSensitiveValuePatternsWorkThroughTflog(t *testing.T) {
	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)
	ctx = tflog.MaskLogRegexes(ctx, SensitiveValuePatterns()...)

	// A message a careless future log call might emit.
	blob, err := json.Marshal(VaultUsernamePasswordAccount{Username: "someone", Password: canary})
	require.NoError(t, err)
	tflog.Debug(ctx, "some future log call: "+string(blob))

	// And the structured-field form, which MaskLogRegexes also covers.
	tflog.Debug(ctx, "structured", map[string]any{"payload": string(blob)})

	assert.NotContains(t, buf.String(), canary,
		"the backstop must redact credentials in both messages and structured fields")
	assert.Contains(t, buf.String(), "someone",
		"it must not be so broad that ordinary fields disappear")
}

// TestSensitiveValuePatternsRedactGoStructRendering covers the shape the JSON
// patterns cannot match, and the shape most likely to come back: fmt rendering a
// struct. Three of the call sites removed on this branch were exactly this —
// `%+v` of the item and of the Terraform plan.
//
// Password and Token are plain strings so fmt prints their values; PrivateKey and
// PrivateKeyPassphrase are pointers, which fmt renders as an address, so they
// cannot leak this way. Both are asserted so the distinction is recorded.
func TestSensitiveValuePatternsRedactGoStructRendering(t *testing.T) {
	pass := canary

	for _, tc := range []struct {
		name  string
		item  any
		leaks bool
		why   string
	}{
		{
			"plain string field", VaultUsernamePasswordAccount{Username: "someone", Password: canary}, true,
			"Password is a plain string, so %+v prints the value",
		},
		{
			"token field", VaultTokenAccount{Token: canary}, true,
			"Token is a plain string too",
		},
		{
			"pointer fields", VaultSSHAccount{Username: "someone", PrivateKey: &pass, PrivateKeyPassphrase: &pass}, false,
			"fmt prints a pointer field as an address, so the value never appears",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rendered := fmt.Sprintf("%+v", tc.item)
			require.Equal(t, tc.leaks, strings.Contains(rendered, canary),
				"fixture assumption wrong (%s): %s", tc.why, rendered)

			out := redact(rendered)
			assert.NotContains(t, out, canary, "the struct rendering must be redacted: %s", out)
		})
	}
}

// TestSensitiveValuePatternsDoNotSwallowTheStructTail pins why the struct pattern
// stops at whitespace or a brace rather than using \S+. A greedy pattern would
// consume the closing brace and everything fmt renders after it.
func TestSensitiveValuePatternsDoNotSwallowTheStructTail(t *testing.T) {
	rendered := fmt.Sprintf("%+v", struct {
		Password string
		Keep     string
	}{canary, "keep-me"})

	out := redact(rendered)
	assert.NotContains(t, out, canary)
	assert.Contains(t, out, "keep-me", "a following field must survive redaction: %s", out)
}

// TestSensitiveGoFieldsExistOnTheModels stops the struct-form patterns drifting
// away from the structs they describe. A renamed field would leave a pattern
// matching nothing, silently, and no other test would notice — the redaction tests
// construct their own fixtures rather than enumerating fields.
func TestSensitiveGoFieldsExistOnTheModels(t *testing.T) {
	models := []any{
		VaultUsernamePasswordAccount{},
		VaultSSHAccount{},
		VaultTokenAccount{},
		VaultSecret{},
	}

	found := map[string]bool{}
	for _, m := range models {
		typ := reflect.TypeOf(m)
		for i := 0; i < typ.NumField(); i++ {
			found[typ.Field(i).Name] = true
		}
	}

	for _, field := range sensitiveGoFields {
		assert.True(t, found[field],
			"%q is in sensitiveGoFields but is not a field on any credential-bearing model — "+
				"the pattern for it now matches nothing", field)
	}
}
