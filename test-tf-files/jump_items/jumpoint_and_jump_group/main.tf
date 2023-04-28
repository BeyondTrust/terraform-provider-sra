terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

locals {
  code_name = "example_${var.random_bits}"
}

resource "sra_jumpoint" "example" {
  name                    = "Test JP ${var.random_bits}"
  code_name               = local.code_name
  platform                = "linux-x86"
  shell_jump_enabled      = true
  protocol_tunnel_enabled = true
}

resource "sra_jump_group" "example" {
  name      = "Test JG ${var.random_bits}"
  code_name = local.code_name
}

data "sra_jumpoint_list" "list" {
  code_name = local.code_name
}
data "sra_jump_group_list" "list" {
  code_name = local.code_name
}
