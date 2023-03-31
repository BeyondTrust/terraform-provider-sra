# Create and manage a new Username/Password account in Vault

resource "sra_vault_account_policy" "new_account_policy" {
  name                        = "Test Account Policy"
  code_name                   = "account_policy_code_name"
  auto_rotate_credentials     = false
  allow_simultaneous_checkout = true
  scheduled_password_rotation = false
}
