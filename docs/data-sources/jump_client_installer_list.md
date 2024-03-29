---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sra_jump_client_installer_list Data Source - sra"
subcategory: ""
description: |-
  Fetch a list of Jump Client Installers.
  For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance
---

# sra_jump_client_installer_list (Data Source)

Fetch a list of Jump Client Installers.

For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance

## Example Usage

```terraform
# List all Jump Client Installers
data "sra_jump_client_installer_list" "all" {}

# Filter by tag
data "sra_jump_client_installer_list" "filtered" {
  tag = "example"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `connection_type` (String) Filter the list for items with a matching "connection_type". Should be either 'active' or 'passive'
- `jump_group_id` (Number) Filter the list for items with a matching "jump_group_id"
- `jump_group_type` (String) Filter the list for items with a matching "jump_group_type"
- `name` (String) Filter the list for items matching "name"
- `tag` (String) Filter the list for items with a matching "tag"

### Read-Only

- `items` (Attributes List) (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Required:

- `jump_group_id` (Number) The unique identifier of the shared Jump Group or user that owns this Jump Client.

Optional:

- `allow_override_attended_session_policy` (Boolean) If true, the attended session policy can be specified during installation, which will override the value specified in this API call. _This field only applies to RS_
- `allow_override_comments` (Boolean) If true, the comments can be specified during installation, which will override the comments specified in this API call.
- `allow_override_jump_group` (Boolean) If true, the jump group can be specified during installation, which will override the jump group id specified in this API call.
- `allow_override_jump_policy` (Boolean) If true, the jump policy can be specified during installation, which will override the jump policy id specified in this API call.
- `allow_override_max_offline_minutes` (Boolean) If true, the max offline minutes can be specified during installation, which will override the max offline minutes specified in this API call.
- `allow_override_name` (Boolean) If true, the name can be specified during installation, which will override the name specified in this API call.
- `allow_override_session_policy` (Boolean) If true, the session policy can be specified during installation, which will override the value specified in this API call. _This field only applies to PRA_
- `allow_override_tag` (Boolean) If true, the tag can be specified during installation, which will override the tag specified in this API call.
- `allow_override_unattended_session_policy` (Boolean) If true, the unattended session policy can be specified during installation, which will override the value specified in this API call. _This field only applies to RS_
- `attended_session_policy_id` (Number) The session policy used when an end user is present on the Jump Client system. _This field only applies to RS_
- `comments` (String) The Jump Client's comments.
- `connection_type` (String) The type of connection maintained between the appliance and the Jump Client. Cloud deployments only allow active Jump Clients.
- `customer_client_start_mode` (String) This setting determines how sessions are started from the deployed Jump Client. If normal, the customer client will start with the window visible. If minimized, it will start with the window minimized. If hidden, it will start with no visible customer window and will not appear in the taskbar. Hidden mode requires additional permission. _This field only applies to RS_
- `elevate_install` (Boolean) If true, the installer will attempt to elevate the Jump Client to make it run as a service.
- `elevate_prompt` (Boolean) If true, the installer will prompt for elevation credentials if necessary. This parameter is ignored if elevate_install is false.
- `jump_group_type` (String) The type of Jump Group that owns this Jump Client.
- `jump_policy_id` (Number) The unique identifier of the Jump Policy used to manage access to this Jump Item.
- `max_offline_minutes` (Number) The maximum number of minutes the installed Jump Client can be offline before being uninstalled. If 0, the Jump Client will follow the global lost client settings.
- `name` (String) The Jump Client's user-friendly name.
- `session_policy_id` (Number) The session policy used on the Jump Client system. _This field only applies to PRA_
- `tag` (String) The Jump Client's tag.
- `unattended_session_policy_id` (Number) The session policy used when an end user is not present on the Jump Client system. _This field only applies to RS_
- `valid_duration` (Number)

Read-Only:

- `expiration_timestamp` (String) The date/time at which the Jump Client installer will no longer be valid.
- `id` (String) The unique identifier assigned to this Jump Client Installer by the appliance.
- `installer_id` (String) The unique installer identifier that can be used to download the installer for a specific platform.
- `is_quiet` (Boolean) If true, the customer client will start minimized when sessions are started from the deployed Jump Client. _This field only applies to RS_
- `key_info` (String) The information needed to deploy a Windows MSI installer. Only included in the response when creating a new installer.
