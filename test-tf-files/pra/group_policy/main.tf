terraform {
  required_providers {
    sra = {
      source  = "beyondtrust/sra"
      version = "1.0.0"
    }
  }
}

resource "sra_group_policy" "item" {
  name = "terraform_group_policy_${var.random_bits}"

  access_perm_status  = "defined"
  perm_access_allowed = true
  perm_jump_client    = true
  perm_remote_rdp     = true

  # A product-only permission with no schema default. Setting the variable to
  # false leaves it out of the configuration, which must revoke it.
  perm_invite_external_user = var.grant_optional_permission ? true : null
}

data "sra_group_policy_list" "listed" {
  name = sra_group_policy.item.name
}
