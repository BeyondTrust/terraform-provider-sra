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
  description = "The created token"
  value       = sra_vault_token_account.new_token
  sensitive   = true
}

output "stand_alone" {
  description = "The created token"
  value       = sra_vault_token_account.stand_alone
  sensitive   = true
}

output "stand_alone_gp" {
  description = "The created token"
  value       = sra_vault_token_account.stand_alone_gp
  sensitive   = true
}

output "stand_alone_ji" {
  description = "The created token"
  value       = sra_vault_token_account.stand_alone_ji
  sensitive   = true
}

output "stand_alone_both" {
  description = "The created token"
  value       = sra_vault_token_account.stand_alone_both
  sensitive   = true
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_list.acc.items
}
