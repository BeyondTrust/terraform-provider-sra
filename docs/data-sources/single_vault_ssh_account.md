---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sra_single_vault_ssh_account Data Source - sra"
subcategory: ""
description: |-
  Fetch a single Vault SSH Account; useful for reading the public_key from an existing SSH account in Vault. If an ID is provided, that account will be returned. If an ID is not provided, the first account found with the specified filters will be returned.
  For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance
---

# sra_single_vault_ssh_account (Data Source)

Fetch a single Vault SSH Account; useful for reading the public_key from an existing SSH account in Vault. If an ID is provided, that account will be returned. If an ID is not provided, the first account found with the specified filters will be returned.

For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance

## Example Usage

```terraform
# Get a single Vault SSH account by id
data "sra_single_vault_ssh_account" "single" {
  id = 5
}

# Get the first Vault SSH account with a given name
data "sra_vault_account_list" "filtered" {
  name = "SSH Account Name"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `account` (Attributes) (see [below for nested schema](#nestedatt--account))
- `account_group_id` (Number) Fetch the first SSH account in account group with id "account_group_id"
- `id` (String) Get the account with ID "id". If provided, no other filter options will be used.
- `include_personal` (Boolean) Set to 'true' to allows results to include personal accounts
- `name` (String) Fetch the first SSH account matching "name"

<a id="nestedatt--account"></a>
### Nested Schema for `account`

Required:

- `name` (String)
- `username` (String)

Optional:

- `description` (String)
- `owner_user_id` (Number)
- `private_key_public_cert` (String)

Read-Only:

- `account_group_id` (Number)
- `account_policy` (String)
- `id` (String)
- `last_checkout_timestamp` (String)
- `personal` (Boolean)
- `public_key` (String)
- `type` (String)
