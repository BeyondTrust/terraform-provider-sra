# List all Jump Item Roles
data "bt_jump_item_role_list" "all" {}

# Filter by name
data "bt_jump_item_role_list" "filtered" {
  name = "Filter name"
}
