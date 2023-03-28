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
data "bt_jumpoint_list" "jp" {
  code_name = var.code_name
}

// Output

output "jp" {
  value = data.bt_jumpoint_list.jp.items
}