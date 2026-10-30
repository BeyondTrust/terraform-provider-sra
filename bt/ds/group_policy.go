package ds

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &groupPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &groupPolicyDataSource{}
	_                                    = &groupPolicyDataSourceModel{}
)

func newGroupPolicyDataSource() datasource.DataSource {
	return &groupPolicyDataSource{}
}

type groupPolicyDataSource struct {
	apiDataSource[groupPolicyDataSourceModel, api.GroupPolicy, models.GroupPolicy]
}

type groupPolicyDataSourceModel struct {
	Items []models.GroupPolicy `tfsdk:"items"`
	Name  types.String         `tfsdk:"name" filter:"name"`
}

func (d *groupPolicyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Group Policies.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
							Required: false,
							Optional: false,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"perm_access_allowed": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to PRA",
						},
						"access_perm_status": schema.StringAttribute{
							Computed:    true,
							Description: "This field only applies to PRA",
						},
						"perm_support_allowed": schema.StringAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"rep_perm_status": schema.StringAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_generate_session_key": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_send_ios_profiles": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_accept_team_sessions": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_transfer_other_team": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_share_other_team": schema.BoolAttribute{
							Computed: true,
						},
						"perm_invite_external_user": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to PRA",
						},
						"perm_invite_external_rep": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_session_idle_timeout": schema.Int64Attribute{
							Computed: true,
						},
						"perm_next_session_button": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_extended_availability_mode_allowed": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_external_key": schema.BoolAttribute{
							Computed: true,
						},
						"perm_disable_auto_assignment": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_routing_idle_timeout": schema.Int64Attribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"auto_assignment_max_sessions": schema.Int64Attribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_collaborate": schema.BoolAttribute{
							Computed: true,
						},
						"perm_collaborate_control": schema.BoolAttribute{
							Computed: true,
						},
						"perm_support_button_personal_deploy": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_support_button_team_manage": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_support_button_change_public_sites": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_support_button_team_deploy": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_jump_client": schema.BoolAttribute{
							Computed: true,
						},
						"perm_local_jump": schema.BoolAttribute{
							Computed: true,
						},
						"perm_remote_jump": schema.BoolAttribute{
							Computed: true,
						},
						"perm_remote_vnc": schema.BoolAttribute{
							Computed: true,
						},
						"perm_remote_rdp": schema.BoolAttribute{
							Computed: true,
						},
						"perm_shell_jump": schema.BoolAttribute{
							Computed: true,
						},
						"perm_local_vnc": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_local_rdp": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_vpro": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
						"perm_web_jump": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to PRA",
						},
						"perm_protocol_tunnel": schema.BoolAttribute{
							Computed:    true,
							Description: "This field only applies to PRA",
						},
						"default_jump_item_role_id": schema.Int64Attribute{
							Computed: true,
						},
						"private_jump_item_role_id": schema.Int64Attribute{
							Computed: true,
						},
						"inferior_jump_item_role_id": schema.Int64Attribute{
							Computed: true,
						},
						"unassigned_jump_item_role_id": schema.Int64Attribute{
							Computed: true,
						},
						"perm_console_idle_timeout": schema.Int64Attribute{
							Computed:    true,
							Description: "This field only applies to RS",
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the group policy list for group policies matching \"name\"",
				Optional:    true,
			},
		},
	}
}
