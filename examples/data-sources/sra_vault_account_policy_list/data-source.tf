# List all Vault Account Policies
data "sra_vault_account_policy_list" "all" {}

# Filter by name
data "sra_vault_account_policy_list" "filtered" {
  name = "Test Policy"
}
