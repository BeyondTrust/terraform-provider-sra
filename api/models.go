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

type JumpGroup struct {
	ID         *int   `json:"id,omitempty"`
	Name       string `json:"name"`
	CodeName   string `json:"code_name"`
	Comments   string `json:"comments"`
	EcmGroupId *int   `json:"ecm_group_id,omitempty"`
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
	ExternalJumpItemNetworkId *string `json:"external_jump_item_network_id,omitempty"`
	ProtocolTunnelEnabled     bool    `json:"protocol_tunnel_enabled"`
	RdpServiceAccountId       *int    `json:"rdp_service_account_id"`
}

func (Jumpoint) endpoint() string {
	return "jumpoint"
}

type JumpItemRole struct {
	ID                     *int   `json:"id"`
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
