package api

import (
	"fmt"
)

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

func (ShellJump) Endpoint() string {
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

func (RemoteRDP) Endpoint() string {
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

func (RemoteVNC) Endpoint() string {
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
	TunnelDefinitions   string `json:"tunnel_definitions,omitempty"`
	TunnelType          string `json:"tunnel_type"`
	Username            string `json:"username,omitempty"`
	Database            string `json:"database,omitempty"`
}

func (ProtocolTunnelJump) Endpoint() string {
	return "jump-item/protocol-tunnel-jump"
}

type WebJump struct {
	ID                    *int   `json:"id,omitempty"`
	Name                  string `json:"name"`
	JumpointID            int    `json:"jumpoint_id"`
	URL                   string `json:"url"`
	UsernameFormat        string `json:"username_format"`
	VerifyCertificate     bool   `json:"verify_certificate"`
	JumpGroupID           int    `json:"jump_group_id"`
	JumpGroupType         string `json:"jump_group_type"`
	AuthenticationTimeout int    `json:"authentication_timeout"`
	Tag                   string `json:"tag"`
	Comments              string `json:"comments"`
	UsernameField         string `json:"username_field"`
	PasswordField         string `json:"password_field"`
	SubmitField           string `json:"submit_field"`
	JumpPolicyID          *int   `json:"jump_policy_id,omitempty"`
	SessionPolicyID       *int   `json:"session_policy_id,omitempty"`
}

func (WebJump) Endpoint() string {
	return "jump-item/web-jump"
}

type JumpGroup struct {
	ID       *int   `json:"id,omitempty"`
	Name     string `json:"name"`
	CodeName string `json:"code_name"`
	Comments string `json:"comments"`

	GroupPolicyMemberships []GroupPolicyJumpGroup `json:"-" sraapi:"skip"`
}

func (JumpGroup) Endpoint() string {
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

func (Jumpoint) Endpoint() string {
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

func (JumpItemRole) Endpoint() string {
	return "jump-item-role"
}

type SessionPolicy struct {
	ID          *int   `json:"id,omitempty"`
	DisplayName string `json:"display_name"`
	CodeName    string `json:"code_name"`
	Description string `json:"description"`
}

func (SessionPolicy) Endpoint() string {
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

func (GroupPolicy) Endpoint() string {
	return "group-policy"
}

type VaultAccount struct {
	ID             *int    `json:"id,omitempty"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Personal       bool    `json:"personal"`
	OwnerUserID    *int    `json:"owner_user_id,omitempty"`
	AccountGroupID int     `json:"account_group_id"`
	AccountPolicy  *string `json:"account_policy"`
}

func (VaultAccount) Endpoint() string {
	return "vault/account"
}

type VaultUsernamePasswordAccount struct {
	ID             *int    `json:"id,omitempty"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Personal       *bool   `json:"personal,omitempty"`
	OwnerUserID    *int    `json:"owner_user_id,omitempty"`
	AccountGroupID int     `json:"account_group_id"`
	AccountPolicy  *string `json:"account_policy"`

	Username              string  `json:"username"`
	Password              string  `json:"password,omitempty"`
	LastCheckoutTimestamp *string `json:"last_checkout_timestamp"`

	JumpItemAssociation    AccountJumpItemAssociation `json:"-" sraapi:"skip"`
	GroupPolicyMemberships []GroupPolicyVaultAccount  `json:"-" sraapi:"skip"`
}

func (VaultUsernamePasswordAccount) Endpoint() string {
	return "vault/account"
}

type VaultSSHAccount struct {
	ID             *int    `json:"id,omitempty"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Personal       *bool   `json:"personal,omitempty"`
	OwnerUserID    *int    `json:"owner_user_id,omitempty"`
	AccountGroupID int     `json:"account_group_id"`
	AccountPolicy  *string `json:"account_policy"`

	Username              string  `json:"username"`
	PublicKey             *string `json:"public_key,omitempty"`
	PrivateKey            string  `json:"private_key"`
	PrivateKeyPassphrase  string  `json:"private_key_passphrase"`
	PrivateKeyPublicCert  string  `json:"private_key_public_cert"`
	LastCheckoutTimestamp *string `json:"last_checkout_timestamp"`

	JumpItemAssociation    AccountJumpItemAssociation `json:"-" sraapi:"skip"`
	GroupPolicyMemberships []GroupPolicyVaultAccount  `json:"-" sraapi:"skip"`
}

func (VaultSSHAccount) Endpoint() string {
	return "vault/account"
}

type VaultAccountGroup struct {
	ID            *int    `json:"id,omitempty"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	AccountPolicy *string `json:"account_policy,omitempty"`

	JumpItemAssociation    AccountJumpItemAssociation     `json:"-" sraapi:"skip"`
	GroupPolicyMemberships []GroupPolicyVaultAccountGroup `json:"-" sraapi:"skip"`
}

func (VaultAccountGroup) Endpoint() string {
	return "vault/account-group"
}

type VaultAccountPolicy struct {
	ID                        *int   `json:"id,omitempty"`
	Name                      string `json:"name"`
	CodeName                  string `json:"code_name"`
	Description               string `json:"description"`
	AutoRotateCredentials     bool   `json:"auto_rotate_credentials"`
	AllowSimultaneousCheckout bool   `json:"allow_simultaneous_checkout"`
	ScheduledPasswordRotation bool   `json:"scheduled_password_rotation"`
	MaximumPasswordAge        *int   `json:"maximum_password_age"`
}

func (VaultAccountPolicy) Endpoint() string {
	return "vault/account-policy"
}

// These models have to follow some stricter rules about type conversion, though
// that's largely defined in the schema of the resource. These are tagged to
// read/write from TF Schema/Plans directly, meaning unknown or null values
// could panic, depending on the type of the field.
type AccountJumpItemAssociation struct {
	ID         *int                         `tfsdk:"-" json:"-"`
	FilterType string                       `json:"filter_type" tfsdk:"filter_type"`
	Criteria   *JumpItemAssociationCriteria `json:"criteria" tfsdk:"criteria"`
	JumpItems  []InjectableJumpItem         `json:"jump_items" tfsdk:"jump_items"`
}

func (a AccountJumpItemAssociation) Endpoint() string {
	return fmt.Sprintf("vault/account/%d/jump-item-association", *a.ID)
}

type AccountGroupJumpItemAssociation struct {
	ID         *int                         `tfsdk:"-" json:"-"`
	FilterType string                       `json:"filter_type" tfsdk:"filter_type"`
	Criteria   *JumpItemAssociationCriteria `json:"criteria" tfsdk:"criteria"`
	JumpItems  []InjectableJumpItem         `json:"jump_items" tfsdk:"jump_items"`
}

func (a AccountGroupJumpItemAssociation) Endpoint() string {
	return fmt.Sprintf("vault/account-group/%d/jump-item-association", *a.ID)
}

type JumpItemAssociationCriteria struct {
	SharedJumpGroups []int    `json:"shared_jump_groups" tfsdk:"shared_jump_groups"`
	Host             []string `json:"host" tfsdk:"host"`
	Name             []string `json:"name" tfsdk:"name"`
	Tag              []string `json:"tag" tfsdk:"tag"`
	Comment          []string `json:"comment" tfsdk:"comment"`
}

type InjectableJumpItem struct {
	ID   int    `json:"id" tfsdk:"id"`
	Type string `json:"type" tfsdk:"type"`
}

type GroupPolicyVaultAccountGroup struct {
	GroupPolicyID  *string `tfsdk:"group_policy_id" json:"-"`
	AccountGroupID *int    `tfsdk:"-" json:"account_group_id"`
	Role           string  `tfsdk:"role" json:"role"`
}

func (a GroupPolicyVaultAccountGroup) Endpoint() string {
	return fmt.Sprintf("group-policy/%s/vault-account-group", *a.GroupPolicyID)
}

type GroupPolicyVaultAccount struct {
	GroupPolicyID *string `tfsdk:"group_policy_id" json:"-"`
	AccountID     *int    `tfsdk:"-" json:"account_id"`
	Role          string  `tfsdk:"role" json:"role"`
}

func (a GroupPolicyVaultAccount) Endpoint() string {
	return fmt.Sprintf("group-policy/%s/vault-account", *a.GroupPolicyID)
}

type GroupPolicyJumpGroup struct {
	GroupPolicyID  *string `tfsdk:"group_policy_id" json:"-"`
	JumpGroupID    *int    `tfsdk:"-" json:"jump_group_id"`
	JumpItemRoleID int     `tfsdk:"jump_item_role_id" json:"jump_item_role_id"`
	JumpPolicyID   int     `tfsdk:"jump_policy_id" json:"jump_policy_id"`
}

func (a GroupPolicyJumpGroup) Endpoint() string {
	return fmt.Sprintf("group-policy/%s/jump-group", *a.GroupPolicyID)
}
