package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type JumpGroup struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	CodeName types.String `tfsdk:"code_name"`
	Comments types.String `tfsdk:"comments"`

	GroupPolicyMemberships types.Set `tfsdk:"group_policy_memberships"`
}
type GroupPolicyJumpGroup struct {
	GroupPolicyID  types.String `tfsdk:"group_policy_id"`
	JumpItemRoleID types.Int64  `tfsdk:"jump_item_role_id"`
	JumpPolicyID   types.Int64  `tfsdk:"jump_policy_id" sraproduct:"pra"`
}

type JumpGroupDS struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	CodeName types.String `tfsdk:"code_name"`
	Comments types.String `tfsdk:"comments"`
}

type Jumpoint struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	CodeName                  types.String `tfsdk:"code_name"`
	Platform                  types.String `tfsdk:"platform"`
	Comments                  types.String `tfsdk:"comments"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	Connected                 types.Bool   `tfsdk:"connected"`
	Clustered                 types.Bool   `tfsdk:"clustered"`
	ShellJumpEnabled          types.Bool   `tfsdk:"shell_jump_enabled"`
	ExternalJumpItemNetworkID types.String `tfsdk:"external_jump_item_network_id"`
	ProtocolTunnelEnabled     types.Bool   `tfsdk:"protocol_tunnel_enabled" sraproduct:"pra"`
	RdpServiceAccountID       types.Int64  `tfsdk:"rdp_service_account_id" sraproduct:"pra"`

	GroupPolicyMemberships types.Set `tfsdk:"group_policy_memberships"`
}

type JumpointDS struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	CodeName                  types.String `tfsdk:"code_name"`
	Platform                  types.String `tfsdk:"platform"`
	Comments                  types.String `tfsdk:"comments"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	Connected                 types.Bool   `tfsdk:"connected"`
	Clustered                 types.Bool   `tfsdk:"clustered"`
	ShellJumpEnabled          types.Bool   `tfsdk:"shell_jump_enabled"`
	ExternalJumpItemNetworkID types.String `tfsdk:"external_jump_item_network_id"`
	ProtocolTunnelEnabled     types.Bool   `tfsdk:"protocol_tunnel_enabled" sraproduct:"pra"`
	RdpServiceAccountID       types.Int64  `tfsdk:"rdp_service_account_id" sraproduct:"pra"`
}

type JumpItemRole struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	PermAdd                types.Bool   `tfsdk:"perm_add"`
	PermAssignJumpGroup    types.Bool   `tfsdk:"perm_assign_jump_group"`
	PermRemove             types.Bool   `tfsdk:"perm_remove"`
	PermStart              types.Bool   `tfsdk:"perm_start"`
	PermEditTag            types.Bool   `tfsdk:"perm_edit_tag"`
	PermEditComments       types.Bool   `tfsdk:"perm_edit_comments"`
	PermEditJumpPolicy     types.Bool   `tfsdk:"perm_edit_jump_policy"`
	PermEditSessionPolicy  types.Bool   `tfsdk:"perm_edit_session_policy"`
	PermEditIdentity       types.Bool   `tfsdk:"perm_edit_identity"`
	PermEditBehavior       types.Bool   `tfsdk:"perm_edit_behavior"`
	PermViewJumpItemReport types.Bool   `tfsdk:"perm_view_jump_item_report"`
}

type SessionPolicy struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	CodeName    types.String `tfsdk:"code_name"`
	Description types.String `tfsdk:"description"`
}

type GroupPolicy struct {
	ID                                  types.String `tfsdk:"id"`
	Name                                types.String `tfsdk:"name"`
	PermShareOtherTeam                  types.Bool   `tfsdk:"perm_share_other_team"`
	PermSessionIdleTimeout              types.Int64  `tfsdk:"perm_session_idle_timeout"`
	PermExtendedAvailabilityModeAllowed types.Bool   `tfsdk:"perm_extended_availability_mode_allowed"`
	PermEditExternalKey                 types.Bool   `tfsdk:"perm_edit_external_key"`
	PermCollaborate                     types.Bool   `tfsdk:"perm_collaborate"`
	PermCollaborateControl              types.Bool   `tfsdk:"perm_collaborate_control"`
	PermJumpClient                      types.Bool   `tfsdk:"perm_jump_client"`
	PermLocalJump                       types.Bool   `tfsdk:"perm_local_jump"`
	PermRemoteJump                      types.Bool   `tfsdk:"perm_remote_jump"`
	PermRemoteVnc                       types.Bool   `tfsdk:"perm_remote_vnc"`
	PermRemoteRdp                       types.Bool   `tfsdk:"perm_remote_rdp"`
	PermShellJump                       types.Bool   `tfsdk:"perm_shell_jump"`
	DefaultJumpItemRoleID               types.Int64  `tfsdk:"default_jump_item_role_id"`
	PrivateJumpItemRoleID               types.Int64  `tfsdk:"private_jump_item_role_id"`
	InferiorJumpItemRoleID              types.Int64  `tfsdk:"inferior_jump_item_role_id"`
	UnassignedJumpItemRoleID            types.Int64  `tfsdk:"unassigned_jump_item_role_id"`

	PermAccessAllowed      types.Bool   `tfsdk:"perm_access_allowed" sraproduct:"pra"`
	AccessPermStatus       types.String `tfsdk:"access_perm_status" sraproduct:"pra"`
	PermInviteExternalUser types.Bool   `tfsdk:"perm_invite_external_user" sraproduct:"pra"`
	PermWebJump            types.Bool   `tfsdk:"perm_web_jump" sraproduct:"pra"`
	PermProtocolTunnel     types.Bool   `tfsdk:"perm_protocol_tunnel" sraproduct:"pra"`

	PermSupportAllowed                 types.String `tfsdk:"perm_support_allowed" sraproduct:"rs"`
	RepPermStatus                      types.String `tfsdk:"rep_perm_status" sraproduct:"rs"`
	PermGenerateSessionKey             types.Bool   `tfsdk:"perm_generate_session_key" sraproduct:"rs"`
	PermSendIosProfiles                types.Bool   `tfsdk:"perm_send_ios_profiles" sraproduct:"rs"`
	PermAcceptTeamSessions             types.Bool   `tfsdk:"perm_accept_team_sessions" sraproduct:"rs"`
	PermTransferOtherTeam              types.Bool   `tfsdk:"perm_transfer_other_team" sraproduct:"rs"`
	PermInviteExternalRep              types.Bool   `tfsdk:"perm_invite_external_rep" sraproduct:"rs"`
	PermNextSessionButton              types.Bool   `tfsdk:"perm_next_session_button" sraproduct:"rs"`
	PermDisableAutoAssignment          types.Bool   `tfsdk:"perm_disable_auto_assignment" sraproduct:"rs"`
	PermRoutingIdleTimeout             types.Int64  `tfsdk:"perm_routing_idle_timeout" sraproduct:"rs"`
	AutoAssignmentMaxSessions          types.Int64  `tfsdk:"auto_assignment_max_sessions" sraproduct:"rs"`
	PermSupportButtonPersonalDeploy    types.Bool   `tfsdk:"perm_support_button_personal_deploy" sraproduct:"rs"`
	PermSupportButtonTeamManage        types.Bool   `tfsdk:"perm_support_button_team_manage" sraproduct:"rs"`
	PermSupportButtonChangePublicSites types.Bool   `tfsdk:"perm_support_button_change_public_sites" sraproduct:"rs"`
	PermSupportButtonTeamDeploy        types.Bool   `tfsdk:"perm_support_button_team_deploy" sraproduct:"rs"`
	PermLocalVNC                       types.Bool   `tfsdk:"perm_local_vnc" sraproduct:"rs"`
	PermLocalRDP                       types.Bool   `tfsdk:"perm_local_rdp" sraproduct:"rs"`
	PermVpro                           types.Bool   `tfsdk:"perm_vpro" sraproduct:"rs"`
	PermConsoleIdleTimeout             types.Int64  `tfsdk:"perm_console_idle_timeout" sraproduct:"rs"`
}

type JumpPolicy struct {
	ID               types.String `tfsdk:"id"`
	DisplayName      types.String `tfsdk:"display_name"`
	CodeName         types.String `tfsdk:"code_name"`
	Description      types.String `tfsdk:"description"`
	ScheduleEnabled  types.Bool   `tfsdk:"schedule_enabled"`
	ScheduleStrict   types.Bool   `tfsdk:"schedule_strict"`
	TicketIdRequired types.Bool   `tfsdk:"ticket_id_required"`

	SessionStartNotification   types.Bool   `tfsdk:"session_start_notification" sraproduct:"pra"`
	SessionEndNotification     types.Bool   `tfsdk:"session_end_notification" sraproduct:"pra"`
	NotificationEmailAddresses types.Set    `tfsdk:"notification_email_addresses" sraproduct:"pra"`
	NotificationDisplayName    types.String `tfsdk:"notification_display_name" sraproduct:"pra"`
	NotificationEmailLanguage  types.String `tfsdk:"notification_email_language" sraproduct:"pra"`
	ApprovalRequired           types.Bool   `tfsdk:"approval_required" sraproduct:"pra"`
	ApprovalMaxDuration        types.Int64  `tfsdk:"approval_max_duration" sraproduct:"pra"`
	ApprovalScope              types.String `tfsdk:"approval_scope" sraproduct:"pra"`
	ApprovalEmailAddresses     types.Set    `tfsdk:"approval_email_addresses" sraproduct:"pra"`
	ApprovalUserIds            types.Set    `tfsdk:"approval_user_ids" sraproduct:"pra"`
	ApprovalDisplayName        types.String `tfsdk:"approval_display_name" sraproduct:"pra"`
	ApprovalEmailLanguage      types.String `tfsdk:"approval_email_language" sraproduct:"pra"`
	RecordingsDisabled         types.Bool   `tfsdk:"recordings_disabled" sraproduct:"pra"`
}
