# Create and manage a new Username/Password account in Vault

variable "account_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "sra_vault_username_password_account" "new_account" {
  name                = "Test User/Pass Account"
  username            = "test"
  password_wo         = var.account_password
  password_wo_version = 1

  # Omit the following configuration to use account group settings
  group_policy_memberships = [
    { group_policy_id : "123", role : "inject" }
  ]

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      shared_jump_groups = [2, 3]
      tag                = ["tftest"]
    }
    jump_items = [
      { id : 123, type : "shell_jump" }
    ]
  }
}
