output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "group" {
  description = "The account group used"
  value       = module.account_group.group
}

output "shell" {
  description = "The shell jump item used"
  value       = module.account_group.shell
}

output "item" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.new_key
  sensitive   = true
}

output "stand_alone" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.stand_alone
  sensitive   = true
}

output "stand_alone_ca_key" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.stand_alone_ca_key
  sensitive   = true
}

output "stand_alone_ca" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.stand_alone_ca
  sensitive   = true
}

output "stand_alone_gp" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.stand_alone_gp
  sensitive   = true
}

output "stand_alone_ji" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.stand_alone_ji
  sensitive   = true
}

output "stand_alone_both" {
  description = "The created ssh account"
  value       = sra_vault_ssh_account.stand_alone_both
  sensitive   = true
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_list.acc.items
}

output "single" {
  description = "The datasource query result"
  value       = data.sra_single_vault_ssh_account.single.account
}
output "single_filter" {
  description = "The datasource result when querying with a filter"
  value       = data.sra_single_vault_ssh_account.single_filter.account
}
