# Manage example Oracle Tunnel Jump Item (PRA 26.3.1 or later)
resource "sra_oracle_tunnel_jump" "example" {
  name                  = "Example Oracle Tunnel"
  hostname              = "oracle.example.com"
  jumpoint_id           = 1 # sent to the API as gateway_id
  jump_group_id         = 1 # sent to the API as asset_group_id
  tunnel_listen_address = "127.0.0.3"
  username              = "app_user"
  database              = "ORCLPDB1"
}
