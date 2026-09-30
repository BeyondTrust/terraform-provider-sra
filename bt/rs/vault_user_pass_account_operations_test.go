package rs

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func vaultPasswordTestModel(sch schema.Schema) models.VaultUsernamePasswordAccount {
	return models.VaultUsernamePasswordAccount{
		ID: types.StringNull(), Type: types.StringValue("username_password"),
		Name: types.StringValue("vault"), Description: types.StringValue(""),
		Personal: types.BoolNull(), OwnerUserID: types.Int64Null(),
		AccountGroupID: types.Int64Value(1), AccountPolicy: types.StringNull(),
		Username: types.StringValue("user"), Password: types.StringNull(),
		PasswordWO: types.StringNull(), PasswordWOVersion: types.Int64Value(1),
		LastCheckoutTimestamp:  types.StringNull(),
		JumpItemAssociation:    types.ObjectNull(sch.GetAttributes()["jump_item_association"].GetType().(types.ObjectType).AttrTypes),
		GroupPolicyMemberships: types.SetNull(sch.GetAttributes()["group_policy_memberships"].GetType().(types.SetType).ElemType),
	}
}

func vaultPasswordTestPlan(t *testing.T, sch schema.Schema, model models.VaultUsernamePasswordAccount) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: sch}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return plan
}

func TestVaultPasswordCreateAndUpdateRequests(t *testing.T) {
	for _, tt := range []struct {
		name, method, wantPassword string
		versionChanged, legacy     bool
	}{
		{"write-only create", http.MethodPost, "ephemeral-secret", false, false},
		{"legacy create", http.MethodPost, "legacy-secret", false, true},
		{"unchanged version update", http.MethodPatch, "", false, false},
		{"changed version update", http.MethodPatch, "ephemeral-secret", true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var requestBody map[string]json.RawMessage
			client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
				if r.Method != tt.method || (r.URL.Path != "/api/config/v1/vault/account" && r.URL.Path != "/api/config/v1/vault/account/42") {
					return false
				}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Errorf("decode request: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":42,"type":"username_password","name":"vault","description":"","account_group_id":1,"username":"user"}`))
				return true
			})
			managed := newVaultUsernamePasswordAccountResource().(*vaultUsernamePasswordAccountResource)
			managed.ApiClient = client
			var schemaResp resource.SchemaResponse
			managed.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			sch := schemaResp.Schema

			planModel := vaultPasswordTestModel(sch)
			configModel := planModel
			if tt.legacy {
				planModel.Password = types.StringValue("legacy-secret")
				configModel.Password = planModel.Password
				planModel.PasswordWOVersion = types.Int64Null()
				configModel.PasswordWOVersion = types.Int64Null()
			} else {
				configModel.PasswordWO = types.StringValue("ephemeral-secret")
			}
			if tt.method == http.MethodPatch {
				planModel.ID = types.StringValue("42")
				configModel.ID = planModel.ID
			}
			plan := vaultPasswordTestPlan(t, sch, planModel)
			configPlan := vaultPasswordTestPlan(t, sch, configModel)
			config := tfsdk.Config{Schema: sch, Raw: configPlan.Raw}

			if tt.method == http.MethodPost {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
				managed.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("create diagnostics: %v", resp.Diagnostics)
				}
				if !tt.legacy {
					assertVaultPasswordAbsentFromState(t, ctx, resp.State)
				}
			} else {
				stateModel := planModel
				if tt.versionChanged {
					planModel.PasswordWOVersion = types.Int64Value(2)
					configModel.PasswordWOVersion = planModel.PasswordWOVersion
					plan = vaultPasswordTestPlan(t, sch, planModel)
					configPlan = vaultPasswordTestPlan(t, sch, configModel)
					config.Raw = configPlan.Raw
				}
				state := tfsdk.State{Schema: sch}
				if diags := state.Set(ctx, &stateModel); diags.HasError() {
					t.Fatalf("set state: %v", diags)
				}
				resp := &resource.UpdateResponse{State: state}
				managed.Update(ctx, resource.UpdateRequest{Plan: plan, State: state, Config: config}, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("update diagnostics: %v", resp.Diagnostics)
				}
				assertVaultPasswordAbsentFromState(t, ctx, resp.State)
			}

			if requestBody == nil {
				t.Fatal("Vault account request was not sent")
			}
			var password string
			if raw, found := requestBody["password"]; found {
				if err := json.Unmarshal(raw, &password); err != nil {
					t.Fatal(err)
				}
			}
			if password != tt.wantPassword {
				t.Errorf("password sent = %q, want %q", password, tt.wantPassword)
			}
			for _, name := range []string{"personal", "owner_user_id", "last_checkout_timestamp"} {
				if tt.method == http.MethodPatch {
					if _, found := requestBody[name]; found {
						t.Errorf("update sent read-only field %s", name)
					}
				}
			}
		})
	}
}

func assertVaultPasswordAbsentFromState(t *testing.T, ctx context.Context, state tfsdk.State) {
	t.Helper()
	for _, name := range []string{"password", "password_wo"} {
		var value types.String
		if diags := state.GetAttribute(ctx, path.Root(name), &value); diags.HasError() {
			t.Fatalf("read %s from state: %v", name, diags)
		}
		if !value.IsNull() {
			t.Errorf("%s must be null in state", name)
		}
	}
}

func TestVaultPasswordCreate204ReturnsDiagnostic(t *testing.T) {
	ctx := context.Background()
	client := mockGPClient(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		return false
	})
	managed := newVaultUsernamePasswordAccountResource().(*vaultUsernamePasswordAccountResource)
	managed.ApiClient = client
	var schemaResp resource.SchemaResponse
	managed.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema
	planModel := vaultPasswordTestModel(sch)
	configModel := planModel
	configModel.PasswordWO = types.StringValue("ephemeral-secret")
	plan := vaultPasswordTestPlan(t, sch, planModel)
	configPlan := vaultPasswordTestPlan(t, sch, configModel)
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	managed.Create(ctx, resource.CreateRequest{Plan: plan, Config: tfsdk.Config{Schema: sch, Raw: configPlan.Raw}}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("204 create must return a diagnostic instead of panicking or writing incomplete state")
	}
}
