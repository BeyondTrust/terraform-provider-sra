# List all Vault Accounts
data "sra_vault_account_list" "all" {}

# Filter by account group id
data "sra_vault_account_list" "filtered" {
  account_group_id = 5
}
