# Create and manage a new Username/Password account in Vault

resource "sra_vault_account_group" "new_account_group" {
  name          = "Test Account Group"
  account_group = "account_policy_code_name"
}
