output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "gp" {
  description = "The group policy used"
  value       = data.sra_group_policy_list.gp.items[0]
}
output "jp" {
  description = "The jump policy used"
  value       = data.sra_jump_policy_list.jp.items[0]
}
output "jir" {
  description = "The jump item role used"
  value       = data.sra_jump_item_role_list.jir.items[0]
}

output "jumpoint" {
  description = "The created jumpoint item"
  value       = sra_jumpoint.example
}
output "jump_group" {
  description = "The created jump group item"
  value       = sra_jump_group.example
}

output "jumpoint_gp" {
  description = "The created jumpoint item"
  value       = sra_jumpoint.example_gp
}
output "jump_group_gp" {
  description = "The created jump group item"
  value       = sra_jump_group.example_gp
}

output "jumpoint_list" {
  description = "The datasource query result"
  value       = data.sra_jumpoint_list.list.items
}
output "jump_group_list" {
  description = "The datasource query result"
  value       = data.sra_jump_group_list.list.items
}
