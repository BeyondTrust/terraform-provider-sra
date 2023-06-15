output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created vnc jump item"
  value       = sra_remote_vnc.test
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_remote_vnc_list.list.items
}
