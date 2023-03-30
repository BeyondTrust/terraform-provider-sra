# Manage example Jumpoint
resource "sra_jumpoint" "example" {
  name      = "Example Jumpoint"
  code_name = "example_jumpoint"
  platform  = "linux-x86"
}
