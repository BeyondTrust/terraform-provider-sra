terraform {
  required_version = ">= 1.0.0"

  required_providers {
    sra = {
      source = "beyondtrust/sra"
    }
  }
}

module "jump_resources" {
  source      = "../jumpoint_and_jump_group"
  random_bits = var.random_bits
}

resource "sra_postgre_sql_tunnel_jump" "test" {
  name                  = var.name
  hostname              = var.hostname
  jumpoint_id           = 1
  jump_group_id         = 1
  tag                   = var.random_bits
  username              = var.username
  database              = var.database
  tunnel_listen_address = var.tunnel_listen_address
}

data "sra_postgre_sql_tunnel_jump_list" "list" {
  tag = var.random_bits
}

resource "sra_postgre_sql_tunnel_jump" "test_secondary" {
  name                  = "${var.name}-secondary"
  hostname              = var.hostname
  jumpoint_id           = 1
  jump_group_id         = 1
  tag                   = var.random_bits
  username              = "secondary_user"
  database              = var.database
  tunnel_listen_address = "127.0.0.2"
}

