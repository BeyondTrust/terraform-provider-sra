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
		newShellJumpDataSource,
	}
}

type apiDataSource[TApi api.APIResource, TTf any] struct {
	apiClient *api.APIClient
}

// type apiDataSourceModel[T any] struct {
// 	Items []T `tfsdk:"items"`
// }

func (d *apiDataSource[TApi, TTf]) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	var tmp TApi
	name := reflect.TypeOf(tmp).String()
	parts := strings.Split(name, ".")

	resp.TypeName = fmt.Sprintf("%s_%s_list", req.ProviderTypeName, api.ToSnakeCase(parts[len(parts)-1]))
	tflog.Info(ctx, fmt.Sprintf("🥃 Registered datasource name [%s]", resp.TypeName))
}

func (d *apiDataSource[TApi, TTf]) doRead(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) []TTf {
	return d.doFilteredRead(ctx, req, resp, nil)
}

func (d *apiDataSource[TApi, TTf]) doFilteredRead(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse, requestFilter map[string]string) []TTf {
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
		itemStateObj := reflect.ValueOf(&itemState).Elem()

		api.CopyAPItoTF(ctx, itemObj, itemStateObj)

		tfItems = append(tfItems, itemState)
	}

	return tfItems
}

func (d *apiDataSource[TApi, TTf]) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil || d == nil {
		return
	}

	d.apiClient = req.ProviderData.(*api.APIClient)
}
