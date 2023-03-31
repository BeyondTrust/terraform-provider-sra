# List all Vault Account Groups
data "sra_vault_account_group_list" "all" {}

# Filter by name
data "sra_vault_account_group_list" "filtered" {
  name = "Test Group"
}
