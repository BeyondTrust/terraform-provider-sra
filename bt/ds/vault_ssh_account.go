package ds

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &vaultSSHAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &vaultSSHAccountDataSource{}
	_                                    = &vaultSSHAccountDataSourceModel{}
)

func newVaultSSHAccountDataSource() datasource.DataSource {
	return &vaultSSHAccountDataSource{}
}

type vaultSSHAccountDataSource struct {
	apiDataSource[vaultSSHAccountDataSourceModel, api.VaultSSHAccount, models.VaultSSHAccountDS]
}

type vaultSSHAccountDataSourceModel struct {
	Account         *models.VaultSSHAccountDS `tfsdk:"account"`
	ID              types.String              `tfsdk:"id"`
	Name            types.String              `tfsdk:"name" filter:"name"`
	IncludePersonal types.Bool                `tfsdk:"include_personal" filter:"include_personal"`
	AccountGroupID  types.Int64               `tfsdk:"account_group_id" filter:"account_group_id"`
}

func (d *vaultSSHAccountDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a single Vault SSH Account; useful for reading the public_key from an existing SSH account in Vault. If an ID is provided, that account will be returned. If an ID is not provided, the first account found with the specified filters will be returned.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"account": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
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
					"username": schema.StringAttribute{
						Required: true,
					},
					"public_key": schema.StringAttribute{
						Computed: true,
					},
					"private_key_public_cert": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"last_checkout_timestamp": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"id": schema.StringAttribute{
				Description: "Get the account with ID \"id\". If provided, no other filter options will be used.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Fetch the first SSH account matching \"name\"",
				Optional:    true,
			},
			"include_personal": schema.BoolAttribute{
				Description: "Set to 'true' to allows results to include personal accounts",
				Optional:    true,
			},
			"account_group_id": schema.Int64Attribute{
				Description: "Fetch the first SSH account in account group with id \"account_group_id\"",
				Optional:    true,
			},
		},
	}
}

func (d *vaultSSHAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_single_vault_ssh_account", req.ProviderTypeName)
	tflog.Debug(ctx, fmt.Sprintf("🥃 Registered datasource name [%s]", resp.TypeName))
}

func (d *vaultSSHAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vaultSSHAccountDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId basetypes.StringValue
	if state.ID.IsNull() || state.ID.IsUnknown() {
		filter := api.MakeFilterMap(ctx, state)
		tflog.Debug(ctx, "🙀 list with filter", map[string]interface{}{
			"data": filter,
		})

		items := d.doFilteredRead(ctx, req, resp, filter)
		if items == nil {
			return
		}

		tfId = items[0].ID
	} else {
		tfId, diags = state.ID.ToStringValue(ctx)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	id, _ := strconv.Atoi(tfId.ValueString())
	item, err := api.GetItem[api.VaultSSHAccount](d.apiClient, &id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ssh account",
			"Unexpected result reading ssh account with id ID ["+strconv.Itoa(id)+"]: "+err.Error(),
		)
		return
	}

	if item == nil {
		return
	}

	var account models.VaultSSHAccountDS
	tfObj := reflect.ValueOf(&account).Elem()
	apiType := reflect.TypeOf(item).Elem()
	apiObj := reflect.ValueOf(item).Elem()
	api.CopyAPItoTF(ctx, apiObj, tfObj, apiType)

	state.Account = &account

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
