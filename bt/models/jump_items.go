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

type RemoteRDPModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	JumpointID       types.Int64  `tfsdk:"jumpoint_id"`
	Hostname         types.String `tfsdk:"hostname"`
	JumpGroupID      types.Int64  `tfsdk:"jump_group_id"`
	JumpGroupType    types.String `tfsdk:"jump_group_type"`
	Quality          types.String `tfsdk:"quality"`
	Console          types.Bool   `tfsdk:"console"`
	IgnoreUntrusted  types.Bool   `tfsdk:"ignore_untrusted"`
	Tag              types.String `tfsdk:"tag"`
	Comments         types.String `tfsdk:"comments"`
	RdpUsername      types.String `tfsdk:"rdp_username"`
	Domain           types.String `tfsdk:"domain"`
	JumpPolicyID     types.Int64  `tfsdk:"jump_policy_id"`
	SessionForensics types.Bool   `tfsdk:"session_forensics"`
	SessionPolicyID  types.Int64  `tfsdk:"session_policy_id"`
	EndpointID       types.Int64  `tfsdk:"endpoint_id"`
	SecureAppType    types.String `tfsdk:"secure_app_type"`
	RemoteAppName    types.String `tfsdk:"remote_app_name"`
	RemoteAppParams  types.String `tfsdk:"remote_app_params"`
	RemoteExePath    types.String `tfsdk:"remote_exe_path"`
	RemoteExeParams  types.String `tfsdk:"remote_exe_params"`
	TargetSystem     types.String `tfsdk:"target_system"`
	CredentialType   types.String `tfsdk:"credential_type"`
}

type RemoteVNCModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	JumpointID      types.Int64  `tfsdk:"jumpoint_id"`
	Hostname        types.String `tfsdk:"hostname"`
	Port            types.Int64  `tfsdk:"port"`
	JumpGroupID     types.Int64  `tfsdk:"jump_group_id"`
	JumpGroupType   types.String `tfsdk:"jump_group_type"`
	Tag             types.String `tfsdk:"tag"`
	Comments        types.String `tfsdk:"comments"`
	JumpPolicyID    types.Int64  `tfsdk:"jump_policy_id"`
	SessionPolicyID types.Int64  `tfsdk:"session_policy_id"`
}

type ProtocolTunnelJumpModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	JumpointID          types.Int64  `tfsdk:"jumpoint_id"`
	Hostname            types.String `tfsdk:"hostname"`
	JumpGroupID         types.Int64  `tfsdk:"jump_group_id"`
	JumpGroupType       types.String `tfsdk:"jump_group_type"`
	Tag                 types.String `tfsdk:"tag"`
	Comments            types.String `tfsdk:"comments"`
	JumpPolicyID        types.Int64  `tfsdk:"jump_policy_id"`
	SessionPolicyID     types.Int64  `tfsdk:"session_policy_id"`
	TunnelListenAddress types.String `tfsdk:"tunnel_listen_address"`
	TunnelDefinitions   types.String `tfsdk:"tunnel_definitions"`
	TunnelType          types.String `tfsdk:"tunnel_type"`
	Username            types.String `tfsdk:"username"`
	Database            types.String `tfsdk:"database"`
}

type WebJumpModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	JumpointID            types.Int64  `tfsdk:"jumpoint_id"`
	URL                   types.String `tfsdk:"url"`
	UsernameFormat        types.String `tfsdk:"username_format"`
	VerifyCertificate     types.Bool   `tfsdk:"verify_certificate"`
	JumpGroupID           types.Int64  `tfsdk:"jump_group_id"`
	JumpGroupType         types.String `tfsdk:"jump_group_type"`
	AuthenticationTimeout types.Int64  `tfsdk:"authentication_timeout"`
	Tag                   types.String `tfsdk:"tag"`
	Comments              types.String `tfsdk:"comments"`
	JumpPolicyID          types.Int64  `tfsdk:"jump_policy_id"`
	UsernameField         types.String `tfsdk:"username_field"`
	PasswordField         types.String `tfsdk:"password_field"`
	SubmitField           types.String `tfsdk:"submit_field"`
	SessionPolicyID       types.Int64  `tfsdk:"session_policy_id"`
}
