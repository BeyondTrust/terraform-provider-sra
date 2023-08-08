package bt

import (
	"context"
	"fmt"
	"os"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/ds"
	"terraform-provider-sra/bt/rs"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &sraProvider{}
)

func New() provider.Provider {
	return &sraProvider{}
}

type sraProvider struct{}

type sraProviderModel struct {
	Host         types.String `tfsdk:"host"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (p *sraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sra"
}

func (p *sraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
# BeyondTrust SRA Terraform Provider

The [BeyondTrust SRA Provider](https://registry.terraform.io/providers/beyondtrust/sra/latest/docs) allows [Terraform](https://terraform.io) to manage access to resources in the [Secure Remote Access (SRA)](https://www.beyondtrust.com/secure-remote-access) products from BeyondTrust.  This module can be used with either the Remote Support and Privileged Remote Access products to interact with the Configuration API using appropriately configured API credentials.

See the SRA Provider documentation as well as the Configuration API documentation in your instance for more information on supported API endpoints and parameters.

This provider requires Remote Support or Privileged Remote Access version 23.2.1+. Using this provider with prior versions is not supported by BeyondTrust and could result in Terraform reporting errors.

## Use Cases

The chief use case for this provider is to manage access to all assets managed within your Terraform instance in conjection with BeyondTrust Remote Support or BeyondTrust Priviliged Remote Access products.

As examples, this provider allows:
* Enabling Jump Item creation and deletion to match the provisioning and deprovisioning of assets within Terraform.
* Enabling Vault credential creation and deletion to match the credentials used within the assets within Terraform.
* Enabling Vault credential associations to Jump Items to enable passwordless authentication to assets.
* Enabling Vault credential policy management to control how credentials are handled and used.
* Enabling Jump Group creation, deletion, and asset membership to leverage existing SRA access controls.
* Enabling Group Policy associations to Jump Groups, Vault Accounts, Vault Account Groups to control overall user access to all Terraform assets.

Examples for all of these use cases can be found within the [test-tf-files](https://github.com/BeyondTrust/terraform-provider-sra/tree/main/test-tf-files) section of our Github repo.
q
## Configuration

To function, the provider requires the hostname of your instance as well as credentials for an API account configured in that instance. This API account must have permission to "Allow Access" to the Configuration API. If you also plan to access or manage Vault accounts with Terraform, then the API account also needs the "Manage Vault Accounts" permission.
To use the API Account within your Terraform scripts, the hostname, Client ID, and Client Secret values should be passed by setting the \"BT_API_HOST\", \"BT_CLIENT_ID\", and \"BT_CLIENT_SECRET\" environment variables which are the same environment settings used by the btapi CLI tool.  While not recommended, it is also possible to set the values within the script itself with the following block.`,
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "The SRA appliance hostname, such as mycompanyname.beyondtrustcloud.com",
			},
			"client_id": schema.StringAttribute{
				Description: "The SRA API Account OAuth Client ID",
				Optional:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The SRA API Account Client Secret",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *sraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring BeyondTrust SRA API client")

	var config sraProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("BT_API_HOST")
	client_id := os.Getenv("BT_CLIENT_ID")
	client_secret := os.Getenv("BT_CLIENT_SECRET")

	if !config.Host.IsNull() && !config.Host.IsUnknown() {
		host = config.Host.ValueString()
	}

	if !config.ClientId.IsNull() && !config.ClientId.IsUnknown() {
		client_id = config.ClientId.ValueString()
	}

	if !config.ClientSecret.IsNull() && !config.ClientSecret.IsUnknown() {
		client_secret = config.ClientSecret.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing BeyondTrust SRA API Host",
			"The provider cannot create the BeyondTrust SRA API client as there is a missing or empty value for the BeyondTrust SRA API host. "+
				"Set the host value in the configuration or use the BT_API_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if client_id == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Missing BeyondTrust SRA API Username",
			"The provider cannot create the BeyondTrust SRA API client as there is a missing or empty value for the BeyondTrust SRA API client_id. "+
				"Set the username value in the configuration or use the BT_CLIENT_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if client_secret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Missing BeyondTrust SRA API Password",
			"The provider cannot create the BeyondTrust SRA API client as there is a missing or empty value for the BeyondTrust SRA API client_secret. "+
				"Set the password value in the configuration or use the BT_CLIENT_SECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "bt_api_host", host)
	ctx = tflog.SetField(ctx, "bt_client_id", client_id)
	ctx = tflog.SetField(ctx, "bt_client_secret", client_secret)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "bt_client_secret")

	tflog.Debug(ctx, "Creating BT API Client")
	c, err := api.NewClient(host, &client_id, &client_secret)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create BeyondTrust SRA API Client",
			"An unexpected error occurred when creating the BeyondTrust SRA API Client"+
				"Error: "+err.Error(),
		)
	}

	mechs, err := api.Get[api.MechList](c)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to determine BeyondTrust Product",
			"An unexpected error occurred when querying the SRA Instance"+
				"Error: "+err.Error(),
		)
	}

	api.SetProductIsRS(mechs.IsRS())
	tflog.Info(ctx, fmt.Sprintf("Detected product is RS? [%v]", api.IsRS()))

	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured BT API client", map[string]any{"success": true})
}

func (p *sraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return ds.DatasourceList()
}

func (p *sraProvider) Resources(_ context.Context) []func() resource.Resource {
	return rs.ResourceList()
}
