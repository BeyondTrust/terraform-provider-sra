---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sra_vault_account_list Data Source - sra"
subcategory: ""
description: |-
  Fetch a list of Session Policies.
  For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance
---

# sra_vault_account_list (Data Source)

Fetch a list of Session Policies.

For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance

## Example Usage

```terraform
# List all Web Jump Items
data "sra_web_jump_list" "all" {}

# Filter by tag
data "sra_web_jump_list" "filtered" {
  url = "https://exciting.site"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `account_group_id` (Number) Filter the list for items in account group with id "account_group_id"
- `include_personal` (Boolean) Set to 'true' to allows results to include personal accounts
- `name` (String) Filter the list for items matching "name"
- `type` (String) Filter the list for items matching "name"

### Read-Only

- `items` (Attributes List) (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Required:

- `name` (String)

Optional:

- `description` (String)
- `owner_user_id` (Number)

Read-Only:

- `account_group_id` (Number)
- `account_policy` (String)
- `id` (String)
- `personal` (Boolean)
- `type` (String)

