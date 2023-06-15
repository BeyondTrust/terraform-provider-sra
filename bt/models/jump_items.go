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
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	JumpointID      types.Int64  `tfsdk:"jumpoint_id"`
	Hostname        types.String `tfsdk:"hostname"`
	JumpGroupID     types.Int64  `tfsdk:"jump_group_id"`
	JumpGroupType   types.String `tfsdk:"jump_group_type"`
	Quality         types.String `tfsdk:"quality"`
	Console         types.Bool   `tfsdk:"console"`
	IgnoreUntrusted types.Bool   `tfsdk:"ignore_untrusted"`
	Tag             types.String `tfsdk:"tag"`
	Comments        types.String `tfsdk:"comments"`
	RdpUsername     types.String `tfsdk:"rdp_username"`
	Domain          types.String `tfsdk:"domain"`
	JumpPolicyID    types.Int64  `tfsdk:"jump_policy_id"`
	SessionPolicyID types.Int64  `tfsdk:"session_policy_id"`
	EndpointID      types.Int64  `tfsdk:"endpoint_id"`

	SecureAppType    types.String `tfsdk:"secure_app_type" sraproduct:"pra"`
	RemoteAppName    types.String `tfsdk:"remote_app_name" sraproduct:"pra"`
	RemoteAppParams  types.String `tfsdk:"remote_app_params" sraproduct:"pra"`
	RemoteExePath    types.String `tfsdk:"remote_exe_path" sraproduct:"pra"`
	RemoteExeParams  types.String `tfsdk:"remote_exe_params" sraproduct:"pra"`
	TargetSystem     types.String `tfsdk:"target_system" sraproduct:"pra"`
	CredentialType   types.String `tfsdk:"credential_type" sraproduct:"pra"`
	SessionForensics types.Bool   `tfsdk:"session_forensics" sraproduct:"pra"`
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
	ValidDuration                  types.Int64  `tfsdk:"valid_duration" sra:"persist_state"`

	SessionPolicyID            types.Int64 `tfsdk:"session_policy_id" sraproduct:"pra"`
	AllowOverrideSessionPolicy types.Bool  `tfsdk:"allow_override_session_policy" sraproduct:"pra"`

	IsQuiet                              types.Bool  `tfsdk:"is_quiet"`
	AttendedSessionPolicyID              types.Int64 `tfsdk:"attended_session_policy_id" sraproduct:"rs"`
	UnattendedSessionPolicyID            types.Int64 `tfsdk:"unattended_session_policy_id" sraproduct:"rs"`
	AllowOverrideAttendedSessionPolicy   types.Bool  `tfsdk:"allow_override_attended_session_policy" sraproduct:"rs"`
	AllowOverrideUnattendedSessionPolicy types.Bool  `tfsdk:"allow_override_unattended_session_policy" sraproduct:"rs"`
}
