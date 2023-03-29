package api

// Models should be named like ResourceName. This name is mapped to snake_case for the
// name users will use in the terraform definitions for these resources
type ShellJump struct {
	ID              *int   `json:"id,omitempty"`
	Name            string `json:"name"`
	JumpointID      int    `json:"jumpoint_id"`
	Hostname        string `json:"hostname"`
	Protocol        string `json:"protocol"`
	JumpGroupID     int    `json:"jump_group_id"`
	JumpGroupType   string `json:"jump_group_type"`
	Port            int    `json:"port"`
	Terminal        string `json:"terminal"`
	KeepAlive       int    `json:"keep_alive"`
	Tag             string `json:"tag"`
	Comments        string `json:"comments"`
	Username        string `json:"username"`
	JumpPolicyID    *int   `json:"jump_policy_id,omitempty"`
	SessionPolicyID *int   `json:"session_policy_id,omitempty"`
}

func (ShellJump) endpoint() string {
	return "jump-item/shell-jump"
}

type RemoteRDP struct {
	ID               *int   `json:"id,omitempty"`
	Name             string `json:"name"`
	JumpointID       int    `json:"jumpoint_id"`
	Hostname         string `json:"hostname"`
	JumpGroupID      int    `json:"jump_group_id"`
	JumpGroupType    string `json:"jump_group_type"`
	Quality          string `json:"quality"`
	Console          bool   `json:"console"`
	IgnoreUntrusted  bool   `json:"ignore_untrusted"`
	Tag              string `json:"tag"`
	Comments         string `json:"comments"`
	RdpUsername      string `json:"rdp_username"`
	Domain           string `json:"domain"`
	SessionForensics bool   `json:"session_forensics"`
	SecureAppType    string `json:"secure_app_type"`
	RemoteAppName    string `json:"remote_app_name"`
	RemoteAppParams  string `json:"remote_app_params"`
	RemoteExePath    string `json:"remote_exe_path"`
	RemoteExeParams  string `json:"remote_exe_params"`
	TargetSystem     string `json:"target_system"`
	CredentialType   string `json:"credential_type"`
	EndpointID       *int   `json:"endpoint_id,omitempty"`
	JumpPolicyID     *int   `json:"jump_policy_id,omitempty"`
	SessionPolicyID  *int   `json:"session_policy_id,omitempty"`
}

func (RemoteRDP) endpoint() string {
	return "jump-item/remote-rdp"
}

type RemoteVNC struct {
	ID              *int   `json:"id,omitempty"`
	Name            string `json:"name"`
	JumpointID      int    `json:"jumpoint_id"`
	Hostname        string `json:"hostname"`
	JumpGroupID     int    `json:"jump_group_id"`
	JumpGroupType   string `json:"jump_group_type"`
	Port            int    `json:"port"`
	Tag             string `json:"tag"`
	Comments        string `json:"comments"`
	JumpPolicyID    *int   `json:"jump_policy_id,omitempty"`
	SessionPolicyID *int   `json:"session_policy_id,omitempty"`
}

func (RemoteVNC) endpoint() string {
	return "jump-item/remote-vnc"
}

type ProtocolTunnelJump struct {
	ID                  *int   `json:"id,omitempty"`
	Name                string `json:"name"`
	JumpointID          int    `json:"jumpoint_id"`
	Hostname            string `json:"hostname"`
	JumpGroupID         int    `json:"jump_group_id"`
	JumpGroupType       string `json:"jump_group_type"`
	Tag                 string `json:"tag"`
	Comments            string `json:"comments"`
	JumpPolicyID        *int   `json:"jump_policy_id,omitempty"`
	SessionPolicyID     *int   `json:"session_policy_id,omitempty"`
	TunnelListenAddress string `json:"tunnel_listen_address"`
	TunnelDefinitions   string `json:"tunnel_definitions"`
	TunnelType          string `json:"tunnel_type"`
	Username            string `json:"username"`
	Database            string `json:"database"`
}

func (ProtocolTunnelJump) endpoint() string {
	return "jump-item/protocol-tunnel-jump"
}

type WebJump struct {
	ID                    *int   `json:"id,omitempty"`
	Name                  string `json:"name"`
	JumpointID            int64  `json:"jumpoint_id"`
	URL                   string `json:"url"`
	UsernameFormat        string `json:"username_format"`
	VerifyCertificate     bool   `json:"verify_certificate"`
	JumpGroupID           int64  `json:"jump_group_id"`
	JumpGroupType         string `json:"jump_group_type"`
	AuthenticationTimeout int64  `json:"authentication_timeout"`
	Tag                   string `json:"tag"`
	Comments              string `json:"comments"`
	UsernameField         string `json:"username_field"`
	PasswordField         string `json:"password_field"`
	SubmitField           string `json:"submit_field"`
	JumpPolicyID          *int   `json:"jump_policy_id,omitempty"`
	SessionPolicyID       *int   `json:"session_policy_id,omitempty"`
}

func (WebJump) endpoint() string {
	return "jump-item/web-jump"
}

type JumpGroup struct {
	ID         *int   `json:"id,omitempty"`
	Name       string `json:"name"`
	CodeName   string `json:"code_name"`
	Comments   string `json:"comments"`
	EcmGroupID *int   `json:"ecm_group_id,omitempty"`
}

func (JumpGroup) endpoint() string {
	return "jump-group"
}

type Jumpoint struct {
	ID                        *int    `json:"id,omitempty"`
	Name                      string  `json:"name"`
	CodeName                  string  `json:"code_name"`
	Platform                  string  `json:"platform"`
	Comments                  string  `json:"comments"`
	Enabled                   bool    `json:"enabled"`
	Connected                 bool    `json:"connected"`
	Clustered                 bool    `json:"clustered"`
	ShellJumpEnabled          bool    `json:"shell_jump_enabled"`
	ExternalJumpItemNetworkID *string `json:"external_jump_item_network_id,omitempty"`
	ProtocolTunnelEnabled     bool    `json:"protocol_tunnel_enabled"`
	RdpServiceAccountID       *int    `json:"rdp_service_account_id"`
}

func (Jumpoint) endpoint() string {
	return "jumpoint"
}

type JumpItemRole struct {
	ID                     *int   `json:"id,omitempty"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	PermAdd                bool   `json:"perm_add"`
	PermAssignJumpGroup    bool   `json:"perm_assign_jump_group"`
	PermRemove             bool   `json:"perm_remove"`
	PermStart              bool   `json:"perm_start"`
	PermEditTag            bool   `json:"perm_edit_tag"`
	PermEditComments       bool   `json:"perm_edit_comments"`
	PermEditJumpPolicy     bool   `json:"perm_edit_jump_policy"`
	PermEditSessionPolicy  bool   `json:"perm_edit_session_policy"`
	PermEditIdentity       bool   `json:"perm_edit_identity"`
	PermEditBehavior       bool   `json:"perm_edit_behavior"`
	PermViewJumpItemReport bool   `json:"perm_view_jump_item_report"`
}

func (JumpItemRole) endpoint() string {
	return "jump-item-role"
}

type SessionPolicy struct {
	ID          *int   `json:"id,omitempty"`
	DisplayName string `json:"display_name"`
	CodeName    string `json:"code_name"`
	Description string `json:"description"`
}

func (SessionPolicy) endpoint() string {
	return "session-policy"
}

type GroupPolicy struct {
	ID                                  *int   `json:"id,omitempty"`
	Name                                string `json:"name"`
	PermAccessAllowed                   bool   `json:"perm_access_allowed"`
	AccessPermStatus                    string `json:"access_perm_status"`
	PermShareOtherTeam                  bool   `json:"perm_share_other_team"`
	PermInviteExternalUser              bool   `json:"perm_invite_external_user"`
	PermSessionIdleTimeout              int    `json:"perm_session_idle_timeout"`
	PermExtendedAvailabilityModeAllowed bool   `json:"perm_extended_availability_mode_allowed"`
	PermEditExternalKey                 bool   `json:"perm_edit_external_key"`
	PermCollaborate                     bool   `json:"perm_collaborate"`
	PermCollaborateControl              bool   `json:"perm_collaborate_control"`
	PermJumpClient                      bool   `json:"perm_jump_client"`
	PermLocalJump                       bool   `json:"perm_local_jump"`
	PermRemoteJump                      bool   `json:"perm_remote_jump"`
	PermRemoteVnc                       bool   `json:"perm_remote_vnc"`
	PermRemoteRdp                       bool   `json:"perm_remote_rdp"`
	PermShellJump                       bool   `json:"perm_shell_jump"`
	PermWebJump                         bool   `json:"perm_web_jump"`
	PermProtocolTunnel                  bool   `json:"perm_protocol_tunnel"`
	DefaultJumpItemRoleID               int    `json:"default_jump_item_role_id"`
	PrivateJumpItemRoleID               int    `json:"private_jump_item_role_id"`
	InferiorJumpItemRoleID              int    `json:"inferior_jump_item_role_id"`
	UnassignedJumpItemRoleID            int    `json:"unassigned_jump_item_role_id"`
}

func (GroupPolicy) endpoint() string {
	return "group-policy"
}
