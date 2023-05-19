# Checkout and return the secret vaule for a Vault account
data "sra_vault_secret" "secret" {
  id = 5
}
