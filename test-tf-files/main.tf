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

module "gp" {
  source = "./data_sources/goup_policy"

  name = "MFA"
}
output "gp_result" {
  value = module.gp.gp
}

module "jg" {
  source = "./data_sources/jump_group"
  code_name = "group_2"
}
output "jg_result" {
  value = module.jg.jg
}

module "jir" {
  source = "./data_sources/jump_item_role"
  name = "Start Sessions Only"
}
output "jir_result" {
  value = module.jir.jir
}

module "jp" {
  source = "./data_sources/jumpoint"
  code_name = "matt_win"
}
output "jp_result" {
  value = module.jp.jp
}

module "sp" {
  source = "./data_sources/session_policy"
  code_name = "fun_policy"
}
output "sp_result" {
  value = module.sp.sp
}

// Resources

module "shell_jump" {
  source = "./resources/shell_jump"
}

output "shell_jump_item" {
  value = module.shell_jump.item
}

module "sj" {
  source = "./data_sources/shell_jump"
  name = "fun_jump"
}
output "sj_items" {
  value = module.sj.items
}
