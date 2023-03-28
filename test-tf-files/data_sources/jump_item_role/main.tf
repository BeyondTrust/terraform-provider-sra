terraform {
  required_providers {
    sra = {
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

data "sra_jump_item_role_list" "jr" {
  name = var.name
}

// Output

output "jir" {
  value = data.sra_jump_item_role_list.jr.items
}