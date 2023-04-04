# Manage example Jump Group
resource "sra_jump_group" "example" {
  name      = "Example Jump Group"
  code_name = "example_group"

  group_policy_memberships = [
    { group_policy_id : "123", jump_item_role_id : 123, jump_policy_id : 123 }
  ]
}
