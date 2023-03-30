terraform {
  required_providers {
    sra = {
      source = "beyondtrust/beyondtrust-sra"
    }
  }
}

module "jp" {
  source    = "../data_sources/jumpoint"
  code_name = "matt_win"
}
module "jg" {
  source    = "../data_sources/jump_group"
  code_name = "group_2"
}

locals {
  name          = "fun_jump"
  hostname      = "10.10.10.16"
  jumpoint_id   = module.jp.jp[0].id
  jump_group_id = module.jg.jg[0].id
}

// Configuration

resource "sra_protocol_tunnel_jump" "example" {
  name               = "Example TCP Tunnel"
  hostname           = "example.host"
  jumpoint_id        = local.jumpoint_id
  jump_group_id      = local.jump_group_id
  tunnel_definitions = "22;24;80;8080"
}
resource "sra_protocol_tunnel_jump" "mssql" {
  name          = "Example MSSQL Tunnel"
  hostname      = "example.database"
  jumpoint_id   = local.jumpoint_id
  jump_group_id = local.jump_group_id
  tunnel_type   = "mssql"
  username      = "db_user"
}
resource "sra_remote_rdp" "item" {
  name          = "fun_rdp"
  hostname      = "10.10.10.10"
  jumpoint_id   = local.jumpoint_id
  jump_group_id = local.jump_group_id
}
resource "sra_remote_vnc" "item" {
  name          = local.name
  hostname      = local.hostname
  jumpoint_id   = local.jumpoint_id
  jump_group_id = local.jump_group_id
}
resource "sra_shell_jump" "item" {
  name          = local.name
  hostname      = local.hostname
  jumpoint_id   = local.jumpoint_id
  jump_group_id = local.jump_group_id
}
resource "sra_web_jump" "item" {
  name          = "Example Web Jump"
  url           = "https://example.host/login"
  jumpoint_id   = local.jumpoint_id
  jump_group_id = local.jump_group_id
}

// Output

output "items" {
  value = {
    ProtocolTunnel = resource.sra_protocol_tunnel_jump.example
    MSSQLTunnel    = resource.sra_protocol_tunnel_jump.mssql
    RemoteRDP      = resource.sra_remote_rdp.item
    RemoteVNC      = resource.sra_remote_vnc.item
    ShellJump      = resource.sra_shell_jump.item
    WebJump        = resource.sra_web_jump.item
  }
}
