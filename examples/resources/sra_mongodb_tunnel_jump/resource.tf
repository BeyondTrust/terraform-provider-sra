# Manage example MongoDB Tunnel Jump Item (PRA 26.1.1 or later)
resource "sra_mongodb_tunnel_jump" "example" {
  name                  = "Example MongoDB Tunnel"
  hostname              = "mongo.example.com"
  jumpoint_id           = 1
  jump_group_id         = 1
  tunnel_listen_address = "127.0.0.2"
  username              = "app_user"
  database              = "app"
}

# Authenticate against a separate database (PRA 26.3.1 or later; older
# appliances reject auth_database)
resource "sra_mongodb_tunnel_jump" "with_auth_database" {
  name          = "Example MongoDB Tunnel with auth database"
  hostname      = "mongo.example.com"
  jumpoint_id   = 1
  jump_group_id = 1
  username      = "app_user"
  database      = "app"
  auth_database = "admin"
}
