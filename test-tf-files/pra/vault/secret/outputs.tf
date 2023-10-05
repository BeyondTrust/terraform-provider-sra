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

output "ssh_item" {
  description = "The vault account being queried"
  value       = module.ssh.item
  sensitive   = true
}
output "ssh_ca_item" {
  description = "The vault account being queried"
  value       = module.ssh.stand_alone_ca
  sensitive   = true
}

output "secret_ssh" {
  description = "The vault secret query result"
  value       = data.sra_vault_secret.secret_ssh.account
  sensitive   = true
}

output "secret_ssh_ca" {
  description = "The vault secret query result"
  value       = data.sra_vault_secret.secret_ssh_ca.account
  sensitive   = true
}
