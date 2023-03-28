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

module "sra_ds" {
  source = "./data_sources"
}
output "data_source_list" {
  value = module.sra_ds.ds_out
}

// Resources

module "ji" {
  source = "./jump_items"
}
output "shell_jump_item" {
  value = {
    ShellJump = module.ji.shell_jump
    RemoteRDP = module.ji.remote_rdp
  }
}

module "sj" {
  source = "./data_sources/shell_jump"
  name = "fun_jump"
}
output "sj_items" {
  value = module.sj.items
}

