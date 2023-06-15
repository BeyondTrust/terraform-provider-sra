terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

module "account" {
  source      = "../user_pass_account"
  random_bits = var.random_bits
  name        = var.name
}

data "sra_vault_secret" "secret" {
  id = module.account.item.id
}
