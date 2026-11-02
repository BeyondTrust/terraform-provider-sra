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
	_ datasource.DataSource              = &oracleTunnelJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &oracleTunnelJumpDataSource{}
	_                                    = &oracleTunnelJumpDataSourceModel{}
)

func newOracleTunnelJumpDataSource() datasource.DataSource { return &oracleTunnelJumpDataSource{} }

type oracleTunnelJumpDataSource struct {
	apiDataSource[oracleTunnelJumpDataSourceModel, api.OracleTunnelJump, models.OracleTunnelJump]
}

// The filter tags are the Oracle endpoint's query parameter names, which differ
// from the attribute names; the endpoint rejects the standard names.
type oracleTunnelJumpDataSourceModel struct {
	Items         []models.OracleTunnelJump `tfsdk:"items"`
	Name          types.String              `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64               `tfsdk:"jumpoint_id" filter:"gateway_id"`
	Hostname      types.String              `tfsdk:"hostname" filter:"hostname"`
	JumpGroupID   types.Int64               `tfsdk:"jump_group_id" filter:"asset_group_id"`
	JumpGroupType types.String              `tfsdk:"jump_group_type" filter:"asset_group_type"`
	Tag           types.String              `tfsdk:"tag" filter:"tag"`
}

func (d *oracleTunnelJumpDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Oracle Tunnel Jump Items. NOTE: PRA only. Requires PRA 26.3.1 or later.",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":                    schema.StringAttribute{Computed: true},
				"name":                  schema.StringAttribute{Computed: true},
				"jumpoint_id":           schema.Int64Attribute{Computed: true, Description: "Returned by the API as `gateway_id`."},
				"hostname":              schema.StringAttribute{Computed: true},
				"jump_group_id":         schema.Int64Attribute{Computed: true, Description: "Returned by the API as `asset_group_id`."},
				"jump_group_type":       schema.StringAttribute{Computed: true, Description: "Returned by the API as `asset_group_type`."},
				"tag":                   schema.StringAttribute{Computed: true},
				"comments":              schema.StringAttribute{Computed: true},
				"jump_policy_id":        schema.Int64Attribute{Computed: true},
				"session_policy_id":     schema.Int64Attribute{Computed: true},
				"tunnel_listen_address": schema.StringAttribute{Computed: true, Description: "Returned by the API as `local_address`."},
				"username":              schema.StringAttribute{Computed: true},
				"database":              schema.StringAttribute{Computed: true},
			}}},
			"name":            schema.StringAttribute{Optional: true, Description: "Filter by name"},
			"jumpoint_id":     schema.Int64Attribute{Optional: true, Description: "Filter by jumpoint_id (sent as `gateway_id`)"},
			"hostname":        schema.StringAttribute{Optional: true, Description: "Filter by hostname"},
			"jump_group_id":   schema.Int64Attribute{Optional: true, Description: "Filter by jump_group_id (sent as `asset_group_id`)"},
			"jump_group_type": schema.StringAttribute{Optional: true, Description: "Filter by jump_group_type (sent as `asset_group_type`)"},
			"tag":             schema.StringAttribute{Optional: true, Description: "Filter by tag"},
		},
	}
}
