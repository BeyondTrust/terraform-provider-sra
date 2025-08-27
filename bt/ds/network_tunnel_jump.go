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
	_ datasource.DataSource              = &networkTunnelJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &networkTunnelJumpDataSource{}
	_                                    = &networkTunnelJumpDataSourceModel{}
)

func newNetworkTunnelJumpDataSource() datasource.DataSource { return &networkTunnelJumpDataSource{} }

type networkTunnelJumpDataSource struct {
	apiDataSource[networkTunnelJumpDataSourceModel, api.NetworkTunnelJump, models.NetworkTunnelJump]
}

type networkTunnelJumpDataSourceModel struct {
	Items         []models.NetworkTunnelJump `tfsdk:"items"`
	Name          types.String               `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64                `tfsdk:"jumpoint_id" filter:"jumpoint_id"`
	JumpGroupID   types.Int64                `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType types.String               `tfsdk:"jump_group_type" filter:"jump_group_type"`
	Tag           types.String               `tfsdk:"tag" filter:"tag"`
}

func (d *networkTunnelJumpDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Network Tunnel Jump Items. NOTE: PRA only.",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":                schema.StringAttribute{Computed: true},
				"name":              schema.StringAttribute{Required: true},
				"jumpoint_id":       schema.Int64Attribute{Required: true},
				"jump_group_id":     schema.Int64Attribute{Required: true},
				"jump_group_type":   schema.StringAttribute{Optional: true, Computed: true},
				"tag":               schema.StringAttribute{Optional: true, Computed: true},
				"comments":          schema.StringAttribute{Optional: true, Computed: true},
				"jump_policy_id":    schema.Int64Attribute{Optional: true},
				"session_policy_id": schema.Int64Attribute{Optional: true},
				"filter_rules": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"ip_addresses": schema.ListAttribute{ElementType: types.StringType, Required: true},
					"ports":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true},
					"protocol":     schema.StringAttribute{Optional: true, Computed: true},
				}}},
			}}},
			"name":            schema.StringAttribute{Optional: true, Description: "Filter by name"},
			"jumpoint_id":     schema.Int64Attribute{Optional: true, Description: "Filter by jumpoint_id"},
			"jump_group_id":   schema.Int64Attribute{Optional: true, Description: "Filter by jump_group_id"},
			"jump_group_type": schema.StringAttribute{Optional: true, Description: "Filter by jump_group_type"},
			"tag":             schema.StringAttribute{Optional: true, Description: "Filter by tag"},
		},
	}
}
