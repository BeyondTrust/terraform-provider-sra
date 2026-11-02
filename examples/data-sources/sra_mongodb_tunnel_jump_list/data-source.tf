# List all MongoDB Tunnel Jump Items
data "sra_mongodb_tunnel_jump_list" "all" {}

# Filter by tag
data "sra_mongodb_tunnel_jump_list" "filtered" {
  tag = "Example"
}
