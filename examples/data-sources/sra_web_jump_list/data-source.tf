# List all Web Jump Items
data "sra_vault_account_list" "all" {}

# Filter by tag
data "sra_vault_account_list" "filtered" {
  account_group_id = 5
}
