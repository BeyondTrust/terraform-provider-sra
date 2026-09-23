package rs

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// private_key_public_cert is write-only in practice: the appliance rejects both
// "" and null on write, and never returns the field on a read. Verified against a
// live appliance — a create that omits it returns 201 and the subsequent GET has
// no such key at all, while sending either "" or null returns
// 422 "This value must not be empty".
//
// That combination is why both halves of the handling matter, and why fixing one
// without the other produced a different failure rather than none:
//
//   - The request must omit the field when there is no certificate. Sending the
//     empty string is what the original report hit.
//   - The refresh must leave the configured value alone. Without that, state came
//     back null while the plan said "", and the apply failed with "provider
//     produced inconsistent result after apply".
//   - The attribute must not be Computed. With it absent from configuration,
//     Computed left the planned value unknown through apply, which Terraform
//     rejects outright.

func TestPrivateKeyPublicCertOmittedFromRequestWhenNotSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		cert types.String
		want any // nil means "the key must be absent"
	}{
		{"empty string", types.StringValue(""), nil},
		{"null", types.StringNull(), nil},
		{"unknown", types.StringUnknown(), nil},
		{"a real certificate", types.StringValue("ssh-ed25519-cert-v01@openssh.com AAAA"), "ssh-ed25519-cert-v01@openssh.com AAAA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tf := models.VaultSSHAccount{
				Name:                 types.StringValue("n"),
				Username:             types.StringValue("u"),
				Type:                 types.StringValue("ssh"),
				PrivateKeyPublicCert: tc.cert,
			}
			var apiObj api.VaultSSHAccount
			api.CopyTFtoAPI(context.Background(), reflect.ValueOf(&tf).Elem(), reflect.ValueOf(&apiObj).Elem(), api.ProductPRA)

			blob, err := json.Marshal(apiObj)
			require.NoError(t, err)
			var body map[string]any
			require.NoError(t, json.Unmarshal(blob, &body))

			got, present := body["private_key_public_cert"]
			if tc.want == nil {
				assert.False(t, present,
					"the appliance rejects both \"\" and null, so the field must be omitted entirely: %s", blob)
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestPrivateKeyPublicCertSurvivesRefresh is the second half. The API response
// carries no certificate, so a refresh that copied it into state would overwrite
// the configured value with null and fail the apply as inconsistent.
func TestPrivateKeyPublicCertSurvivesRefresh(t *testing.T) {
	for _, configured := range []string{"", "ssh-ed25519-cert-v01@openssh.com AAAA"} {
		name := "configured as a certificate"
		if configured == "" {
			name = "configured as the empty string"
		}
		t.Run(name, func(t *testing.T) {
			// What the plan holds going into the refresh.
			tf := models.VaultSSHAccount{PrivateKeyPublicCert: types.StringValue(configured)}

			// What the appliance returns: no certificate field at all.
			id := 1
			apiObj := api.VaultSSHAccount{ID: &id, Name: "n", Username: "u", Type: "ssh"}
			require.Nil(t, apiObj.PrivateKeyPublicCert, "fixture assumption: the read carries no certificate")

			err := api.CopyAPItoTF(context.Background(),
				reflect.ValueOf(&apiObj).Elem(), reflect.ValueOf(&tf).Elem(),
				reflect.TypeOf(apiObj), api.ProductPRA)
			require.NoError(t, err)

			assert.Equal(t, configured, tf.PrivateKeyPublicCert.ValueString(),
				"the configured value must survive a refresh that returns nothing")
			assert.False(t, tf.PrivateKeyPublicCert.IsNull(),
				"a null here is what produced \"inconsistent result after apply\"")
		})
	}
}

// TestPrivateKeyPublicCertIsNotComputed pins the schema. Computed promises
// Terraform the provider will supply a value; this one never can, and the symptom
// is an apply that fails with "still indicated an unknown value" whenever the
// attribute is absent from configuration.
func TestPrivateKeyPublicCertIsNotComputed(t *testing.T) {
	r := &vaultSSHAccountResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	attr, ok := resp.Schema.Attributes["private_key_public_cert"]
	require.True(t, ok, "attribute missing from the schema")

	assert.True(t, attr.IsOptional(), "must stay settable")
	assert.False(t, attr.IsComputed(),
		"the API never returns this field, so Computed leaves the planned value unknown through apply")
}
