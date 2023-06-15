output "item" {
  description = "The vault account being queried"
  value       = module.account.item
  sensitive   = true
}

output "secret" {
  description = "The vault secret query result"
  value       = data.sra_vault_secret.secret.account
  sensitive   = true
}
