output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created mysql tunnel jump item"
  value       = sra_my_sql_tunnel_jump.test
}

output "item_secondary" {
  description = "The secondary mysql tunnel jump item"
  value       = sra_my_sql_tunnel_jump.test_secondary
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_my_sql_tunnel_jump_list.list.items
}
