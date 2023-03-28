terraform {
  required_providers {
    bt = {
        source = "hashicorp.com/edu/beyondtrust-sra"
    }
  }
}

module "gp" {
  source = "./goup_policy"

  name = "MFA"
}
module "jg" {
  source = "./jump_group"
  code_name = "group_2"
}
module "jir" {
  source = "./jump_item_role"
  name = "Start Sessions Only"
}
module "jp" {
  source = "./jumpoint"
  code_name = "matt_win"
}
module "sp" {
  source = "./session_policy"
  code_name = "fun_policy"
}

output "ds_out" {
    value = {
        GpResult = module.gp.gp
        JgResult = module.jg.jg
        JirResult = module.jir.jir
        JpResult = module.jp.jp
        SpResult = module.sp.sp
    }
}