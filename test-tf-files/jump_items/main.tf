terraform {
  required_providers {
    sra = {
        source = "beyondtrust/beyondtrust-sra"
    }
  }
}

module "jp" {
  source = "../data_sources/jumpoint"
  code_name = "matt_win"
}
module "jg" {
  source = "../data_sources/jump_group"
  code_name = "group_2"
}

locals {
  name = "fun_jump"
  hostname = "10.10.10.16"
  jumpoint_id = module.jp.jp[0].id
  jump_group_id = module.jg.jg[0].id
}

// Configuration

resource "sra_shell_jump" "item" {
    name = local.name
    hostname = local.hostname
    jumpoint_id = local.jumpoint_id
    jump_group_id = local.jump_group_id
}

resource "sra_remote_rdp" "item" {
  name = "fun_rdp"
  jumpoint_id = local.jumpoint_id
  hostname = "10.10.10.10"
  jump_group_id = local.jump_group_id
}
resource "sra_remote_vnc" "item" {
    name = local.name
    hostname = local.hostname
    jumpoint_id = local.jumpoint_id
    jump_group_id = local.jump_group_id
}

// Output

output "shell_jump" {
  value = resource.sra_shell_jump.item
}
output "remote_rdp" {
  value = resource.sra_remote_rdp.item
}
output "remote_vnc" {
  value = resource.sra_remote_vnc.item
}