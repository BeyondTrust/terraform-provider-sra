package rs

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- harness ------------------------------------------------------------------

// unknown marks an attribute as not yet known, as a reference to another
// resource's output is during plan, or any variable under terraform validate.
var unknown = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

func ptType(t *testing.T) tftypes.Object {
	t.Helper()
	return resourceSchema(t, &protocolTunnelJumpResource{}).Type().TerraformType(context.Background()).(tftypes.Object)
}

// ptValues builds a full attribute map: strings and ints become known values, a
// tftypes.Value is used as given, and every attribute not named is null.
func ptValues(t *testing.T, attrs map[string]any) map[string]tftypes.Value {
	t.Helper()
	typ := ptType(t)
	vals := map[string]tftypes.Value{}
	for name, attrType := range typ.AttributeTypes {
		vals[name] = tftypes.NewValue(attrType, nil)
	}
	for name, v := range attrs {
		_, ok := typ.AttributeTypes[name]
		require.True(t, ok, "no attribute %q", name)
		switch v := v.(type) {
		case string:
			vals[name] = tftypes.NewValue(tftypes.String, v)
		case int:
			vals[name] = tftypes.NewValue(tftypes.Number, v)
		case tftypes.Value:
			vals[name] = v
		default:
			t.Fatalf("unsupported value for %s: %T", name, v)
		}
	}
	return vals
}

func ptObject(t *testing.T, vals map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	return tftypes.NewValue(ptType(t), vals)
}

// validate runs the resource's ValidateConfig against a configuration.
func validate(t *testing.T, attrs map[string]any) diag.Diagnostics {
	t.Helper()
	resp := &resource.ValidateConfigResponse{}
	cfg := tfsdk.Config{Schema: resourceSchema(t, &protocolTunnelJumpResource{}), Raw: ptObject(t, ptValues(t, attrs))}
	(&protocolTunnelJumpResource{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	return resp.Diagnostics
}

// ptProvider serves only the protocol tunnel resource, so a plan can be driven
// through the real framework server: defaults, the unknown-marking of computed
// attributes, attribute plan modifiers and ModifyPlan, in the order Terraform
// runs them. The rs package can't import bt's provider without a cycle.
type ptProvider struct{}

func (ptProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sra"
}
func (ptProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}
func (ptProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (ptProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }
func (ptProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newProtocolTunnelJumpResource}
}

type ptPlanResult struct {
	planned         map[string]tftypes.Value
	plannedRaw      tftypes.Value
	requiresReplace []*tftypes.AttributePath
	diags           []*tfprotov6.Diagnostic
}

// plan asks the framework for a plan, as `terraform plan` would. prior is nil
// on create. The proposed new state is built the way Terraform core builds it:
// the configured value where there is one, otherwise the prior value for a
// computed attribute.
func plan(t *testing.T, config map[string]any, prior map[string]tftypes.Value) ptPlanResult {
	t.Helper()
	ctx := context.Background()
	typ := ptType(t)
	sch := resourceSchema(t, &protocolTunnelJumpResource{})
	cfg := ptValues(t, config)

	proposed := map[string]tftypes.Value{}
	for name, v := range cfg {
		proposed[name] = v
		if v.IsNull() && prior != nil && sch.Attributes[name].IsComputed() {
			proposed[name] = prior[name]
		}
	}
	priorRaw := tftypes.NewValue(typ, nil)
	if prior != nil {
		priorRaw = tftypes.NewValue(typ, prior)
	}

	dv := func(v tftypes.Value) *tfprotov6.DynamicValue {
		d, err := tfprotov6.NewDynamicValue(typ, v)
		require.NoError(t, err)
		return &d
	}
	server := providerserver.NewProtocol6(ptProvider{})()
	resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "sra_protocol_tunnel_jump",
		PriorState:       dv(priorRaw),
		ProposedNewState: dv(tftypes.NewValue(typ, proposed)),
		Config:           dv(tftypes.NewValue(typ, cfg)),
	})
	require.NoError(t, err)

	res := ptPlanResult{requiresReplace: resp.RequiresReplace, diags: resp.Diagnostics}
	if resp.PlannedState != nil {
		res.plannedRaw, err = resp.PlannedState.Unmarshal(typ)
		require.NoError(t, err)
		require.NoError(t, res.plannedRaw.As(&res.planned))
	}
	return res
}

func str(t *testing.T, v tftypes.Value) string {
	t.Helper()
	require.True(t, v.IsKnown(), "value is unknown")
	var s string
	require.NoError(t, v.As(&s))
	return s
}

func replaces(r ptPlanResult, attr string) bool {
	for _, p := range r.requiresReplace {
		if p.Equal(tftypes.NewAttributePath().WithAttributeName(attr)) {
			return true
		}
	}
	return false
}

// tcpConfig is the tcp item from the repository fixture at v1.4.0
// (test-tf-files/pra/jump_items/protocol_tunnel_jump): tunnel_type omitted.
func tcpConfig() map[string]any {
	return map[string]any{
		"name": "This is a Name", "hostname": "this.host", "jumpoint_id": 1, "jump_group_id": 2,
		"tag": "bits", "tunnel_definitions": "22;24;80;8080",
	}
}

// mssqlConfig is the mssql item from the same fixture.
func mssqlConfig() map[string]any {
	return map[string]any{
		"name": "This is a Name", "hostname": "this.host", "jumpoint_id": 1, "jump_group_id": 2,
		"tag": "bits", "tunnel_type": "mssql", "username": "bits",
	}
}

func k8sConfig() map[string]any {
	return map[string]any{
		"name": "k8s", "jumpoint_id": 1, "jump_group_id": 2, "tunnel_type": "k8s",
		"url": "https://k8s.example.com:6443", "ca_certificates": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
	}
}

func with(base map[string]any, overrides map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = v
		}
	}
	return out
}

// v140TCPState is a tcp item as provider v1.4.0 writes it to state.
func v140TCPState(t *testing.T) map[string]tftypes.Value {
	return ptValues(t, map[string]any{
		"id": "11", "name": "This is a Name", "hostname": "this.host", "jumpoint_id": 1, "jump_group_id": 2,
		"jump_group_type": "shared", "tag": "bits", "comments": "",
		"tunnel_listen_address": "127.0.0.1", "tunnel_definitions": "22;24;80;8080", "tunnel_type": "tcp",
		"username": "", "database": "", "url": "", "ca_certificates": "",
	})
}

// v140MSSQLState is an mssql item as provider v1.4.0 writes it to state.
func v140MSSQLState(t *testing.T) map[string]tftypes.Value {
	return ptValues(t, map[string]any{
		"id": "12", "name": "This is a Name", "hostname": "this.host", "jumpoint_id": 1, "jump_group_id": 2,
		"jump_group_type": "shared", "tag": "bits", "comments": "",
		"tunnel_listen_address": "", "tunnel_definitions": "", "tunnel_type": "mssql",
		"username": "bits", "database": "", "url": "", "ca_certificates": "",
	})
}

// --- schema validators -----------------------------------------------------------

// The OneOf that rejected "k8s" is the bug this resource shipped with.
func TestProtocolTunnelTypeValidator(t *testing.T) {
	sch := resourceSchema(t, &protocolTunnelJumpResource{})
	for value, ok := range map[string]bool{
		"tcp": true, "mssql": true, "k8s": true,
		"ip": false, "mysql": false, "psql": false, "mongodb": false, "oracle": false, "bogus": false,
	} {
		assert.Equal(t, ok, runStringValidators(t, sch, "tunnel_type", types.StringValue(value)), "%q", value)
	}
}

func TestProtocolTunnelUntrimmedValidator(t *testing.T) {
	sch := resourceSchema(t, &protocolTunnelJumpResource{})
	for _, attr := range []string{"url", "ca_certificates"} {
		assert.True(t, runStringValidators(t, sch, attr, types.StringValue("https://k8s.example.com")), attr)
		assert.True(t, runStringValidators(t, sch, attr, types.StringValue("a\nb")), "inner newlines are kept: %s", attr)
		assert.True(t, runStringValidators(t, sch, attr, types.StringValue("")), attr)
		assert.True(t, runStringValidators(t, sch, attr, types.StringUnknown()), attr)
		assert.False(t, runStringValidators(t, sch, attr, types.StringValue("-----END CERTIFICATE-----\n")), "trailing newline from file(): %s", attr)
		assert.False(t, runStringValidators(t, sch, attr, types.StringValue(" https://k8s.example.com")), attr)
		assert.False(t, runStringValidators(t, sch, attr, types.StringValue("\uFEFFhttps://k8s.example.com")), attr)
	}
}

// --- ValidateConfig ----------------------------------------------------------------

func TestProtocolTunnelValidateConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		errOn  string // "" means no diagnostics
	}{
		{"tcp by default, with definitions and hostname", tcpConfig(), ""},
		{"tcp by default, no hostname", with(tcpConfig(), map[string]any{"hostname": nil}), "hostname"},
		{"tcp, no hostname", with(tcpConfig(), map[string]any{"tunnel_type": "tcp", "hostname": nil}), "hostname"},
		{"tcp, no definitions", with(tcpConfig(), map[string]any{"tunnel_definitions": nil}), "tunnel_definitions"},
		{"tcp, odd definitions", with(tcpConfig(), map[string]any{"tunnel_definitions": "22;24;26"}), "tunnel_definitions"},
		{"tcp, port out of range", with(tcpConfig(), map[string]any{"tunnel_definitions": "70000;24"}), "tunnel_definitions"},
		{"tcp, empty listen address", with(tcpConfig(), map[string]any{"tunnel_listen_address": ""}), "tunnel_listen_address"},
		{"tcp, listen address outside 127.0.0.0/24", with(tcpConfig(), map[string]any{"tunnel_listen_address": "192.168.1.1"}), "tunnel_listen_address"},
		{"tcp, IPv4-mapped listen address", with(tcpConfig(), map[string]any{"tunnel_listen_address": "::ffff:127.0.0.1"}), "tunnel_listen_address"},
		{"tcp, non-default listen address", with(tcpConfig(), map[string]any{"tunnel_listen_address": "127.0.0.5"}), ""},
		{"tcp, with url", with(tcpConfig(), map[string]any{"url": "https://x"}), "url"},
		{"tcp, with username", with(tcpConfig(), map[string]any{"username": "u"}), "username"},
		{"tcp, empty username passes", with(tcpConfig(), map[string]any{"username": ""}), ""},

		{"mssql", mssqlConfig(), ""},
		{"mssql, no username", with(mssqlConfig(), map[string]any{"username": nil}), "username"},
		{"mssql, no hostname", with(mssqlConfig(), map[string]any{"hostname": nil}), "hostname"},
		{"mssql, with url", with(mssqlConfig(), map[string]any{"url": "https://x"}), "url"},
		{"mssql, empty tunnel_definitions passes", with(mssqlConfig(), map[string]any{"tunnel_definitions": ""}), ""},
		{"mssql, tunnel_definitions", with(mssqlConfig(), map[string]any{"tunnel_definitions": "22;24"}), "tunnel_definitions"},
		{"mssql, listen address", with(mssqlConfig(), map[string]any{"tunnel_listen_address": "127.0.0.1"}), "tunnel_listen_address"},
		{"mssql, empty listen address passes", with(mssqlConfig(), map[string]any{"tunnel_listen_address": ""}), ""},

		{"k8s, url and ca, no hostname", k8sConfig(), ""},
		{"k8s, hostname equal to url", with(k8sConfig(), map[string]any{"hostname": "https://k8s.example.com:6443"}), ""},
		{"k8s, different hostname", with(k8sConfig(), map[string]any{"hostname": "k8s.example.com"}), "hostname"},
		{"k8s, empty hostname", with(k8sConfig(), map[string]any{"hostname": ""}), "hostname"},
		{"k8s, empty hostname with url unknown", with(k8sConfig(), map[string]any{"hostname": "", "url": unknown}), "hostname"},
		{"k8s, no url", with(k8sConfig(), map[string]any{"url": nil}), "url"},
		{"k8s, no ca", with(k8sConfig(), map[string]any{"ca_certificates": ""}), "ca_certificates"},
		{"k8s, with definitions", with(k8sConfig(), map[string]any{"tunnel_definitions": "22;24"}), "tunnel_definitions"},
		{"k8s, with database", with(k8sConfig(), map[string]any{"database": "db"}), "database"},

		// Values not known yet pass, and are checked again once known.
		{"unknown name", with(tcpConfig(), map[string]any{"name": unknown}), ""},
		{"unknown mssql username", with(mssqlConfig(), map[string]any{"username": unknown}), ""},
		{"unknown tcp definitions", with(tcpConfig(), map[string]any{"tunnel_definitions": unknown}), ""},
		{"unknown hostname", with(tcpConfig(), map[string]any{"hostname": unknown}), ""},
		{"unknown k8s url", with(k8sConfig(), map[string]any{"url": unknown}), ""},
		{"unknown k8s url with a hostname", with(k8sConfig(), map[string]any{"url": unknown, "hostname": "h"}), ""},
		{"unknown tunnel_type skips type checks", with(k8sConfig(), map[string]any{"tunnel_type": unknown, "tunnel_definitions": "22;24"}), ""},
		{"unknown listen address", with(tcpConfig(), map[string]any{"tunnel_listen_address": unknown}), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := validate(t, tc.config)
			if tc.errOn == "" {
				assert.False(t, diags.HasError(), "%v", diags)
				return
			}
			require.True(t, diags.HasError(), "expected an error on %s", tc.errOn)
			for _, d := range diags.Errors() {
				withPath, ok := d.(diag.DiagnosticWithPath)
				require.True(t, ok, "%v has no attribute path", d)
				assert.Equal(t, tc.errOn, withPath.Path().String())
			}
		})
	}
}

// terraform validate runs with every variable unknown. A required check that
// treated unknown as missing would fail a valid module before it is ever used.
func TestProtocolTunnelValidateConfigAllUnknown(t *testing.T) {
	typ := ptType(t)
	attrs := map[string]any{}
	for name, attrType := range typ.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, tftypes.UnknownValue)
	}
	assert.False(t, validate(t, attrs).HasError())
}

// --- planning ----------------------------------------------------------------------

func TestProtocolTunnelPlanListenAddressDefaults(t *testing.T) {
	t.Run("tcp create, omitted", func(t *testing.T) {
		r := plan(t, tcpConfig(), nil)
		require.Empty(t, r.diags)
		assert.Equal(t, "127.0.0.1", str(t, r.planned["tunnel_listen_address"]))
		assert.Equal(t, "tcp", str(t, r.planned["tunnel_type"]))
	})
	t.Run("mssql create, omitted", func(t *testing.T) {
		r := plan(t, mssqlConfig(), nil)
		require.Empty(t, r.diags)
		assert.Equal(t, "", str(t, r.planned["tunnel_listen_address"]))
		assert.Equal(t, "", str(t, r.planned["tunnel_definitions"]))
	})
	t.Run("unknown in config stays unknown", func(t *testing.T) {
		r := plan(t, with(tcpConfig(), map[string]any{"tunnel_listen_address": unknown}), nil)
		require.Empty(t, r.diags)
		assert.False(t, r.planned["tunnel_listen_address"].IsKnown())
	})

	// An imported or console-set address is kept when the configuration omits
	// it, both on a plan with no other change and on one that changes tag.
	state := v140TCPState(t)
	state["tunnel_listen_address"] = tftypes.NewValue(tftypes.String, "127.0.0.5")
	t.Run("non-default state, no diff", func(t *testing.T) {
		r := plan(t, tcpConfig(), state)
		require.Empty(t, r.diags)
		assert.Equal(t, "127.0.0.5", str(t, r.planned["tunnel_listen_address"]))
		assert.True(t, r.plannedRaw.Equal(tftypes.NewValue(ptType(t), state)), "plan must equal state")
	})
	t.Run("non-default state, tag changed", func(t *testing.T) {
		r := plan(t, with(tcpConfig(), map[string]any{"tag": "changed"}), state)
		require.Empty(t, r.diags)
		assert.Equal(t, "127.0.0.5", str(t, r.planned["tunnel_listen_address"]))
		assert.Equal(t, "22;24;80;8080", str(t, r.planned["tunnel_definitions"]))
	})
}

func TestProtocolTunnelPlanK8sHostname(t *testing.T) {
	t.Run("derived from url", func(t *testing.T) {
		r := plan(t, k8sConfig(), nil)
		require.Empty(t, r.diags)
		assert.Equal(t, "https://k8s.example.com:6443", str(t, r.planned["hostname"]))
		assert.Equal(t, "", str(t, r.planned["tunnel_listen_address"]))
		assert.Equal(t, "", str(t, r.planned["tunnel_definitions"]))
	})
	t.Run("truncated to 128 bytes", func(t *testing.T) {
		long := "https://k8s.example.com/" + strings.Repeat("a", 200)
		r := plan(t, with(k8sConfig(), map[string]any{"url": long}), nil)
		require.Empty(t, r.diags)
		assert.Equal(t, long[:128], str(t, r.planned["hostname"]))
	})
	t.Run("url unknown", func(t *testing.T) {
		r := plan(t, with(k8sConfig(), map[string]any{"url": unknown}), nil)
		require.Empty(t, r.diags)
		assert.False(t, r.planned["hostname"].IsKnown())
	})
	t.Run("update with url unchanged plans no diff", func(t *testing.T) {
		state := ptValues(t, map[string]any{
			"id": "13", "name": "k8s", "hostname": "https://k8s.example.com:6443", "jumpoint_id": 1, "jump_group_id": 2,
			"jump_group_type": "shared", "tag": "", "comments": "",
			"tunnel_listen_address": "", "tunnel_definitions": "", "tunnel_type": "k8s",
			"username": "", "database": "", "url": "https://k8s.example.com:6443",
			"ca_certificates": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		})
		r := plan(t, k8sConfig(), state)
		require.Empty(t, r.diags)
		assert.True(t, r.plannedRaw.Equal(tftypes.NewValue(ptType(t), state)), "plan must equal state")

		r = plan(t, with(k8sConfig(), map[string]any{"tag": "changed"}), state)
		require.Empty(t, r.diags)
		assert.Equal(t, "https://k8s.example.com:6443", str(t, r.planned["hostname"]))
	})
}

func TestProtocolTunnelTypeChangeReplaces(t *testing.T) {
	state := v140TCPState(t)

	r := plan(t, k8sConfig(), state)
	require.Empty(t, r.diags)
	assert.True(t, replaces(r, "tunnel_type"), "tcp -> k8s must replace")
	assert.Equal(t, "", str(t, r.planned["tunnel_listen_address"]), "the new item is planned as k8s, not with the tcp state")

	r = plan(t, with(tcpConfig(), map[string]any{"tunnel_type": "tcp"}), state)
	require.Empty(t, r.diags)
	assert.False(t, replaces(r, "tunnel_type"), "tcp -> tcp")

	r = plan(t, with(tcpConfig(), map[string]any{"tunnel_type": unknown}), state)
	require.Empty(t, r.diags)
	assert.False(t, replaces(r, "tunnel_type"), "tcp -> unknown must not plan a destroy")
}

// State written by v1.4.0, against the v1.4.0 repository fixture, must re-plan
// with no diagnostics, no replacement, and a plan equal to state.
func TestProtocolTunnelV140StateReplansClean(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		state  map[string]tftypes.Value
	}{
		{"tcp", tcpConfig(), v140TCPState(t)},
		{"mssql", mssqlConfig(), v140MSSQLState(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, validate(t, tc.config).HasError())

			r := plan(t, tc.config, tc.state)
			require.Empty(t, r.diags)
			assert.Empty(t, r.requiresReplace)
			assert.True(t, r.plannedRaw.Equal(tftypes.NewValue(ptType(t), tc.state)), "plan must equal state:\n%v", r.plannedRaw)
		})
	}
}
