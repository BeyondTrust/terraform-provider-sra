# List all Remote RDP Jump Items
data "sra_remote_vnc_list" "all" {}

# Filter by tag
data "sra_remote_vnc_list" "filtered" {
  tag = "Example"
}
