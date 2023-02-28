terraform {
  required_providers {
    bt = {
        source = "hasicorp.com/edu/beyondtrust-sra"
    }
  }
}

provider "bt" {
    host = "mpam.dev.bomgar.com"
    client_id = "114635791a8bc6e21d813d5385d100afcb883a2d"
    client_secret = "wUwZTVwC0Erh3/01TcG41TbWHcntMgdRZHkhqcwNKYQK"
}

output "shell_jump" {
  value = resource.bt_shell_jump.fun_jump
}

data "bt_jump_group_list" "jg" {
  code_name = "group_2"
}
data "bt_jumpoint_list" "jp" {
  code_name = "matt_win"
}

resource "bt_shell_jump" "fun_jump" {
    name = "fun_jump"
    hostname = "10.10.10.125"
    protocol = "ssh"
    jumpoint_id = data.bt_jumpoint_list.jp.items[0].id
    jump_group_id = data.bt_jump_group_list.jg.items[0].id
    jump_group_type = "shared"
}

data "bt_shell_jump_list" "sj" {
  name = "fun_jump"
}

output "existing_items" {
    value = data.bt_shell_jump_list.sj.items
}