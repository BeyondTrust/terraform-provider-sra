# Manage example Jumpoint
resource "sra_jumpoint" "example" {
  name      = "Example Jumpoint"
  code_name = "example_jumpoint"
  platform  = "linux-x86"

  group_policy_memberships = [
    { group_policy_id : "123" }
  ]
}
