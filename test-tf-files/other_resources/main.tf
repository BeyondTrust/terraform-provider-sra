terraform {
  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

resource "sra_jump_group" "example" {
  name      = "Example Jump Group"
  code_name = "example_group"

  group_policy_memberships = [
    { group_policy_id : "5" }
  ]
}
# Manage example Jumpoint
resource "sra_jumpoint" "example" {
  name      = "Example Jumpoint"
  code_name = "example_jumpoint"
  platform  = "linux-x86"

  group_policy_memberships = [
    { group_policy_id : "5" }
  ]
}

output "resources" {
  value = {
    JumpGroup = resource.sra_jump_group.example
    Jumpoint  = resource.sra_jumpoint.example
  }
}
