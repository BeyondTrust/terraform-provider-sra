output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "policy" {
  description = "The created account policy"
  value       = sra_vault_account_policy.new_account_policy
}

output "policy_false" {
  description = "The created account policy"
  value       = sra_vault_account_policy.new_account_policy_false
}

output "policy_mixed" {
  description = "The created account policy"
  value       = sra_vault_account_policy.new_account_policy_mixed
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_vault_account_policy_list.ap.items
}
