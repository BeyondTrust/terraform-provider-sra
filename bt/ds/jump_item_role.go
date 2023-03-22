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
	_ datasource.DataSource              = &jumpItemRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &jumpItemRoleDataSource{}
)

func newJumpItemRoleDataSource() datasource.DataSource {
	return &jumpItemRoleDataSource{}
}

type jumpItemRoleDataSource struct {
	apiDataSource[jumpItemRoleDataSourceModel, api.JumpItemRole, models.JumpItemRoleModel]
}

type jumpItemRoleDataSourceModel struct {
	Items    []models.JumpItemRoleModel `tfsdk:"items"`
	Name     types.String            `tfsdk:"name" filter:"name"`
	CodeName types.String            `tfsdk:"code_name" filter:"code_name"`
}

func (d *jumpItemRoleDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
						"ecm_group_id": schema.Int64Attribute{
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
		},
	}
}
