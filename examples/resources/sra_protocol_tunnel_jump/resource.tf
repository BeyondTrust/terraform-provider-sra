# Manage example Remote VNC Jump Item
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
  useranme      = "db_user"
}

resource "sra_k8s_tunnel_jump" "k8s" {
  name          = "Example Kubernetes Tunnel"
  url      = "example.cluster"
  jumpoint_id   = 1
  jump_group_id = 1
  tunnel_type   = "k8s"
  ca_certificates = "**PEM CA certificate**"
}
