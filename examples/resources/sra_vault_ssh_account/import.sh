#!/usr/bin/env bash

# Item can be imported by specifying the ID
terraform import sra_vault_ssh_account.example 123

# Import also reads group_policy_memberships. A configuration that omits them
# plans their removal, which revokes that group policy's access. Declare them,
# generate the configuration with -generate-config-out (it includes them), or
# add lifecycle { ignore_changes = [group_policy_memberships] }.
#
# private_key and private_key_passphrase plan an update after import: the API
# never returns them.
