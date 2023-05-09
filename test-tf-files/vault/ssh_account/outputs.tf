output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "group" {
  description = "The account group used"
  value       = module.account_group.group
}

output "item" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.new_key
  sensitive   = true
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_list.acc.items
}
