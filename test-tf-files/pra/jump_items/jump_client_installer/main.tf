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

data "sra_session_policy_list" "sp" {}
locals {
  sp_map = { for i, sp in data.sra_session_policy_list.sp.items : sp.id => sp }
}

resource "sra_jump_client_installer" "test" {
  name          = var.name
  jump_group_id = module.jump_resources.jump_group.id
  tag           = var.random_bits

  session_policy_id             = local.sp_map["2"].id
  allow_override_session_policy = true
}

data "sra_jump_client_installer_list" "list" {
  tag = var.random_bits
}
