output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created protocol tunnel jump item"
  value       = sra_protocol_tunnel_jump.test
}

output "sql_item" {
  description = "The created sql server tunnel jump item"
  value       = sra_protocol_tunnel_jump.test_sql
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_protocol_tunnel_jump_list.list.items
}
