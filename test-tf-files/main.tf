terraform {
  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

variable "api_auth" {
  type = object({
    host          = string
    client_id     = string
    client_secret = string
  })
  sensitive = true
}

provider "sra" {
  host          = var.api_auth.host
  client_id     = var.api_auth.client_id
  client_secret = var.api_auth.client_secret
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
  name   = "fun_jump"
}
output "sj_items" {
  value = module.sj.items
}

