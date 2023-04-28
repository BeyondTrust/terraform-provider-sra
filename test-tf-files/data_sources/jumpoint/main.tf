terraform {
  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

// Variables

variable "code_name" {
  type    = string
  default = ""
}

// Configuration
data "sra_jumpoint_list" "jp" {
  code_name = var.code_name
}

// Output

output "jp" {
  value = data.sra_jumpoint_list.jp.items
}
