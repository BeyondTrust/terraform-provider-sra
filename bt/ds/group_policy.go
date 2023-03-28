package ds

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &groupPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &groupPolicyDataSource{}
)

func newGroupPolicyDataSource() datasource.DataSource {
	return &groupPolicyDataSource{}
}

type groupPolicyDataSource struct {
	apiDataSource[groupPolicyDataSourceModel, api.GroupPolicy, models.GroupPolicyModel]
}

type groupPolicyDataSourceModel struct {
	Items []models.GroupPolicyModel `tfsdk:"items"`
	Name  types.String              `tfsdk:"name" filter:"name"`
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
							Required: true,
						},
						"perm_access_allowed": schema.BoolAttribute{
							Optional: true,
						},
						"access_perm_status": schema.StringAttribute{
							Optional: true,
						},
						"perm_share_other_team": schema.BoolAttribute{
							Optional: true,
						},
						"perm_invite_external_user": schema.BoolAttribute{
							Optional: true,
						},
						"perm_session_idle_timeout": schema.Int64Attribute{
							Optional: true,
						},
						"perm_extended_availability_mode_allowed": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_external_key": schema.BoolAttribute{
							Optional: true,
						},
						"perm_collaborate": schema.BoolAttribute{
							Optional: true,
						},
						"perm_collaborate_control": schema.BoolAttribute{
							Optional: true,
						},
						"perm_jump_client": schema.BoolAttribute{
							Optional: true,
						},
						"perm_local_jump": schema.BoolAttribute{
							Optional: true,
						},
						"perm_remote_jump": schema.BoolAttribute{
							Optional: true,
						},
						"perm_remote_vnc": schema.BoolAttribute{
							Optional: true,
						},
						"perm_remote_rdp": schema.BoolAttribute{
							Optional: true,
						},
						"perm_shell_jump": schema.BoolAttribute{
							Optional: true,
						},
						"perm_web_jump": schema.BoolAttribute{
							Optional: true,
						},
						"perm_protocol_tunnel": schema.BoolAttribute{
							Optional: true,
						},
						"default_jump_item_role_id": schema.Int64Attribute{
							Optional: true,
						},
						"private_jump_item_role_id": schema.Int64Attribute{
							Optional: true,
						},
						"inferior_jump_item_role_id": schema.Int64Attribute{
							Optional: true,
						},
						"unassigned_jump_item_role_id": schema.Int64Attribute{
							Optional: true,
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
