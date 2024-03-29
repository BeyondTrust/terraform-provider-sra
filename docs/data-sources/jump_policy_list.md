---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sra_jump_policy_list Data Source - sra"
subcategory: ""
description: |-
  Fetch a list of Jump Policies.
  For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance
---

# sra_jump_policy_list (Data Source)

Fetch a list of Jump Policies.

For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance

## Example Usage

```terraform
# List all Jump Policy
data "sra_jump_policy_list" "all" {}

# Filter by code_name
data "sra_jump_policy_list" "filtered" {
  code_name = "filter_code_name"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `code_name` (String) Filter the list for Jumpoints with a matching "code_name"

### Read-Only

- `items` (Attributes List) (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Required:

- `display_name` (String) The display name of the Jump Policy.

Optional:

- `approval_approver_scope` (String) The scope of approval granted to approvers. If "not_requestor", then the approver cannot approve their own requests. If "anyone", anyone who is permitted to approve, can approve requests, including their own. _This field only applies to PRA_
- `approval_display_name` (String) The display name of the approvers that requestors will see. It is required only if approvals are enabled. _This field only applies to PRA_
- `approval_email_addresses` (Set of String) This field only applies to PRA
- `approval_email_language` (String) The language in which approval emails will be sent. Must be the locale code for one of the locales listed on the Localization → Languages page. _This field only applies to PRA_
- `approval_max_duration` (Number) The number of minutes a user is allowed to access the Jump Item after approval is granted. The maximum is 52 weeks in minutes. _This field only applies to PRA_
- `approval_required` (Boolean) If true, users must wait for approval from one of the approvers before they can start a session. This setting cannot be enabled when ```schedule_enabled``` is true. _This field only applies to PRA_
- `approval_scope` (String) The scope of access granted by approvals. If "requestor", only the requestor has access. If "anyone", anyone who is permitted to request access has access. _This field only applies to PRA_
- `approval_user_ids` (Set of String) This field only applies to PRA
- `code_name` (String) The code name of the Jump Policy.
- `description` (String) The Jump Policy's comments.
- `notification_display_name` (String) The display name of the recipients shown to users. Required in POST only if one or more notifications are enabled. _This field only applies to PRA_
- `notification_email_addresses` (Set of String) This field only applies to PRA
- `notification_email_language` (String) The language in which notification emails will be sent. Must be the locale code for one of the locales listed on the Localization → Languages page. _This field only applies to PRA_
- `recordings_disabled` (Boolean) If true, sessions will not be recorded even if recordings are enabled on the Configuration → Options page. This affects Screen Sharing, User Recordings for Protocol Tunnel Jump, and Command Shell recordings. _This field only applies to PRA_
- `schedule_enabled` (Boolean) If true, users are restricted to accessing Jump Items within the scheduled hours. This setting cannot be enabled when require_approval is true.
- `schedule_strict` (Boolean) If true, users are forcefully removed from sessions when the schedule does not permit access. This can only be set to true if schedule_enabled is also true.
- `session_end_notification` (Boolean) If true, an email notification is sent to the configured recipients when a session ends. _This field only applies to PRA_
- `session_start_notification` (Boolean) If true, an email notification is sent to the configured recipients when a session starts. _This field only applies to PRA_
- `ticket_id_required` (Boolean) If true, users must enter a valid ticket ID that will be verified against the Ticket System configured on the Jump → Jump Policies page. This setting has no effect if a Ticket System is not configured.

Read-Only:

- `id` (String) The unique identifier assigned to this Jump Policy by the appliance.
