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
	ID              *int    `json:"id,omitempty"`
	Name            string  `json:"name"`
	JumpointID      int     `json:"jumpoint_id"`
	Hostname        string  `json:"hostname"`
	JumpGroupID     int     `json:"jump_group_id"`
	JumpGroupType   string  `json:"jump_group_type"`
	Quality         string  `json:"quality"`
	Console         bool    `json:"console"`
	IgnoreUntrusted bool    `json:"ignore_untrusted"`
	Tag             string  `json:"tag"`
	Comments        string  `json:"comments"`
	RdpUsername     string  `json:"rdp_username"`
	Domain          string  `json:"domain"`
	SecureAppType   *string `json:"secure_app_type,omitempty"`
	RemoteAppName   *string `json:"remote_app_name,omitempty"`
	RemoteAppParams *string `json:"remote_app_params,omitempty"`
	RemoteExePath   *string `json:"remote_exe_path,omitempty"`
	RemoteExeParams *string `json:"remote_exe_params,omitempty"`
	TargetSystem    *string `json:"target_system,omitempty"`
	CredentialType  *string `json:"credential_type,omitempty"`
	EndpointID      *int    `json:"endpoint_id,omitempty"`
	JumpPolicyID    *int    `json:"jump_policy_id,omitempty"`
	SessionPolicyID *int    `json:"session_policy_id,omitempty"`

	SessionForensics *bool `json:"session_forensics,omitempty" sraproduct:"pra"`
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

func (ProtocolTunnelJump) AllowPRA() bool {
	return true
}

func (ProtocolTunnelJump) AllowRS() bool {
	return false
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

func (WebJump) AllowPRA() bool {
	return true
}

func (WebJump) AllowRS() bool {
	return false
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

	ProtocolTunnelEnabled *bool `json:"protocol_tunnel_enabled,omitempty" sraproduct:"pra"`
	RdpServiceAccountID   *int  `json:"rdp_service_account_id,omitempty" sraproduct:"pra"`

	GroupPolicyMemberships []GroupPolicyJumpoint `json:"-" sraapi:"skip"`
}

func (Jumpoint) Endpoint() string {
	return "jumpoint"
}

type JumpClientInstaller struct {
	ID                             *int      `json:"id,omitempty"`
	JumpGroupID                    int       `json:"jump_group_id"`
	Name                           string    `json:"name"`
	Tag                            string    `json:"tag"`
	Comments                       string    `json:"comments"`
	ConnectionType                 string    `json:"connection_type"`
	JumpGroupType                  string    `json:"jump_group_type"`
	JumpPolicyID                   *int      `json:"jump_policy_id,omitempty"`
	MaxOfflineMinutes              int       `json:"max_offline_minutes"`
	InstallerID                    string    `json:"installer_id,omitempty"`
	KeyInfo                        string    `json:"key_info,omitempty"`
	ElevateInstall                 bool      `json:"elevate_install"`
	ElevatePrompt                  bool      `json:"elevate_prompt"`
	ExpirationTimestamp            Timestamp `json:"expiration_timestamp,omitempty"`
	AllowOverrideJumpGroup         bool      `json:"allow_override_jump_group"`
	AllowOverrideJumpPolicy        bool      `json:"allow_override_jump_policy"`
	AllowOverrideName              bool      `json:"allow_override_name"`
	AllowOverrideTag               bool      `json:"allow_override_tag"`
	AllowOverrideComments          bool      `json:"allow_override_comments"`
	AllowOverrideMaxOfflineMinutes bool      `json:"allow_override_max_offline_minutes"`
	ValidDuration                  *int      `json:"valid_duration,omitempty"`

	SessionPolicyID            *int  `json:"session_policy_id,omitempty" sraproduct:"pra"`
	AllowOverrideSessionPolicy *bool `json:"allow_override_session_policy,omitempty" sraproduct:"pra"`

	AttendedSessionPolicyID              *int  `json:"attended_session_policy_id,omitempty" sraproduct:"rs"`
	UnattendedSessionPolicyID            *int  `json:"unattended_session_policy_id,omitempty" sraproduct:"rs"`
	IsQuiet                              *bool `json:"is_quiet,omitempty" sraproduct:"rs"`
	AllowOverrideAttendedSessionPolicy   *bool `json:"allow_override_attended_session_policy,omitempty" sraproduct:"rs"`
	AllowOverrideUnattendedSessionPolicy *bool `json:"allow_override_unattended_session_policy,omitempty" sraproduct:"rs"`
}

func (JumpClientInstaller) Endpoint() string {
	return "jump-client/installer"
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

type JumpPolicy struct {
	ID               *int   `json:"id,omitempty"`
	DisplayName      string `json:"display_name"`
	CodeName         string `json:"code_name"`
	Description      string `json:"description"`
	ScheduleEnabled  bool   `json:"schedule_enabled"`
	ScheduleStrict   bool   `json:"schedule_strict"`
	TicketIdRequired bool   `json:"ticket_id_required"`

	SessionStartNotification   *bool     `json:"session_start_notification,omitempty" sraproduct:"pra"`
	SessionEndNotification     *bool     `json:"session_end_notification,omitempty" sraproduct:"pra"`
	NotificationEmailAddresses *[]string `json:"notification_email_addresses,omitempty" sraproduct:"pra"`
	NotificationDisplayName    *string   `json:"notification_display_name,omitempty" sraproduct:"pra"`
	NotificationEmailLanguage  *string   `json:"notification_email_language,omitempty" sraproduct:"pra"`
	ApprovalRequired           *bool     `json:"approval_required,omitempty" sraproduct:"pra"`
	ApprovalMaxDuration        *int      `json:"approval_max_duration,omitempty" sraproduct:"pra"`
	ApprovalScope              *string   `json:"approval_scope,omitempty" sraproduct:"pra"`
	ApprovalEmailAddresses     *[]string `json:"approval_email_addresses,omitempty" sraproduct:"pra"`
	ApprovalUserIds            *[]string `json:"approval_user_ids,omitempty" sraproduct:"pra"`
	ApprovalDisplayName        *string   `json:"approval_display_name,omitempty" sraproduct:"pra"`
	ApprovalEmailLanguage      *string   `json:"approval_email_language,omitempty" sraproduct:"pra"`
	RecordingsDisabled         *bool     `json:"recordings_disabled,omitempty" sraproduct:"pra"`
}

func (JumpPolicy) Endpoint() string {
	return "jump-policy"
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
	PermShareOtherTeam                  bool   `json:"perm_share_other_team"`
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
	DefaultJumpItemRoleID               int    `json:"default_jump_item_role_id"`
	PrivateJumpItemRoleID               int    `json:"private_jump_item_role_id"`
	InferiorJumpItemRoleID              int    `json:"inferior_jump_item_role_id"`
	UnassignedJumpItemRoleID            int    `json:"unassigned_jump_item_role_id"`

	PermAccessAllowed      *bool   `json:"perm_access_allowed,omitempty" sraproduct:"pra"`
	AccessPermStatus       *string `json:"access_perm_status,omitempty" sraproduct:"pra"`
	PermInviteExternalUser *bool   `json:"perm_invite_external_user,omitempty" sraproduct:"pra"`
	PermWebJump            *bool   `json:"perm_web_jump,omitempty" sraproduct:"pra"`
	PermProtocolTunnel     *bool   `json:"perm_protocol_tunnel,omitempty" sraproduct:"pra"`

	PermSupportAllowed                 *bool   `json:"perm_support_allowed,omitempty" sraproduct:"rs"`
	RepPermStatus                      *string `json:"rep_perm_status,omitempty" sraproduct:"rs"`
	PermGenerateSessionKey             *bool   `json:"perm_generate_session_key,omitempty" sraproduct:"rs"`
	PermSendIosProfiles                *bool   `json:"perm_send_ios_profiles,omitempty" sraproduct:"rs"`
	PermAcceptTeamSessions             *bool   `json:"perm_accept_team_sessions,omitempty" sraproduct:"rs"`
	PermTransferOtherTeam              *bool   `json:"perm_transfer_other_team,omitempty" sraproduct:"rs"`
	PermInviteExternalRep              *bool   `json:"perm_invite_external_rep,omitempty" sraproduct:"rs"`
	PermNextSessionButton              *bool   `json:"perm_next_session_button,omitempty" sraproduct:"rs"`
	PermDisableAutoAssignment          *bool   `json:"perm_disable_auto_assignment,omitempty" sraproduct:"rs"`
	PermRoutingIdleTimeout             *int    `json:"perm_routing_idle_timeout,omitempty" sraproduct:"rs"`
	AutoAssignmentMaxSessions          *int    `json:"auto_assignment_max_sessions,omitempty" sraproduct:"rs"`
	PermSupportButtonPersonalDeploy    *bool   `json:"perm_support_button_personal_deploy,omitempty" sraproduct:"rs"`
	PermSupportButtonTeamManage        *bool   `json:"perm_support_button_team_manage,omitempty" sraproduct:"rs"`
	PermSupportButtonChangePublicSites *bool   `json:"perm_support_button_change_public_sites,omitempty" sraproduct:"rs"`
	PermSupportButtonTeamDeploy        *bool   `json:"perm_support_button_team_deploy,omitempty" sraproduct:"rs"`
	PermLocalVNC                       *bool   `json:"perm_local_vnc,omitempty" sraproduct:"rs"`
	PermLocalRDP                       *bool   `json:"perm_local_rdp,omitempty" sraproduct:"rs"`
	PermVpro                           *bool   `json:"perm_vpro,omitempty" sraproduct:"rs"`
	PermConsoleIdleTimeout             *int    `json:"perm_console_idle_timeout,omitempty" sraproduct:"rs"`
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
	AccountPolicy  *string `json:"account_policy,omitempty"`

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

type GroupPolicyProvision struct {
	GroupPolicyID *string `tfsdk:"group_policy_id" json:"-"`
}

func (a GroupPolicyProvision) Endpoint() string {
	return fmt.Sprintf("group-policy/%s/provision", *a.GroupPolicyID)
}

type GroupPolicyJumpGroup struct {
	GroupPolicyID  *string `tfsdk:"group_policy_id" json:"-"`
	JumpGroupID    *int    `tfsdk:"-" json:"jump_group_id"`
	JumpItemRoleID int     `tfsdk:"jump_item_role_id" json:"jump_item_role_id"`
	JumpPolicyID   *int    `tfsdk:"jump_policy_id" json:"jump_policy_id" sraproduct:"pra"`
}

func (a GroupPolicyJumpGroup) Endpoint() string {
	return fmt.Sprintf("group-policy/%s/jump-group", *a.GroupPolicyID)
}

type GroupPolicyJumpoint struct {
	GroupPolicyID *string `tfsdk:"group_policy_id" json:"-"`
	JumpointID    *int    `tfsdk:"-" json:"jumpoint_id"`
}

func (a GroupPolicyJumpoint) Endpoint() string {
	return fmt.Sprintf("group-policy/%s/jumpoint", *a.GroupPolicyID)
}

type MechList struct {
	Mechs       []string   `json:"mechs"`
	DefaultMech string     `json:"default_mech"`
	Cache       ConfigBool `json:"cache"`
	Company     string     `json:"company"`
	Product     string     `json:"product"`
}

func (a MechList) Endpoint() string {
	return "get_mech_list?version=3"
}

func (a *MechList) IsPRA() bool {
	return a.Product == "bpam"
}

func (a *MechList) IsRS() bool {
	return !a.IsPRA()
}

type VaultSecret struct {
	ID         *int    `json:"id,omitempty"`
	Username   string  `json:"username"`
	Type       string  `json:"type"`
	Password   *string `json:"password,omitempty"`
	PrivateKey *string `json:"private_key,omitempty"`
	Secret     *string `json:"-"`
}

func (a VaultSecret) Endpoint() string {
	return fmt.Sprintf("vault/account/%d", *a.ID)
}
