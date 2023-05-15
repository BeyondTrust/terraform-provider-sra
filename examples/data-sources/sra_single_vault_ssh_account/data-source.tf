# Get a single Vault SSH account by id
data "sra_single_vault_ssh_account" "single" {
  id = 5
}

# Get the first Vault SSH account with a given name
data "sra_vault_account_list" "filtered" {
  name = "SSH Account Name"
}
