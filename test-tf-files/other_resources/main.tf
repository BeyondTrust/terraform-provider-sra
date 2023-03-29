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
}

output "resources" {
  value = {
    JumpGroup = resource.sra_jump_group.example
  }
}
