output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created mongodb tunnel jump item"
  value       = sra_mongodb_tunnel_jump.test
}

output "item_secondary" {
  description = "The secondary mongodb tunnel jump item, left at every default"
  value       = sra_mongodb_tunnel_jump.test_secondary
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_mongodb_tunnel_jump_list.list.items
}
