output "item" {
  value = sra_group_policy.item
}

output "listed" {
  value = data.sra_group_policy_list.listed.items
}
