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
	_ datasource.DataSource              = &jumpGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &jumpGroupDataSource{}
	_                                    = &jumpGroupDataSourceModel{}
)

func newJumpGroupDataSource() datasource.DataSource {
	return &jumpGroupDataSource{}
}

type jumpGroupDataSource struct {
	apiDataSource[jumpGroupDataSourceModel, api.JumpGroup, models.JumpGroupDS]
}

type jumpGroupDataSourceModel struct {
	Items    []models.JumpGroupDS `tfsdk:"items"`
	Name     types.String         `tfsdk:"name" filter:"name"`
	CodeName types.String         `tfsdk:"code_name" filter:"code_name"`
}

func (d *jumpGroupDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Jump Groups.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the Jump Group list for groups matching \"name\"",
				Optional:    true,
			},
			"code_name": schema.StringAttribute{
				Description: "Filter the Jump Group list for groups with a matching \"code_name\"",
				Optional:    true,
			},
		},
	}
}
