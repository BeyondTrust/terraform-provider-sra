package ds

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &vaultSecretDataSource{}
	_ datasource.DataSourceWithConfigure = &vaultSecretDataSource{}
	_                                    = &vaultSecretDataSourceModel{}
)

func newVaultSecretDataSource() datasource.DataSource {
	return &vaultSecretDataSource{}
}

type vaultSecretDataSource struct {
	apiDataSource[vaultSecretDataSourceModel, api.VaultSecret, models.VaultSecret]
}

type vaultSecretDataSourceModel struct {
	Account *models.VaultSecret `tfsdk:"account"`
	ID      types.String        `tfsdk:"id"`
}

func (d *vaultSecretDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Fetch the secret data for any Vault Account given the account ID.

NOTE: The API account being used must have permission to check out the account with the provided ID, and the account must be able to be checked out. If the account cannot be checked out, this data source will produce an error.

This data source will check out and then check in the account with the given ID. It is not recommended to use this data source with accounts that rotate upon check in.

For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance`,
		Attributes: map[string]schema.Attribute{
			"account": schema.SingleNestedAttribute{
				Computed: true,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "The type of the account that was retrieved",
					},
					"username": schema.StringAttribute{
						Computed:    true,
						Description: "The data stored in the username field in Vault.",
					},
					"secret": schema.StringAttribute{
						Computed:    true,
						Sensitive:   true,
						Description: "The secret data stored in Vault.",
					},
					"signed_public_cert": schema.StringAttribute{
						Computed:    true,
						Sensitive:   true,
						Description: "The signed public cert for a ssh or ssh_ca secret, if one exists.",
					},
				},
			},
			"id": schema.StringAttribute{
				Description: "Get the secret for the account with ID \"id\". If provided, no other filter options will be used.",
				Required:    true,
			},
		},
	}
}

func (d *vaultSecretDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_vault_secret", req.ProviderTypeName)
	tflog.Debug(ctx, fmt.Sprintf("🥃 Registered datasource name [%s]", resp.TypeName))
}

func (d *vaultSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vaultSecretDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.Diagnostics.AddError(
			"ID is required",
			"You must provide an account ID to use this data source",
		)
		return
	}

	tfId, diags := state.ID.ToStringValue(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, _ := strconv.Atoi(tfId.ValueString())
	item := &api.VaultSecret{}
	item.ID = &id
	tflog.Info(ctx, "🥮 checking out account", map[string]interface{}{
		"id":    id,
		"state": fmt.Sprintf("%+v", state),
		"item":  fmt.Sprintf("%+v", item),
	})
	item, err := api.Post(d.apiClient, "check-out", *item, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error checking out account",
			"Error checkout out the account with id ID ["+strconv.Itoa(id)+"]. Please ensure the API account can checkout this item.\n"+err.Error(),
		)
		return
	}

	if item == nil {
		return
	}

	var account models.VaultSecret
	account.ID = types.StringValue(strconv.Itoa(*item.ID))
	account.Username = types.StringValue(item.Username)
	account.Type = types.StringValue(item.Type)

	if item.Type == "ssh" || item.Type == "ssh_ca" {
		account.Secret = types.StringValue(*item.PrivateKey)
	} else {
		account.Secret = types.StringValue(*item.Password)
	}

	if item.SignedPublicCert != nil {
		account.SignedPublicCert = types.StringValue(*item.SignedPublicCert)
	} else {
		account.SignedPublicCert = types.StringNull()
	}

	state.Account = &account

	_, err = api.Post(d.apiClient, "check-in", *item, true)
	// If checking it back in isn't allowed… just ignore that
	if err != nil && !strings.HasPrefix(err.Error(), "status: 422") {
		resp.Diagnostics.AddError(
			"Error checking in account",
			"Error checking in the account with id ID ["+strconv.Itoa(id)+"]. Please ensure the account can be checked in.\n"+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
