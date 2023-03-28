# List all Group Policies
data "sra_group_policy_list" "all" {}

# Filter by name
data "sra_group_policy_list" "filtered" {
  name = "Filter by name"
}
