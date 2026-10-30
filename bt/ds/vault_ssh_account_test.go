package ds

import (
	"net/http"
	"testing"

	"terraform-provider-sra/api/apimock"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sshAccountRows(n int) []apimock.Row {
	rows := make([]apimock.Row, n)
	for i := range rows {
		rows[i] = apimock.Row{"id": i + 1, "type": "ssh", "name": "key", "username": "root", "account_group_id": 1}
	}
	return rows
}

// The single SSH account data source uses only the first match, so it reads one
// page, asks for SSH accounts only, and sends include_personal (a bool filter,
// which used to be dropped).
func TestSingleSSHAccountReadsOnePageOfSSHAccounts(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("vault/account", sshAccountRows(150)...)

	state, diags := readDataSource(t, newVaultSSHAccountDataSource(), mockClient(srv), map[string]tftypes.Value{
		"include_personal": tftypes.NewValue(tftypes.Bool, true),
	})
	require.False(t, diags.HasError(), "%v", diags)
	assert.False(t, state.Raw.IsNull())

	assert.Equal(t, []string{
		"GET vault/account?include_personal=true&type=ssh",
		"GET vault/account/1",
	}, srv.Requests(), "one list call, with no current_page=2")
	assert.Equal(t, 1, srv.Calls(http.MethodGet, "vault/account"))
}

// No match used to index an empty slice and panic the provider.
func TestSingleSSHAccountNoMatchIsAnError(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("vault/account")

	_, diags := readDataSource(t, newVaultSSHAccountDataSource(), mockClient(srv), map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "missing"),
	})
	require.True(t, diags.HasError())
	assert.Equal(t, "No SSH account matched", diags.Errors()[0].Summary())
}
