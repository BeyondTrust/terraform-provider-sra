// Modified by Stegra AB for the Stegra-maintained distribution.
// SPDX-License-Identifier: Apache-2.0
package rs

import (
	"encoding/json"
	"strings"
	"terraform-provider-sra/api"
	"testing"
)

func TestVaultUsernamePasswordAccountUpdateOmitsReadOnlyFields(t *testing.T) {
	t.Parallel()

	personal := false
	ownerUserID := 42
	lastCheckoutTimestamp := "2026-09-11T09:12:38Z"
	item := api.VaultUsernamePasswordAccount{
		Personal:              &personal,
		OwnerUserID:           &ownerUserID,
		LastCheckoutTimestamp: &lastCheckoutTimestamp,
	}

	stripVaultUsernamePasswordAccountReadOnlyFields(&item)
	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal update payload: %v", err)
	}

	for _, field := range []string{"personal", "owner_user_id", "last_checkout_timestamp"} {
		if strings.Contains(string(payload), `"`+field+`"`) {
			t.Fatalf("update payload contains read-only field %q: %s", field, payload)
		}
	}
}
