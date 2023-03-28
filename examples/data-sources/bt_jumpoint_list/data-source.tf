# List all Jumpoints
data "bt_jumpoint_list" "all" {}

# Filter by public IP
data "bt_jumpoint_list" "filtered" {
  public_ip = "10.10.10.10"
}
