terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

module "account_group" {
  source      = "../account_group"
  random_bits = var.random_bits
  name        = var.name
}

data "sra_group_policy_list" "gp" {}

resource "sra_vault_token_account" "new_token" {
  name             = "Group TOKEN ${var.name} ${var.random_bits}"
  token         = "${var.random_bits}${var.random_bits}"
  account_group_id = module.account_group.group.id
}

resource "sra_vault_token_account" "stand_alone" {
  name     = "Standalone TOKEN ${var.name} ${var.random_bits}"
  token = "${var.random_bits}${var.random_bits}"
}

resource "sra_vault_token_account" "stand_alone_gp" {
  name     = "Standalone TOKEN GP ${var.name} ${var.random_bits}"
  token = "${var.random_bits}${var.random_bits}"

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ]
}

resource "sra_vault_token_account" "stand_alone_ji" {
  name     = "Standalone TOKEN JIA ${var.name} ${var.random_bits}"
  token = "${var.random_bits}${var.random_bits}"

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      tag = [var.random_bits]
    }
    jump_items = [
      { id : module.account_group.shell.id, type : "shell_jump" }
    ]
  }
}

resource "sra_vault_token_account" "stand_alone_both" {
  name     = "Standalone TOKEN Both ${var.name} ${var.random_bits}"
  token = "${var.random_bits}${var.random_bits}"

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ]

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      tag = [var.random_bits]
    }
    jump_items = [
      { id : module.account_group.shell.id, type : "shell_jump" }
    ]
  }
}

data "sra_vault_account_list" "acc" {
  account_group_id = module.account_group.group.id
}
