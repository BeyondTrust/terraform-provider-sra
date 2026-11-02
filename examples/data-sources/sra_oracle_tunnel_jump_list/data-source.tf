# List all Oracle Tunnel Jump Items
data "sra_oracle_tunnel_jump_list" "all" {}

# Filter by tag
data "sra_oracle_tunnel_jump_list" "filtered" {
  tag = "Example"
}
