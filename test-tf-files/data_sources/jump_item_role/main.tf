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

data "bt_jump_item_role_list" "jr" {
  name = var.name
}

// Output

output "jir" {
  value = data.bt_jump_item_role_list.jr.items
}