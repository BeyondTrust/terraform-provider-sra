terraform {
  required_providers {
    bt = {
        source = "hashicorp.com/edu/beyondtrust-sra"
    }
  }
}

// Variables

variable "name" {
  type = string
  default = ""
}

// Configuration

data "bt_group_policy_list" "gp" {
  name = var.name
}

// Output

output "gp" {
  value = data.bt_group_policy_list.gp.items
}