package ds

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = &sessionPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &sessionPolicyDataSource{}
	_                                    = &sessionPolicyDataSourceModel{}
)

func newSessionPolicyDataSource() datasource.DataSource {
	return &sessionPolicyDataSource{}
}

type sessionPolicyDataSource struct {
	apiDataSource[sessionPolicyDataSourceModel, api.SessionPolicy, models.SessionPolicy]
}

type sessionPolicyDataSourceModel struct {
	Items []models.SessionPolicy `tfsdk:"items"`
}

func (d *sessionPolicyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Session Policies.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
						"display_name": schema.StringAttribute{
							Required: true,
						},
						"code_name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}
