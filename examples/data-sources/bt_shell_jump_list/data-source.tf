# List all Shell Jump Items
data "bt_shell_jump_list" "all" {}

# Filter by tag
data "bt_shell_jump_list" "filtered" {
  tag = "Example"
}
