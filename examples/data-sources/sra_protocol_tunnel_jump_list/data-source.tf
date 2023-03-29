# List all Remote RDP Jump Items
data "sra_protocol_tunnel_jump_list" "all" {}

# Filter by tag
data "sra_protocol_tunnel_jump_list" "filtered" {
  tag = "Example"
}
