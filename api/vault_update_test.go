package api

import (
	"encoding/json"
	"testing"
)

func TestVaultAccountUpdatesOmitReadOnlyFields(t *testing.T) {
	personal := false
	owner := 42
	checkout := "2026-09-11T09:12:38Z"

	tests := []struct {
		name string
		item UpdateRequestPreparer
	}{
		{"username/password", &VaultUsernamePasswordAccount{Personal: &personal, OwnerUserID: &owner, LastCheckoutTimestamp: &checkout}},
		{"SSH", &VaultSSHAccount{Personal: &personal, OwnerUserID: &owner, LastCheckoutTimestamp: &checkout}},
		{"token", &VaultTokenAccount{Personal: &personal, OwnerUserID: &owner, LastCheckoutTimestamp: &checkout}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.item.PrepareUpdateRequest()
			payload, err := json.Marshal(tt.item)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(payload, &fields); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"personal", "owner_user_id", "last_checkout_timestamp"} {
				if _, found := fields[name]; found {
					t.Errorf("update payload includes read-only field %s: %s", name, payload)
				}
			}
		})
	}
}
