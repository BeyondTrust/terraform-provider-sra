# Manage example MongoDB Tunnel Jump Item (PRA 26.1.1 or later)
resource "sra_mongodb_tunnel_jump" "example" {
  name                  = "Example MongoDB Tunnel"
  hostname              = "mongo.example.com"
  jumpoint_id           = 1
  jump_group_id         = 1
  tunnel_listen_address = "127.0.0.2"
  username              = "app_user"
  database              = "app"

  # Requires PRA 26.3.1 or later. Omit it on older appliances.
  auth_database = "admin"
}
