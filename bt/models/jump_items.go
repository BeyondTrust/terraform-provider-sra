package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ShellJumpModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	JumpointID      types.Int64  `tfsdk:"jumpoint_id"`
	Hostname        types.String `tfsdk:"hostname"`
	Protocol        types.String `tfsdk:"protocol"`
	Port            types.Int64  `tfsdk:"port"`
	JumpGroupID     types.Int64  `tfsdk:"jump_group_id"`
	JumpGroupType   types.String `tfsdk:"jump_group_type"`
	Terminal        types.String `tfsdk:"terminal"`
	KeepAlive       types.Int64  `tfsdk:"keep_alive"`
	Tag             types.String `tfsdk:"tag"`
	Comments        types.String `tfsdk:"comments"`
	JumpPolicyID    types.Int64  `tfsdk:"jump_policy_id"`
	Username        types.String `tfsdk:"username"`
	SessionPolicyID types.Int64  `tfsdk:"session_policy_id"`
}
