package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type JumpGroupModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CodeName   types.String `tfsdk:"code_name"`
	Comments   types.String `tfsdk:"comments"`
	EcmGroupId types.Int64  `tfsdk:"ecm_group_id"`
}

type JumpointModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	CodeName                  types.String `tfsdk:"code_name"`
	Platform                  types.String `tfsdk:"platform"`
	Comments                  types.String `tfsdk:"comments"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	Connected                 types.Bool   `tfsdk:"connected"`
	Clustered                 types.Bool   `tfsdk:"clustered"`
	ShellJumpEnabled          types.Bool   `tfsdk:"shell_jump_enabled"`
	ExternalJumpItemNetworkId types.String `tfsdk:"external_jump_item_network_id"`
	ProtocolTunnelEnabled     types.Bool   `tfsdk:"protocol_tunnel_enabled"`
	RdpServiceAccountId       types.Int64  `tfsdk:"rdp_service_account_id"`
}

type JumpItemRoleModel struct {
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

type SessionPolicyModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	CodeName    types.String `tfsdk:"code_name"`
	Description types.String `tfsdk:"description"`
}
