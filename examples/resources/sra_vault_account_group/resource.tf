# Create and manage a new Account Group account in Vault

resource "sra_vault_account_group" "new_account_group" {
  name           = "Test Account Group"
  account_policy = "account_policy_code_name"

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
