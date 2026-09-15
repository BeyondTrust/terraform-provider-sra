terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

data "sra_group_policy_list" "gp" {}

# Deliberately minimal: one account group, one toggle, no jump item association
# and no module dependencies, so a failure localises to membership handling.
#
# `cond ? [...] : null` is the correct HCL for a genuinely null Optional
# attribute and is distinct from `[]`. The distinction is the point of the test:
# removing the last membership must leave the attribute null, not empty.
resource "sra_vault_account_group" "toggle" {
  name        = "${var.name} ${var.random_bits} Toggle"
  description = var.random_bits

  group_policy_memberships = var.with_gp_membership ? [
    { group_policy_id : data.sra_group_policy_list.gp.items[0].id, role : "inject" }
  ] : null
}
