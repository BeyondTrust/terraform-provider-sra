terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

locals {
  code_name = "example_${var.random_bits}"
}

data "sra_group_policy_list" "gp" {}
data "sra_jump_policy_list" "jp" {}
data "sra_jump_item_role_list" "jir" {}

resource "sra_jumpoint" "example" {
  name                    = "Test JP ${var.random_bits}"
  code_name               = local.code_name
  platform                = "linux-x86"
  shell_jump_enabled      = true
  protocol_tunnel_enabled = true
}

resource "sra_jump_group" "example" {
  name      = "Test JG ${var.random_bits}"
  code_name = local.code_name
}

resource "sra_jumpoint" "example_gp" {
  name                    = "Test JP GP ${var.random_bits}"
  code_name               = "gp_${local.code_name}"
  platform                = "linux-x86"
  shell_jump_enabled      = true
  protocol_tunnel_enabled = true

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id }
  ]
}

resource "sra_jump_group" "example_gp" {
  name      = "Test JG GP ${var.random_bits}"
  code_name = "gp_${local.code_name}"

  group_policy_memberships = [
    {
      group_policy_id : data.sra_group_policy_list.gp.items[0].id,
      jump_item_role_id : data.sra_jump_item_role_list.jir.items[0].id,
      jump_policy_id : data.sra_jump_policy_list.jp.items[0].id
    }
  ]
}

data "sra_jumpoint_list" "list" {
  code_name = local.code_name
}
data "sra_jump_group_list" "list" {
  code_name = local.code_name
}
