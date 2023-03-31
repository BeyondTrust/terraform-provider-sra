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
	_ datasource.DataSource              = &vaultAccountGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &vaultAccountGroupDataSource{}
	_                                    = &vaultAccountGroupDataSourceModel{}
)

func newVaultAccountGroupDataSource() datasource.DataSource {
	return &vaultAccountGroupDataSource{}
}

type vaultAccountGroupDataSource struct {
	apiDataSource[vaultAccountGroupDataSourceModel, api.VaultAccountGroup, models.VaultAccountGroup]
}

type vaultAccountGroupDataSourceModel struct {
	Items []models.VaultAccountGroup `tfsdk:"items"`
	Name  types.String               `tfsdk:"name" filter:"name"`
}

func (d *vaultAccountGroupDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Vault Account Groups.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"account_policy": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the list for items matching \"name\"",
				Optional:    true,
			},
		},
	}
}
