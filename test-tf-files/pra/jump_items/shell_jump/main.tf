terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

module "jump_resources" {
  source      = "../jumpoint_and_jump_group"
  random_bits = var.random_bits
}

resource "sra_shell_jump" "test" {
  name          = var.name
  hostname      = var.hostname
  jumpoint_id   = module.jump_resources.jumpoint.id
  jump_group_id = module.jump_resources.jump_group.id
  tag           = var.random_bits
}

data "sra_shell_jump_list" "list" {
  tag = var.random_bits
}
