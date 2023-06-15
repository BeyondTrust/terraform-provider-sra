output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created web jump item"
  value       = sra_web_jump.item
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_web_jump_list.list.items
}
