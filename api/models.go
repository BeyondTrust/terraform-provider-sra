package api

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

func (_ ShellJump) endpoint() string {
	return "jump-item/shell-jump"
}
