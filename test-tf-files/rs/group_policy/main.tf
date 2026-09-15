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

  rep_perm_status      = "defined"
  perm_support_allowed = "full_support"
  perm_jump_client     = true
  perm_remote_rdp      = true
}

data "sra_group_policy_list" "listed" {
  name = sra_group_policy.item.name
}
