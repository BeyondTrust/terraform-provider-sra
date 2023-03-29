# Manage example Remote VNC Jump Item
resource "sra_remote_vnc" "example" {
  name          = "Example Remote VNC"
  hostname      = "example.host"
  jumpoint_id   = 1
  jump_group_id = 1
}
