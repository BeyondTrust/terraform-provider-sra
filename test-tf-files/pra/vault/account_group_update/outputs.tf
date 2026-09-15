output "group" {
  description = "The created account group"
  value       = sra_vault_account_group.toggle
}

output "gp" {
  description = "The group policy used for the membership"
  value       = data.sra_group_policy_list.gp.items[0]
}
