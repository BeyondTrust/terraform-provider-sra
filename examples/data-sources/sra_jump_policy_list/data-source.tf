# List all Jump Policy
data "sra_jump_policy_list" "all" {}

# Filter by code_name
data "sra_jump_policy_list" "filtered" {
  code_name = "filter_code_name"
}
