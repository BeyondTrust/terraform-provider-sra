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
	_ datasource.DataSource              = &jumpointDataSource{}
	_ datasource.DataSourceWithConfigure = &jumpointDataSource{}
)

func newJumpointDataSource() datasource.DataSource {
	return &jumpointDataSource{}
}

type jumpointDataSource struct {
	apiDataSource[jumpointDataSourceModel, api.Jumpoint, models.JumpointModel]
}

type jumpointDataSourceModel struct {
	Items     []models.JumpointModel `tfsdk:"items"`
	Name      types.String           `tfsdk:"name" filter:"name"`
	CodeName  types.String           `tfsdk:"code_name" filter:"code_name"`
	PublicIp  types.String           `tfsdk:"public_ip" filter:"public_ip"`
	PrivateIp types.String           `tfsdk:"private_ip" filter:"private_ip"`
	Hostname  types.String           `tfsdk:"hostname" filter:"hostname"`
}

func (d *jumpointDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
						"code_name": schema.StringAttribute{
							Required: true,
						},
						"comments": schema.StringAttribute{
							Optional: true,
						},
						"platform": schema.StringAttribute{
							Required: true,
						},
						"enabled": schema.BoolAttribute{
							Optional: true,
						},
						"connected": schema.BoolAttribute{
							Optional: true,
						},
						"clustered": schema.BoolAttribute{
							Optional: true,
						},
						"shell_jump_enabled": schema.BoolAttribute{
							Optional: true,
						},
						"external_jump_item_network_id": schema.StringAttribute{
							Optional: true,
						},
						"protocol_tunnel_enabled": schema.BoolAttribute{
							Optional: true,
						},
						"rdp_service_account_id": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Optional: true,
			},
			"code_name": schema.StringAttribute{
				Optional: true,
			},
			"public_ip": schema.StringAttribute{
				Optional: true,
			},
			"private_ip": schema.StringAttribute{
				Optional: true,
			},
			"hostname": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}
