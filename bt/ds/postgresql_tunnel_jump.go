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
	_ datasource.DataSource              = &postgresqlTunnelJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &postgresqlTunnelJumpDataSource{}
	_                                    = &postgresqlTunnelJumpDataSourceModel{}
)

func newPostgreSQLTunnelJumpDataSource() datasource.DataSource {
	return &postgresqlTunnelJumpDataSource{}
}

type postgresqlTunnelJumpDataSource struct {
	apiDataSource[postgresqlTunnelJumpDataSourceModel, api.PostgreSQLTunnelJump, models.PostgreSQLTunnelJump]
}

type postgresqlTunnelJumpDataSourceModel struct {
	Items         []models.PostgreSQLTunnelJump `tfsdk:"items"`
	Name          types.String                  `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64                   `tfsdk:"jumpoint_id" filter:"jumpoint_id"`
	Hostname      types.String                  `tfsdk:"hostname" filter:"hostname"`
	JumpGroupID   types.Int64                   `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType types.String                  `tfsdk:"jump_group_type" filter:"jump_group_type"`
	Tag           types.String                  `tfsdk:"tag" filter:"tag"`
}

func (d *postgresqlTunnelJumpDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of PostgreSQL Tunnel Jump Items. NOTE: PRA only.",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":                    schema.StringAttribute{Computed: true},
				"name":                  schema.StringAttribute{Required: true},
				"jumpoint_id":           schema.Int64Attribute{Required: true},
				"hostname":              schema.StringAttribute{Required: true},
				"jump_group_id":         schema.Int64Attribute{Required: true},
				"jump_group_type":       schema.StringAttribute{Optional: true, Computed: true},
				"tag":                   schema.StringAttribute{Optional: true, Computed: true},
				"comments":              schema.StringAttribute{Optional: true, Computed: true},
				"jump_policy_id":        schema.Int64Attribute{Optional: true},
				"session_policy_id":     schema.Int64Attribute{Optional: true},
				"tunnel_listen_address": schema.StringAttribute{Optional: true, Computed: true},
				"username":              schema.StringAttribute{Optional: true, Computed: true},
				"database":              schema.StringAttribute{Optional: true, Computed: true},
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
