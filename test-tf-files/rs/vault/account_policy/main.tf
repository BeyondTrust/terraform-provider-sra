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
  description                 = var.random_bits
  auto_rotate_credentials     = true
  allow_simultaneous_checkout = true
  scheduled_password_rotation = true
}

resource "sra_vault_account_policy" "new_account_policy_false" {
  name                        = "${var.name} ${var.random_bits} False"
  description                 = var.random_bits
  auto_rotate_credentials     = false
  allow_simultaneous_checkout = false
  scheduled_password_rotation = false
}

resource "sra_vault_account_policy" "new_account_policy_mixed" {
  name                        = "${var.name} ${var.random_bits} Mixed"
  description                 = var.random_bits
  auto_rotate_credentials     = false
  allow_simultaneous_checkout = true
  scheduled_password_rotation = false
}

data "sra_vault_account_policy_list" "ap" {
  name = "${var.name} ${var.random_bits}"
}
