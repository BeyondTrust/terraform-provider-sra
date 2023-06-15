output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created shell jump item"
  value       = sra_jump_client_installer.test
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_jump_client_installer_list.list.items
}
