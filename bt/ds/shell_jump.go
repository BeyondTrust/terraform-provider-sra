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
	_ datasource.DataSource              = &shellJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &shellJumpDataSource{}
	_                                    = &shellJumpDataSourceModel{}
)

func newShellJumpDataSource() datasource.DataSource {
	return &shellJumpDataSource{}
}

type shellJumpDataSource struct {
	apiDataSource[shellJumpDataSourceModel, api.ShellJump, models.ShellJump]
}

type shellJumpDataSourceModel struct {
	Items         []models.ShellJump `tfsdk:"items"`
	Name          types.String       `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64        `tfsdk:"jumpoint_id" filter:"jumpoint_id"`
	Hostname      types.String       `tfsdk:"hostname" filter:"hostname"`
	JumpGroupID   types.Int64        `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType types.String       `tfsdk:"jump_group_type" filter:"jump_group_type"`
	Tag           types.String       `tfsdk:"tag" filter:"tag"`
}

func (d *shellJumpDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Shell Jump Items.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
						"jumpoint_id": schema.Int64Attribute{
							Computed: true,
						},
						"hostname": schema.StringAttribute{
							Computed: true,
						},
						"protocol": schema.StringAttribute{
							Computed: true,
						},
						"port": schema.Int64Attribute{
							Computed: true,
						},
						"jump_group_id": schema.Int64Attribute{
							Computed: true,
						},
						"jump_group_type": schema.StringAttribute{
							Computed: true,
						},
						"terminal": schema.StringAttribute{
							Computed: true,
						},
						"keep_alive": schema.Int64Attribute{
							Computed: true,
						},
						"tag": schema.StringAttribute{
							Computed: true,
						},
						"comments": schema.StringAttribute{
							Computed: true,
						},
						"jump_policy_id": schema.Int64Attribute{
							Computed: true,
						},
						"username": schema.StringAttribute{
							Computed: true,
						},
						"session_policy_id": schema.Int64Attribute{
							Computed: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the list for items matching \"name\"",
				Optional:    true,
			},
			"jumpoint_id": schema.Int64Attribute{
				Description: "Filter the list for items with a matching \"jumpoint_id\"",
				Optional:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"hostname\"",
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
		},
	}
}
