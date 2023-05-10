terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

module "shell_jump" {
  source      = "../../jump_items/shell_jump"
  random_bits = var.random_bits
}

data "sra_group_policy_list" "gp" {}

resource "sra_vault_account_group" "new_account_group" {
  name        = "${var.name} ${var.random_bits}"
  description = var.random_bits

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ]

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      tag = [var.random_bits]
    }
    jump_items = [
      { id : module.shell_jump.item.id, type : "shell_jump" }
    ]
  }
}

resource "sra_vault_account_group" "new_account_group_nothing" {
  name        = "${var.name} ${var.random_bits} Nothing"
  description = var.random_bits
}

resource "sra_vault_account_group" "new_account_group_gp" {
  name        = "${var.name} ${var.random_bits} GP"
  description = var.random_bits

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ]
}

resource "sra_vault_account_group" "new_account_group_jia" {
  name        = "${var.name} ${var.random_bits} JIA"
  description = var.random_bits

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      tag = [var.random_bits]
    }
    jump_items = [
      { id : module.shell_jump.item.id, type : "shell_jump" }
    ]
  }
}

data "sra_vault_account_group_list" "ag" {
  name = "${var.name} ${var.random_bits}"
}
