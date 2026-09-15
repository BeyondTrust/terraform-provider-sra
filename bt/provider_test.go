package bt

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigureWiresRedactionIntoTheAPIClient covers the wiring, not the patterns.
//
// api/logging_test.go proves the redaction patterns work when applied. It cannot
// prove they are actually installed, because it applies them to its own context —
// so deleting the MaskLogRegexes call in Configure left that whole suite green.
// This test fails when that call is removed.
//
// It needs no appliance: Configure makes exactly two requests, the OAuth handshake
// and the product-detection call, and both are served here. The masked context
// reaches the client because Configure hands it over via SetLogContext, which is
// what makes an assertion on the client's own output a test of the wiring.
func TestConfigureWiresRedactionIntoTheAPIClient(t *testing.T) {
	const canary = "c4n4ryValueThatMustNotBeLogged"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/token"):
			_, _ = w.Write([]byte(`{"token_type":"Bearer","expires_in":3600,"access_token":"stub"}`))
		default:
			// Product detection (get_mech_list). Anything else would show up as a
			// Configure diagnostic, which is asserted on below.
			_, _ = w.Write([]byte(`{"mechs":["local"],"default_mech":"local","company":"acme","product":"bpam"}`))
		}
	}))
	defer srv.Close()

	t.Setenv("BT_API_HOST", srv.URL)
	t.Setenv("BT_CLIENT_ID", "stub-id")
	t.Setenv("BT_CLIENT_SECRET", "stub-secret")

	ctx := context.Background()
	p := New()

	// An all-null config over the provider's real schema, so Configure falls
	// through to the environment variables set above.
	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	nulls := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		nulls[name] = tftypes.NewValue(attrType, nil)
	}
	cfg := tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nulls)}

	var buf bytes.Buffer
	var resp provider.ConfigureResponse
	p.Configure(tflogtest.RootLogger(ctx, &buf), provider.ConfigureRequest{Config: cfg}, &resp)

	require.False(t, resp.Diagnostics.HasError(), "Configure failed: %v", resp.Diagnostics)
	client, ok := resp.ResourceData.(*api.APIClient)
	require.True(t, ok, "Configure did not hand back an *api.APIClient")

	// The assertion: a credential logged through the context Configure captured
	// must come out redacted.
	client.LogString(`some future log call: {"username":"someone","password":%q}`, canary)

	out := buf.String()
	assert.NotContains(t, out, canary,
		"Configure did not install the redaction on the context it gave the API client: %s", out)
	assert.Contains(t, out, "someone",
		"the redaction must not be broad enough to remove ordinary fields: %s", out)
	assert.NotContains(t, out, "stub-secret",
		"the pre-existing client_secret field masking must still apply: %s", out)
}
