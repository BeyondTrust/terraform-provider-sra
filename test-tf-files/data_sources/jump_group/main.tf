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

data "sra_jump_group_list" "jg" {
  code_name = var.code_name
}

// Output

output "jg" {
    value = data.sra_jump_group_list.jg.items
}

