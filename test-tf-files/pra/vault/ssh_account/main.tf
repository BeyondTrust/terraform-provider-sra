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

resource "sra_vault_ssh_account" "new_key" {
  name                   = "Group Key ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""
  account_group_id       = module.account_group.group.id
}

resource "sra_vault_ssh_account" "stand_alone" {
  name                   = "Standalone Key ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""

  # Deliberately the empty string. The API rejects both "" and null for this
  # field and never returns it on a read, so a configuration written this way --
  # which is what a module passing an unset variable produces -- used to fail the
  # apply outright. Keep it set this way so that path stays covered.
  private_key_public_cert = ""
}

resource "sra_vault_ssh_account" "stand_alone_ca_key" {
  name                   = "Standalone CA Key ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""
  type                   = "ssh_ca"
}

resource "sra_vault_ssh_account" "stand_alone_ca" {
  name     = "Standalone CA ${var.name} ${var.random_bits}"
  username = var.random_bits
  type     = "ssh_ca"
}

resource "sra_vault_ssh_account" "stand_alone_gp" {
  name                   = "Standalone Key GP ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""

  group_policy_memberships = [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ]
}

resource "sra_vault_ssh_account" "stand_alone_ji" {
  name                   = "Standalone Key JIA ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""

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

resource "sra_vault_ssh_account" "stand_alone_both" {
  name                   = "Standalone Key Both ${var.name} ${var.random_bits}"
  username               = var.random_bits
  private_key            = var.private_key
  private_key_passphrase = ""

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

# The other two legal filter_type values. Until these existed, every fixture in
# the repo used "criteria", so two thirds of the enum had never been executed
# anywhere -- which is how a provider that could not create either of them
# shipped.
#
# The nested criteria block is deliberately ABSENT rather than empty. An absent
# block is what makes Criteria nil, which is the case that used to marshal as
# "criteria": null and be rejected. Writing `criteria = {}` here would pass
# against the unfixed provider and prove nothing.
resource "sra_vault_ssh_account" "stand_alone_any" {
  name                   = "Standalone Key Any ${var.name} ${var.random_bits}"
  username               = "${var.random_bits}any"
  private_key            = var.private_key
  private_key_passphrase = ""

  jump_item_association = {
    filter_type = "any_jump_items"
  }
}

resource "sra_vault_ssh_account" "stand_alone_none" {
  name                   = "Standalone Key None ${var.name} ${var.random_bits}"
  username               = "${var.random_bits}none"
  private_key            = var.private_key
  private_key_passphrase = ""

  jump_item_association = {
    filter_type = "no_jump_items"
  }
}

data "sra_vault_account_list" "acc" {
  account_group_id = module.account_group.group.id
}

data "sra_single_vault_ssh_account" "single" {
  id = sra_vault_ssh_account.new_key.id
}

data "sra_single_vault_ssh_account" "single_filter" {
  name = sra_vault_ssh_account.new_key.name
}
