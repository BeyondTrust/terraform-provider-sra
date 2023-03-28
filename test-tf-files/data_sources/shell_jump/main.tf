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

data "sra_shell_jump_list" "sj" {
  name = var.name
}

// Output

output "items" {
    value = data.sra_shell_jump_list.sj.items
}