terraform {
  # This module is now only being tested with Terraform 1.1.x. However, to make upgrading easier, we are setting 1.0.0 as the minimum version.
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

resource "sra_jumpoint" "example" {
  name      = "Test JP ${var.random_bits}"
  code_name = "example_${var.random_bits}"
  platform  = "linux-x86"
}

resource "sra_jump_group" "example" {
  name      = "Test JG ${var.random_bits}"
  code_name = "example_${var.random_bits}"
}

resource "sra_remote_rdp" "test" {
  name          = var.name
  hostname      = var.hostname
  jumpoint_id   = sra_jumpoint.example.id
  jump_group_id = sra_jump_group.example.id
  tag           = var.random_bits
}

data "sra_remote_rdp_list" "list" {
  tag = var.random_bits
}
