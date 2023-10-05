package ds

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &vaultAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &vaultAccountDataSource{}
	_                                    = &vaultAccountDataSourceModel{}
)

func newVaultAccountDataSource() datasource.DataSource {
	return &vaultAccountDataSource{}
}

type vaultAccountDataSource struct {
	apiDataSource[vaultAccountDataSourceModel, api.VaultAccount, models.VaultAccount]
}

type vaultAccountDataSourceModel struct {
	Items           []models.VaultAccount `tfsdk:"items"`
	Name            types.String          `tfsdk:"name" filter:"name"`
	Type            types.String          `tfsdk:"type" filter:"type"`
	IncludePersonal types.Bool            `tfsdk:"include_personal" filter:"include_personal"`
	AccountGroupID  types.Int64           `tfsdk:"account_group_id" filter:"account_group_id"`
	EndpointID      types.Int64           `tfsdk:"endpoint_id" filter:"endpoint_id"`
}

func (d *vaultAccountDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Vault Accounts.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"personal": schema.BoolAttribute{
							Computed: true,
						},
						"owner_user_id": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
						"account_group_id": schema.Int64Attribute{
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
			"type": schema.StringAttribute{
				Description: "Filter the list for items matching \"name\"",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"username_password", "ssh", "ssh_ca", "windows_local", "windows_domain"}...),
				},
			},
			"include_personal": schema.BoolAttribute{
				Description: "Set to 'true' to allows results to include personal accounts",
				Optional:    true,
			},
			"account_group_id": schema.Int64Attribute{
				Description: "Filter the list for items in account group with id \"account_group_id\"",
				Optional:    true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Filters results to include only Windows Local accounts with the given Endpoint",
				Optional:    true,
			},
		},
	}
}
