# Manage example Shell Jump Item
resource "bt_shell_jump" "example" {
    name = "Example Shell Jump"
    hostname = "example.host"
    jumpoint_id = 1
    jump_group_id = 1
}