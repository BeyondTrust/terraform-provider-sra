# Create and manage a new Username/Password account in Vault

resource "sra_vault_username_password_account" "new_account" {
  name     = "Test User/Pass Account"
  username = "test"
  password = "this-is-a-test-password-that-should-be-generated-somehow"

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      shared_jump_groups = [2, 3]
      tag                = ["tftest"]
    }
    jump_items = [
      { id : 123, type : "remote_rdp" }
    ]
  }
}
