# List all Group Policies
data "bt_group_policy_list" "all" {}

# Filter by name
data "bt_group_policy_list" "filtered" {
  name = "Filter by name"
}
