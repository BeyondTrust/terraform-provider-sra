# List all Jump Item Roles
data "sra_jump_item_role_list" "all" {}

# Filter by name
data "sra_jump_item_role_list" "filtered" {
  name = "Filter name"
}
