#!/usr/bin/env bash

# Item can be imported by specifying the ID
terraform import sra_vault_account_group.example 123

# Import also reads group_policy_memberships. A configuration that omits them
# plans their removal, which revokes that group policy's access. Declare them,
# generate the configuration with -generate-config-out (it includes them), or
# add lifecycle { ignore_changes = [group_policy_memberships] }.
#
# It also reads jump_item_association. A configuration that omits it plans a
# reset to any_jump_items, which lets this group's credentials be injected into
# any jump item. Import warns when that would widen the association.
