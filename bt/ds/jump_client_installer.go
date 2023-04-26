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
	_ datasource.DataSource              = &jumpClientInstallerDataSource{}
	_ datasource.DataSourceWithConfigure = &jumpClientInstallerDataSource{}
	_                                    = &jumpClientInstallerDataSourceModel{}
)

func newJumpClientInstallerDataSource() datasource.DataSource {
	return &jumpClientInstallerDataSource{}
}

type jumpClientInstallerDataSource struct {
	apiDataSource[jumpClientInstallerDataSourceModel, api.JumpClientInstaller, models.JumpClientInstaller]
}

type jumpClientInstallerDataSourceModel struct {
	Items          []models.JumpClientInstaller `tfsdk:"items"`
	Name           types.String                 `tfsdk:"name" filter:"name"`
	Tag            types.String                 `tfsdk:"tag" filter:"tag"`
	JumpGroupID    types.Int64                  `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType  types.String                 `tfsdk:"jump_group_type" filter:"jump_group_type"`
	ConnectionType types.String                 `tfsdk:"connection_type" filter:"connection_type"`
}

func (d *jumpClientInstallerDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Jump Client Installers.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"valid_duration": schema.Int64Attribute{
							Optional: true,
						},
						"name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"jump_group_id": schema.Int64Attribute{
							Required: true,
						},
						"jump_group_type": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"tag": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"comments": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"jump_policy_id": schema.Int64Attribute{
							Optional: true,
						},
						"session_policy_id": schema.Int64Attribute{
							Optional: true,
						},
						"connection_type": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"expiration_timestamp": schema.StringAttribute{
							Computed: true,
						},
						"max_offline_minutes": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
						"elevate_install": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"elevate_prompt": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_jump_policy": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_jump_group": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_name": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_tag": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_comments": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_max_offline_minutes": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"allow_override_session_policy": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"installer_id": schema.StringAttribute{
							Computed: true,
						},
						"key_info": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the list for items matching \"name\"",
				Optional:    true,
			},
			"jump_group_id": schema.Int64Attribute{
				Description: "Filter the list for items with a matching \"jump_group_id\"",
				Optional:    true,
			},
			"jump_group_type": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"jump_group_type\"",
				Optional:    true,
			},
			"tag": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"tag\"",
				Optional:    true,
			},
			"connection_type": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"connection_type\". Should be either 'active' or 'passive'",
				Optional:    true,
			},
		},
	}
}