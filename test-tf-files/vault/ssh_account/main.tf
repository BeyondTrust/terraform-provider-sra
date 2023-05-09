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

module "account_group" {
  source      = "../account_group"
  random_bits = var.random_bits
  name        = var.name
}

data "sra_group_policy_list" "gp" {}

resource "sra_vault_ssh_account" "new_key" {
  name                   = "TF Test SSH Key ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""
  account_group_id       = module.account_group.group.id

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ]
}

data "sra_vault_account_list" "acc" {
  account_group_id = resource.sra_vault_account_group.new_account_group.id
}
