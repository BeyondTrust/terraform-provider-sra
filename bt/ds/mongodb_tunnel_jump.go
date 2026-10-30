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
	_ datasource.DataSource              = &mongodbTunnelJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &mongodbTunnelJumpDataSource{}
	_                                    = &mongodbTunnelJumpDataSourceModel{}
)

func newMongodbTunnelJumpDataSource() datasource.DataSource { return &mongodbTunnelJumpDataSource{} }

type mongodbTunnelJumpDataSource struct {
	apiDataSource[mongodbTunnelJumpDataSourceModel, api.MongodbTunnelJump, models.MongodbTunnelJump]
}

type mongodbTunnelJumpDataSourceModel struct {
	Items         []models.MongodbTunnelJump `tfsdk:"items"`
	Name          types.String               `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64                `tfsdk:"jumpoint_id" filter:"jumpoint_id"`
	Hostname      types.String               `tfsdk:"hostname" filter:"hostname"`
	JumpGroupID   types.Int64                `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType types.String               `tfsdk:"jump_group_type" filter:"jump_group_type"`
	Tag           types.String               `tfsdk:"tag" filter:"tag"`
}

func (d *mongodbTunnelJumpDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of MongoDB Tunnel Jump Items. NOTE: PRA only. Requires PRA 26.1.1 or later.",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":                    schema.StringAttribute{Computed: true},
				"name":                  schema.StringAttribute{Computed: true},
				"jumpoint_id":           schema.Int64Attribute{Computed: true},
				"hostname":              schema.StringAttribute{Computed: true},
				"jump_group_id":         schema.Int64Attribute{Computed: true},
				"jump_group_type":       schema.StringAttribute{Computed: true},
				"tag":                   schema.StringAttribute{Computed: true},
				"comments":              schema.StringAttribute{Computed: true},
				"jump_policy_id":        schema.Int64Attribute{Computed: true},
				"session_policy_id":     schema.Int64Attribute{Computed: true},
				"tunnel_listen_address": schema.StringAttribute{Computed: true, Description: "Sent to and returned by the API as `local_address`."},
				"username":              schema.StringAttribute{Computed: true},
				"database":              schema.StringAttribute{Computed: true},
				"auth_database":         schema.StringAttribute{Computed: true, Description: "Empty before PRA 26.3.1, which does not return the field."},
			}}},
			"name":            schema.StringAttribute{Optional: true, Description: "Filter by name"},
			"jumpoint_id":     schema.Int64Attribute{Optional: true, Description: "Filter by jumpoint_id"},
			"hostname":        schema.StringAttribute{Optional: true, Description: "Filter by hostname"},
			"jump_group_id":   schema.Int64Attribute{Optional: true, Description: "Filter by jump_group_id"},
			"jump_group_type": schema.StringAttribute{Optional: true, Description: "Filter by jump_group_type"},
			"tag":             schema.StringAttribute{Optional: true, Description: "Filter by tag"},
		},
	}
}
