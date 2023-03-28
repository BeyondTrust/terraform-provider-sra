# List all Jumpoints
data "sra_jumpoint_list" "all" {}

# Filter by public IP
data "sra_jumpoint_list" "filtered" {
  public_ip = "10.10.10.10"
}
