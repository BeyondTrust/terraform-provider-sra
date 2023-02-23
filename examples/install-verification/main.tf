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

data "bt_shell_jump_item" "shell_jumps" {}

output "shell_jump_items" {
    value = data.bt_shell_jump_item.shell_jumps
}

resource "bt_shell_jump" "fun_jump" {
    name = "fun_jump"
    jumpoint_id = 1
    hostname = "10.10.10.15"
    protocol = "ssh"
    jump_group_id = 23
    jump_group_type = "personal"
}