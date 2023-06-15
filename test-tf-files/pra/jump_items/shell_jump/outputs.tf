output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created shell jump item"
  value       = sra_shell_jump.test
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_shell_jump_list.list.items
}
