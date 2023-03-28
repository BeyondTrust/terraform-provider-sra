# List all Shell Jump Items
data "sra_shell_jump_list" "all" {}

# Filter by tag
data "sra_shell_jump_list" "filtered" {
  tag = "Example"
}
