package rs

import (
	"context"
	"reflect"
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVaultUsernamePasswordAccountWriteOnlyPasswordSchema(t *testing.T) {
	t.Parallel()

	managed := &vaultUsernamePasswordAccountResource{}
	var resp resource.SchemaResponse
	managed.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned diagnostics: %v", resp.Diagnostics)
	}

	password := resp.Schema.Attributes["password"].(schema.StringAttribute)
	if !password.Optional || password.Required {
		t.Fatal("legacy password must remain optional for backwards compatibility")
	}

	passwordWO := resp.Schema.Attributes["password_wo"].(schema.StringAttribute)
	if !passwordWO.Optional || !passwordWO.Sensitive || !passwordWO.WriteOnly {
		t.Fatal("password_wo must be optional, sensitive, and write-only")
	}

	if got := len(managed.ConfigValidators(context.Background())); got != 3 {
		t.Fatalf("expected three password configuration validators, got %d", got)
	}

	modelType := reflect.TypeOf(models.VaultUsernamePasswordAccount{})
	modelAttributes := make(map[string]struct{}, modelType.NumField())
	for index := 0; index < modelType.NumField(); index++ {
		modelAttributes[modelType.Field(index).Tag.Get("tfsdk")] = struct{}{}
	}
	for _, attribute := range []string{"password", "password_wo", "password_wo_version"} {
		if _, ok := modelAttributes[attribute]; !ok {
			t.Fatalf("Terraform model does not contain %s", attribute)
		}
	}
}

func TestVaultAccountPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        models.VaultUsernamePasswordAccount
		plan          models.VaultUsernamePasswordAccount
		useWriteOnly  bool
		wantPassword  string
		wantOK        bool
		wantDiagError bool
	}{
		{
			name: "legacy password",
			plan: models.VaultUsernamePasswordAccount{
				Password: types.StringValue("legacy-secret"),
			},
			useWriteOnly: true,
			wantPassword: "legacy-secret",
			wantOK:       true,
		},
		{
			name: "write-only password",
			config: models.VaultUsernamePasswordAccount{
				PasswordWO: types.StringValue("ephemeral-secret"),
			},
			plan: models.VaultUsernamePasswordAccount{
				Password: types.StringNull(),
			},
			useWriteOnly: true,
			wantPassword: "ephemeral-secret",
			wantOK:       true,
		},
		{
			name: "unchanged write-only password",
			plan: models.VaultUsernamePasswordAccount{
				Password: types.StringNull(),
			},
			wantPassword: "",
			wantOK:       true,
		},
		{
			name: "missing write-only password",
			config: models.VaultUsernamePasswordAccount{
				PasswordWO: types.StringNull(),
			},
			plan: models.VaultUsernamePasswordAccount{
				Password: types.StringNull(),
			},
			useWriteOnly:  true,
			wantOK:        false,
			wantDiagError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var diagnostics diag.Diagnostics
			password, ok := vaultAccountPassword(test.config, test.plan, test.useWriteOnly, &diagnostics)
			if password != test.wantPassword || ok != test.wantOK {
				t.Fatalf("vaultAccountPassword() = (%q, %t), want (%q, %t)", password, ok, test.wantPassword, test.wantOK)
			}
			if diagnostics.HasError() != test.wantDiagError {
				t.Fatalf("diagnostics error = %t, want %t: %v", diagnostics.HasError(), test.wantDiagError, diagnostics)
			}
		})
	}
}
