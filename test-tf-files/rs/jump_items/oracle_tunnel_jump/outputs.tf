output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created oracle tunnel jump item"
  value       = sra_oracle_tunnel_jump.test
}

output "item_secondary" {
  description = "The secondary oracle tunnel jump item, left at every default"
  value       = sra_oracle_tunnel_jump.test_secondary
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_oracle_tunnel_jump_list.list.items
}
