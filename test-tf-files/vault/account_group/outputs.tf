output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "group" {
  description = "The created account group"
  value       = sra_vault_account_group.new_account_group
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_group_list.ag.items
}

output "shell" {
  description = "The shell jump used in the jump item association"
  value       = module.shell_jump.item
}
