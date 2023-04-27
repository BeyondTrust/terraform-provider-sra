terraform {
  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

module "jp" {
  source    = "../data_sources/jumpoint"
  code_name = "matt_win"
}
module "jg" {
  source    = "../data_sources/jump_group"
  code_name = "group_2"
}

locals {
  name          = "fun_jump"
  hostname      = "10.10.10.16"
  jumpoint_id   = module.jp.jp[0].id
  jump_group_id = module.jg.jg[0].id
}

// Configuration
