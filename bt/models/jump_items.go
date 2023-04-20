package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ShellJump struct {
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

type RemoteRDP struct {
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

type RemoteVNC struct {
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

type ProtocolTunnelJump struct {
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

type WebJump struct {
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

type JumpClientInstaller struct {
	ID                             types.String `tfsdk:"id"`
	JumpGroupID                    types.Int64  `tfsdk:"jump_group_id"`
	Name                           types.String `tfsdk:"name"`
	Tag                            types.String `tfsdk:"tag"`
	Comments                       types.String `tfsdk:"comments"`
	JumpPolicyID                   types.Int64  `tfsdk:"jump_policy_id"`
	ConnectionType                 types.String `tfsdk:"connection_type"`
	JumpGroupType                  types.String `tfsdk:"jump_group_type"`
	SessionPolicyID                types.Int64  `tfsdk:"session_policy_id"`
	MaxOfflineMinutes              types.Int64  `tfsdk:"max_offline_minutes"`
	InstallerID                    types.String `tfsdk:"installer_id"`
	KeyInfo                        types.String `tfsdk:"key_info"`
	ElevateInstall                 types.Bool   `tfsdk:"elevate_install"`
	ElevatePrompt                  types.Bool   `tfsdk:"elevate_prompt"`
	ExpirationTimestamp            types.String `tfsdk:"expiration_timestamp"`
	AllowOverrideJumpGroup         types.Bool   `tfsdk:"allow_override_jump_group"`
	AllowOverrideJumpPolicy        types.Bool   `tfsdk:"allow_override_jump_policy"`
	AllowOverrideName              types.Bool   `tfsdk:"allow_override_name"`
	AllowOverrideTag               types.Bool   `tfsdk:"allow_override_tag"`
	AllowOverrideComments          types.Bool   `tfsdk:"allow_override_comments"`
	AllowOverrideMaxOfflineMinutes types.Bool   `tfsdk:"allow_override_max_offline_minutes"`
	AllowOverrideSessionPolicy     types.Bool   `tfsdk:"allow_override_session_policy"`
	ValidDuration                  types.Int64  `tfsdk:"valid_duration" sra:"persist_state"`
}
