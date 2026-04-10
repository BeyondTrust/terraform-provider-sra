package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformToSnakeCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		// Existing cases
		{"thisIsATestString123", "this_is_a_test_string123"},
		{"ThisOneHASCaps", "this_one_has_caps"},

		// Edge cases
		{"", ""},
		{"already_snake", "already_snake"},
		{"ShellJump", "shell_jump"},
		{"RemoteRDP", "remote_rdp"},
		{"RemoteVNC", "remote_vnc"},
		{"VaultSSHAccount", "vault_ssh_account"},
		{"PostgreSQLTunnelJump", "postgre_sql_tunnel_jump"},
		{"MySQLTunnelJump", "my_sql_tunnel_jump"},
		{"JumpClientInstaller", "jump_client_installer"},
		{"A", "a"},
		{"ID", "id"},
		{"VaultAccountPolicy", "vault_account_policy"},
		{"NetworkTunnelJump", "network_tunnel_jump"},
		{"ProtocolTunnelJump", "protocol_tunnel_jump"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
