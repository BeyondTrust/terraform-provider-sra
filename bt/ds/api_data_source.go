package ds

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"terraform-provider-beyondtrust-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Factory function to return the list of all datasource–generating factories to the main provider
// Add new datasource factory functions here.
func DatasourceList() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Alphabetical by file name
		newGroupPolicyDataSource,
		newJumpGroupDataSource,
		newJumpItemRoleDataSource,
		newJumpointDataSource,
		newProtocolTunnelJumpDataSource,
		newRemoteRDPDataSource,
		newRemoteVNCDataSource,
		newSessionPolicyDataSource,
		newShellJumpDataSource,
		newWebJumpDataSource,

		newVaultAccountDataSource,
		newVaultAccountGroupDataSource,
	}
}

type apiDataSource[TDataSource any, TApi api.APIResource, TTf any] struct {
	apiClient *api.APIClient
}

// Can't compose structs for the terraform types,
// https://github.com/hashicorp/terraform-plugin-framework/issues/309
// If that ever gets fixed then we can have a base TF model for data sources
// type apiDataSourceModel[T any] struct {
// 	Items []T `tfsdk:"items"`
// }

func (d *apiDataSource[TDataSource, TApi, TTf]) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	var tmp TApi
	name := reflect.TypeOf(tmp).String()
	parts := strings.Split(name, ".")

	resp.TypeName = fmt.Sprintf("%s_%s_list", req.ProviderTypeName, api.ToSnakeCase(parts[len(parts)-1]))
	tflog.Info(ctx, fmt.Sprintf("🥃 Registered datasource name [%s]", resp.TypeName))
}

func (d *apiDataSource[TDataSource, TApi, TTf]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TDataSource
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	filter := api.MakeFilterMap(ctx, state)

	tflog.Info(ctx, "🙀 list with filter", map[string]interface{}{
		"data": filter,
	})

	items := d.doFilteredRead(ctx, req, resp, filter)

	if items == nil {
		return
	}

	itemField := reflect.ValueOf(&state).Elem().FieldByName("Items")
	listElem := reflect.ValueOf(&items).Elem()
	itemField.Set(listElem)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// func (d *apiDataSource[TDataSource, TApi, TTf]) doRead(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) []TTf {
// 	return d.doFilteredRead(ctx, req, resp, nil)
// }

func (d *apiDataSource[TDataSource, TApi, TTf]) doFilteredRead(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse, requestFilter map[string]string) []TTf {
	items, err := api.ListItems[TApi](d.apiClient, requestFilter)
	rb, _ := json.Marshal(items)
	tflog.Info(ctx, "🙀 ListItems got data", map[string]interface{}{
		"data": string(rb),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list shell jump items",
			err.Error(),
		)
		return nil
	}

	tfItems := []TTf{}
	for _, item := range items {
		var itemState TTf

		itemObj := reflect.ValueOf(&item).Elem()
		apiType := reflect.TypeOf(&item).Elem()
		itemStateObj := reflect.ValueOf(&itemState).Elem()

		api.CopyAPItoTF(ctx, itemObj, itemStateObj, apiType)

		tfItems = append(tfItems, itemState)
	}

	return tfItems
}

func (d *apiDataSource[TDataSource, TApi, TTf]) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil || d == nil {
		return
	}

	d.apiClient = req.ProviderData.(*api.APIClient)
}
