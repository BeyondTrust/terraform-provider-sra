# Manage example Protocol Tunnel Jump Items
resource "sra_protocol_tunnel_jump" "example" {
  name               = "Example TCP Tunnel"
  hostname           = "example.host"
  jumpoint_id        = 1
  jump_group_id      = 1
  tunnel_definitions = "22;24;80;8080"
}

resource "sra_protocol_tunnel_jump" "mssql" {
  name          = "Example MSSQL Tunnel"
  hostname      = "example.database"
  jumpoint_id   = 1
  jump_group_id = 1
  tunnel_type   = "mssql"
  username      = "db_user"
}

# Kubernetes tunnels need a Linux Jumpoint. hostname is omitted: the appliance
# derives it from url. The appliance trims both values, so trimspace() strips the
# newline file() leaves at the end of a PEM.
resource "sra_protocol_tunnel_jump" "k8s" {
  name            = "Example Kubernetes Tunnel"
  url             = "https://k8s.example.com:6443"
  jumpoint_id     = 1
  jump_group_id   = 1
  tunnel_type     = "k8s"
  ca_certificates = trimspace(file("ca.pem"))
}
