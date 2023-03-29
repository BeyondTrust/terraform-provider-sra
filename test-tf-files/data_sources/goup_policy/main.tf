terraform {
  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

// Variables

variable "name" {
  type    = string
  default = ""
}

// Configuration

data "sra_group_policy_list" "gp" {
  name = var.name
}

// Output

output "gp" {
  value = data.sra_group_policy_list.gp.items
}
