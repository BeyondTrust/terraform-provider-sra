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
	_ datasource.DataSource              = &vaultAccountPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &vaultAccountPolicyDataSource{}
	_                                    = &vaultAccountPolicyDataSourceModel{}
)

func newVaultAccountPolicyDataSource() datasource.DataSource {
	return &vaultAccountPolicyDataSource{}
}

type vaultAccountPolicyDataSource struct {
	apiDataSource[vaultAccountPolicyDataSourceModel, api.VaultAccountPolicy, models.VaultAccountPolicy]
}

type vaultAccountPolicyDataSourceModel struct {
	Items    []models.VaultAccountPolicy `tfsdk:"items"`
	Name     types.String                `tfsdk:"name" filter:"name"`
	CodeName types.String                `tfsdk:"code_name" filter:"code_name"`
}

func (d *vaultAccountPolicyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Vault Account Policies.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
						"code_name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"auto_rotate_credentials": schema.BoolAttribute{
							Computed: true,
						},
						"allow_simultaneous_checkout": schema.BoolAttribute{
							Computed: true,
						},
						"scheduled_password_rotation": schema.BoolAttribute{
							Computed: true,
						},
						"maximum_password_age": schema.Int64Attribute{
							Computed: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the list for items matching \"name\"",
				Optional:    true,
			},
			"code_name": schema.StringAttribute{
				Description: "Filter the list for items matching \"code_name\"",
				Optional:    true,
			},
		},
	}
}
