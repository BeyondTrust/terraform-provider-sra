# Manage example Shell Jump Item
resource "sra_remote_rdp" "example" {
  name          = "Example RDP Jump"
  hostname      = "example.host"
  jumpoint_id   = 1
  jump_group_id = 1
}
