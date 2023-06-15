output "bits" {
  description = "Random bits used for naming"
  value       = var.random_bits
}

output "item" {
  description = "The created rdp jump item"
  value       = sra_remote_rdp.test
}

output "list" {
  description = "The datasource query result"
  value       = data.sra_remote_rdp_list.list.items
}
