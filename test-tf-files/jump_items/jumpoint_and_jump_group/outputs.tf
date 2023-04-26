output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "jumpoint" {
  description = "The created jumpoint item"
  value       = sra_jumpoint.example
}
output "jump_group" {
  description = "The created jump group item"
  value       = sra_jump_group.example
}

output "jumpoint_list" {
  description = "The datasource query result"
  value       = data.sra_jumpoint_list.list.items
}
output "jump_group_list" {
  description = "The datasource query result"
  value       = data.sra_jump_group_list.list.items
}
