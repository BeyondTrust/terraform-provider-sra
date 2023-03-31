# Create and manage a new SSH Key in Vault using the "tls" provider
# for key genereation

resource "tls_private_key" "test_key" {
  algorithm = "ED25519"
}

resource "sra_vault_ssh_account" "new_key" {
  name                   = "Test SSH Key"
  username               = "test"
  private_key            = tls_private_key.test_key.private_key_openssh
  private_key_passphrase = ""
}

output "pub_key" {
  value = sra_vault_ssh_account.new_key.public_key
}
