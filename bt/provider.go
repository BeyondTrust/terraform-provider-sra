package bt

import (
	"context"
	"os"

	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/ds"
	"terraform-provider-beyondtrust-sra/bt/rs"

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
	resp.TypeName = "bt"
}

func (p *sraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"client_id": schema.StringAttribute{
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *sraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring BeyondTrust SRA API client")
	var config sraProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown BeyondTrust SRA Appliance Address",
			"The provider cannot create the BeyondTrust SRA API client as there is an unknown configuration value for the BeyondTrust SRA Appliance Address. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BT_API_HOST environment variable.",
		)
	}

	if config.ClientId.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Unknown BeyondTrust SRA API Client ID",
			"The provider cannot create the BeyondTrust SRA API client as there is an unknown configuration value for the BeyondTrust SRA API client_id. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BT_CLIENT_ID environment variable.",
		)
	}

	if config.ClientSecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Unknown BeyondTrust SRA API Client Secret",
			"The provider cannot create the BeyondTrust SRA API client as there is an unknown configuration value for the BeyondTrust SRA API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BT_CLIENT_SECRET environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("BT_API_HOST")
	client_id := os.Getenv("BT_CLIENT_ID")
	client_secret := os.Getenv("BT_CLIENT_SECRET")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.ClientId.IsNull() {
		client_id = config.ClientId.ValueString()
	}

	if !config.ClientSecret.IsNull() {
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

	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured BT API client", map[string]any{"success": true})
}

func (p *sraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		ds.NewShellJumpDataSource,
	}
}

func (p *sraProvider) Resources(_ context.Context) []func() resource.Resource {
	return rs.ResourceList()
}
