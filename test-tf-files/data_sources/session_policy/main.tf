terraform {
  required_providers {
    sra = {
        source = "beyondtrust/beyondtrust-sra"
    }
  }
}

// Variables

variable "code_name" {
  type = string
  default = ""
}

// Configuration

data "sra_session_policy_list" "sp" {}
locals {
  sp_map = { for i, sp in data.sra_session_policy_list.sp.items : sp.code_name => sp }
}

// Output

output "sp" {
    value = var.code_name != "" ? local.sp_map[var.code_name] : data.sra_session_policy_list.sp.items[0]
}
output "sp_list" {
    value = data.sra_session_policy_list.sp.items
}