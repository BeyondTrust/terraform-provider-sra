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
	ProtocolTunnelEnabled     types.Bool   `tfsdk:"protocol_tunnel_enabled"`
	RdpServiceAccountID       types.Int64  `tfsdk:"rdp_service_account_id"`
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
	PermAccessAllowed                   types.Bool   `tfsdk:"perm_access_allowed"`
	AccessPermStatus                    types.String `tfsdk:"access_perm_status"`
	PermShareOtherTeam                  types.Bool   `tfsdk:"perm_share_other_team"`
	PermInviteExternalUser              types.Bool   `tfsdk:"perm_invite_external_user"`
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
	PermWebJump                         types.Bool   `tfsdk:"perm_web_jump"`
	PermProtocolTunnel                  types.Bool   `tfsdk:"perm_protocol_tunnel"`
	DefaultJumpItemRoleID               types.Int64  `tfsdk:"default_jump_item_role_id"`
	PrivateJumpItemRoleID               types.Int64  `tfsdk:"private_jump_item_role_id"`
	InferiorJumpItemRoleID              types.Int64  `tfsdk:"inferior_jump_item_role_id"`
	UnassignedJumpItemRoleID            types.Int64  `tfsdk:"unassigned_jump_item_role_id"`
}
