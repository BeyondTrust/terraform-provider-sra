# List all Jump Groups
data "bt_jump_group_list" "all" {}

# Filter by code_name
data "bt_jump_group_list" "filtered" {
  code_name = "filter_code_name"
}
