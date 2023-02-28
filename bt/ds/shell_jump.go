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
	_ datasource.DataSource              = &shellJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &shellJumpDataSource{}
)

func newShellJumpDataSource() datasource.DataSource {
	return &shellJumpDataSource{}
}

type shellJumpDataSource struct {
	apiDataSource[shellJumpDataSourceModel, api.ShellJump, models.ShellJumpModel]
}

type shellJumpDataSourceModel struct {
	Items         []models.ShellJumpModel `tfsdk:"items"`
	Name          types.String            `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64             `tfsdk:"jumpoint_id" filter:"jumpoint_id"`
	Hostname      types.String            `tfsdk:"hostname" filter:"hostname"`
	JumpGroupID   types.Int64             `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType types.String            `tfsdk:"jump_group_type" filter:"jump_group_type"`
	Tag           types.String            `tfsdk:"tag" filter:"tag"`
}

func (d *shellJumpDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
						"jumpoint_id": schema.Int64Attribute{
							Required: true,
						},
						"hostname": schema.StringAttribute{
							Required: true,
						},
						"protocol": schema.StringAttribute{
							Required: true,
						},
						"port": schema.Int64Attribute{
							Optional: true,
						},
						"jump_group_id": schema.Int64Attribute{
							Required: true,
						},
						"jump_group_type": schema.StringAttribute{
							Required: true,
						},
						"terminal": schema.StringAttribute{
							Optional: true,
						},
						"keep_alive": schema.Int64Attribute{
							Optional: true,
						},
						"tag": schema.StringAttribute{
							Optional: true,
						},
						"comments": schema.StringAttribute{
							Optional: true,
						},
						"jump_policy_id": schema.Int64Attribute{
							Optional: true,
						},
						"username": schema.StringAttribute{
							Optional: true,
						},
						"session_policy_id": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Optional: true,
			},
			"jumpoint_id": schema.Int64Attribute{
				Optional: true,
			},
			"hostname": schema.StringAttribute{
				Optional: true,
			},
			"jump_group_id": schema.Int64Attribute{
				Optional: true,
			},
			"jump_group_type": schema.StringAttribute{
				Optional: true,
			},
			"tag": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}
