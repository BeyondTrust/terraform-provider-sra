# List all Remote RDP Jump Items
data "sra_remote_rdp_list" "all" {}

# Filter by tag
data "sra_remote_rdp_list" "filtered" {
  tag = "Example"
}
