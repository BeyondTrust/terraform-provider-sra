output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created network tunnel jump item"
  value       = sra_network_tunnel_jump.test
}

output "item_secondary" {
  description = "The secondary network tunnel jump item"
  value       = sra_network_tunnel_jump.test_secondary
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_network_tunnel_jump_list.list.items
}
