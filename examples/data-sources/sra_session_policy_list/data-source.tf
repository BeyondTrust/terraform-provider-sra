# List all Session Policies
data "sra_session_policy_list" "sp" {}

# Map by code_name for easy access
locals {
  sp_map = { for i, sp in data.sra_session_policy_list.sp.items : sp.code_name => sp }
}

output "specific_session_policy" {
  value = local.sp_map["example_code_name"]
}
