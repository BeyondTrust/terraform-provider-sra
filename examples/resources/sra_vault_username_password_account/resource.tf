# Create and manage a new Username/Password account in Vault

resource "sra_vault_username_password_account" "new_account" {
  name     = "Test SSH Account"
  username = "test"
  password = "this-is-a-test-password-that-should-be-generated-somehow"
}
