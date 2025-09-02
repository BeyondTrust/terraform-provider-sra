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

resource "sra_network_tunnel_jump" "test" {
  name          = var.name
  jumpoint_id   = module.jump_resources.jumpoint.id
  jump_group_id = module.jump_resources.jump_group.id
  tag           = var.random_bits
  filter_rules  = [ { ip_addresses = { list = ["10.0.0.5"] } } ]
}

data "sra_network_tunnel_jump_list" "list" {
  tag = var.random_bits
}

resource "sra_network_tunnel_jump" "test_secondary" {
  name          = "${var.name}-secondary"
  jumpoint_id   = module.jump_resources.jumpoint.id
  jump_group_id = module.jump_resources.jump_group.id
  tag           = var.random_bits
  filter_rules = [
    {
      ip_addresses = { list = ["10.0.0.1", "10.0.1.1"] }
      ports        = { range = { start = 1000, end = 2000 } }
      protocol     = "TCP"
    },
    {
      ip_addresses = { cidr = "192.168.0.0/24" }
      ports        = { list = [80, 443] }
      protocol     = "TCP"
    }
  ]
}
