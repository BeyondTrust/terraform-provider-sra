output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created postgresql tunnel jump item"
  value       = sra_postgresql_tunnel_jump.test
}

output "item_secondary" {
  description = "The secondary postgresql tunnel jump item"
  value       = sra_postgresql_tunnel_jump.test_secondary
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_postgresql_tunnel_jump_list.list.items
}




