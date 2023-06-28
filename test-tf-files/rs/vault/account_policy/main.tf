terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

resource "sra_vault_account_policy" "new_account_policy" {
  name                        = "${var.name} ${var.random_bits}"
  code_name                   = var.random_bits
  auto_rotate_credentials     = true
  allow_simultaneous_checkout = true
  scheduled_password_rotation = true
  maximum_password_age        = 10
}

resource "sra_vault_account_policy" "new_account_policy_false" {
  name                        = "${var.name} ${var.random_bits} False"
  code_name                   = "${var.random_bits}_false"
  auto_rotate_credentials     = false
  allow_simultaneous_checkout = false
  scheduled_password_rotation = false
}

resource "sra_vault_account_policy" "new_account_policy_mixed" {
  name                        = "${var.name} ${var.random_bits} Mixed"
  code_name                   = "${var.random_bits}_mixed"
  auto_rotate_credentials     = false
  allow_simultaneous_checkout = true
  scheduled_password_rotation = false
}

data "sra_vault_account_policy_list" "ap" {
  code_name = var.random_bits
}
