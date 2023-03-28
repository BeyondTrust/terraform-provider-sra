terraform {
  required_providers {
    bt = {
        source = "hashicorp.com/edu/beyondtrust-sra"
    }
  }
}

// Variables

variable "code_name" {
  type = string
  default = ""
}

// Configuration

data "bt_jump_group_list" "jg" {
  code_name = var.code_name
}

// Output

output "jg" {
    value = data.bt_jump_group_list.jg.items
}

