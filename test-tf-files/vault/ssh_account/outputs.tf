output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "group" {
  description = "The created shell jump item"
  value       = sra_vault_account_group.new_account_group
}
output "item" {
  description = "The created shell jump item"
  value       = sra_vault_ssh_account.new_key
  sensitive   = true
}

output "group_list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_group_list.ag.items
}

output "item_list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_list.acc.items
}
