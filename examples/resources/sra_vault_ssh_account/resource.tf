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

  # Omit the following configuration to use account group settings
  group_policy_memberships = [
    { group_policy_id : "123", role : "inject" }
  ]

  jump_item_association = {
    filter_type = "criteria"
    criteria = {
      shared_jump_groups = [2, 3]
      tag                = ["tftest"]
    }
    jump_items = [
      { id : 123, type : "shell_jump" }
    ]
  }
}

output "pub_key" {
  value = sra_vault_ssh_account.new_key.public_key
}
