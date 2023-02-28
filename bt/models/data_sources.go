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
