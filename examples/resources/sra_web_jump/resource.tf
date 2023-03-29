# Manage example Web Jump Item
resource "sra_web_jump" "example" {
  name          = "Example Web Jump"
  url           = "https://example.host/login"
  jumpoint_id   = 1
  jump_group_id = 1
}
