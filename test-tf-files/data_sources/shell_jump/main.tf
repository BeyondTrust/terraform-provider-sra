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

data "bt_shell_jump_list" "sj" {
  name = var.name
}

// Output

output "items" {
    value = data.bt_shell_jump_list.sj.items
}