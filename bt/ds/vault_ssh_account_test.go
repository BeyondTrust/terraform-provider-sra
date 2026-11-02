package ds

import (
	"context"
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

func readSSHAccount(t *testing.T, srv *apimock.Server, config map[string]tftypes.Value) vaultSSHAccountDataSourceModel {
	t.Helper()
	state, diags := readDataSource(t, newVaultSSHAccountDataSource(), mockClient(srv), config)
	require.False(t, diags.HasError(), "%v", diags)
	var model vaultSSHAccountDataSourceModel
	require.False(t, state.Get(context.Background(), &model).HasError())
	require.NotNil(t, model.Account)
	return model
}

// The single SSH account data source uses only the first match, so it reads one
// row per lookup: type=ssh first, and type=ssh_ca only when that finds nothing.
// Here an SSH account matches, so there is no ssh_ca lookup. It also sends
// include_personal (a bool filter, which used to be dropped).
func TestSingleSSHAccountReadsOnePageOfSSHAccounts(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("vault/account", sshAccountRows(150)...)

	model := readSSHAccount(t, srv, map[string]tftypes.Value{
		"include_personal": tftypes.NewValue(tftypes.Bool, true),
	})

	assert.Contains(t, srv.Requests(), "GET vault/account?include_personal=true&per_page=1&type=ssh")
	assert.Equal(t, 1, srv.Calls(http.MethodGet, "vault/account"), "one list call, with no current_page=2 and no ssh_ca fallback")
	assert.Equal(t, "1", model.Account.ID.ValueString())
	assert.Equal(t, "ssh", model.Account.Type.ValueString())
	assert.Equal(t, "root", model.Account.Username.ValueString())
}

// The appliance's type filter is exact, so an SSH CA account is not an "ssh"
// row. When no SSH account matches, the data source falls back to SSH CA
// accounts; any other type is never returned.
func TestSingleSSHAccountFallsBackToSSHCA(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("vault/account",
		apimock.Row{"id": 1, "type": "username_password", "name": "key", "username": "admin", "account_group_id": 1},
		apimock.Row{"id": 2, "type": "ssh_ca", "name": "key", "username": "ca", "account_group_id": 1},
	)

	model := readSSHAccount(t, srv, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "key"),
	})

	assert.Contains(t, srv.Requests(), "GET vault/account?name=key&per_page=1&type=ssh")
	assert.Contains(t, srv.Requests(), "GET vault/account?name=key&per_page=1&type=ssh_ca")
	assert.Equal(t, "2", model.Account.ID.ValueString())
	assert.Equal(t, "ssh_ca", model.Account.Type.ValueString())
	assert.Equal(t, "ca", model.Account.Username.ValueString())
}

// No match used to index an empty slice and panic the provider. A non-SSH
// account with the right name is still no match.
func TestSingleSSHAccountNoMatchIsAnError(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("vault/account",
		apimock.Row{"id": 1, "type": "opaque_token", "name": "missing", "username": "", "account_group_id": 1},
	)

	_, diags := readDataSource(t, newVaultSSHAccountDataSource(), mockClient(srv), map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "missing"),
	})
	require.True(t, diags.HasError())
	assert.Equal(t, "No SSH account matched", diags.Errors()[0].Summary())
	assert.Equal(t, 2, srv.Calls(http.MethodGet, "vault/account"), "ssh, then ssh_ca")
}
