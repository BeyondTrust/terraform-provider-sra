terraform {
  required_providers {
    bt = {
        source = "hashicorp.com/edu/beyondtrust-sra"
    }
  }
}

module "jp" {
  source = "../../data_sources/jumpoint"
  code_name = "matt_win"
}
module "jg" {
  source = "../../data_sources/jump_group"
  code_name = "group_2"
}

locals {
  name = "fun_jump"
  hostname = "10.10.10.16"
  jumpoint_id = module.jp.jp[0].id
  jump_group_id = module.jg.jg[0].id
}

// Configuration

resource "bt_shell_jump" "item" {
    name = local.name
    hostname = local.hostname
    jumpoint_id = local.jumpoint_id
    jump_group_id = local.jump_group_id
}

// Output

output "item" {
  value = resource.bt_shell_jump.item
}