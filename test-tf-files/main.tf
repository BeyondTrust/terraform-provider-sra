terraform {
  required_providers {
    bt = {
        source = "hashicorp.com/edu/beyondtrust-sra"
    }
  }
}

provider "bt" {
    host = "mpam.dev.bomgar.com"
    client_id = "114635791a8bc6e21d813d5385d100afcb883a2d"
    client_secret = "wUwZTVwC0Erh3/01TcG41TbWHcntMgdRZHkhqcwNKYQK"
}

// Data Sources

data "bt_jump_group_list" "jg" {
  code_name = "group_2"
}
output "existing_jg" {
    value = data.bt_jump_group_list.jg.items[0]
}

data "bt_jump_item_role_list" "jr" {
  name = "Start Sessions Only"
}
output "existing_jir" {
    value = data.bt_jump_item_role_list.jr.items[0]
}

data "bt_jumpoint_list" "jp" {
  code_name = "matt_win"
}
output "existing_jp" {
    value = data.bt_jumpoint_list.jp.items[0]
}

data "bt_session_policy_list" "sp" {}
locals {
  sp_map = { for i, sp in data.bt_session_policy_list.sp.items : sp.code_name => sp }
}
output "existing_sp" {
    value = local.sp_map["fun_policy"]
}

data "bt_group_policy_list" "gp" {
  name = "MFA"
}
output "existing_gp" {
  value = data.bt_group_policy_list.gp.items[0]
}

// Resources

resource "bt_shell_jump" "fun_jump" {
    name = "fun_jump"
    hostname = "10.10.10.125"
    protocol = "ssh"
    jumpoint_id = data.bt_jumpoint_list.jp.items[0].id
    jump_group_id = data.bt_jump_group_list.jg.items[0].id
    jump_group_type = "shared"
}

output "shell_jump" {
  value = resource.bt_shell_jump.fun_jump
}

data "bt_shell_jump_list" "sj" {
  name = "fun_jump"
}

output "existing_items" {
    value = data.bt_shell_jump_list.sj.items
}
