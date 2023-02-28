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
resource "bt_shell_jump" "fun_jump" {
    name = "fun_jump"
    jumpoint_id = 1
    hostname = "10.10.10.125"
    protocol = "ssh"
    jump_group_id = 23
    jump_group_type = "personal"
    session_policy_id = 2
}

output "shell_jump" {
  value = resource.bt_shell_jump.fun_jump.hostname
}

data "bt_shell_jump_list" "sj" {
  name = "fun_jump"
}


output "existing_items" {
    value = data.bt_shell_jump_list.sj
}