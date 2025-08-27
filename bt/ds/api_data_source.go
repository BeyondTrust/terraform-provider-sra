package ds

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Factory function to return the list of all datasource–generating factories to the main provider
// Add new datasource factory functions here.
func DatasourceList() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Alphabetical by file name
		newGroupPolicyDataSource,
		newJumpClientInstallerDataSource,
		newJumpGroupDataSource,
		newJumpItemRoleDataSource,
		newJumpPolicyDataSource,
		newJumpointDataSource,
		newProtocolTunnelJumpDataSource,
		newRemoteRDPDataSource,
		newRemoteVNCDataSource,
		newSessionPolicyDataSource,
		newShellJumpDataSource,
		newWebJumpDataSource,
		newPostgreSQLTunnelJumpDataSource,
		newMySQLTunnelJumpDataSource,
		newNetworkTunnelJumpDataSource,

		newVaultAccountDataSource,
		newVaultSSHAccountDataSource,
		newVaultAccountGroupDataSource,
		newVaultAccountPolicyDataSource,
		newVaultSecretDataSource,
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
	resp.TypeName = fmt.Sprintf("%s_%s", req.ProviderTypeName, d.printableName())
	tflog.Debug(ctx, fmt.Sprintf("🥃 Registered datasource name [%s]", resp.TypeName))
}

func (d *apiDataSource[TDataSource, TApi, TTf]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var item TApi
	if !api.IsProductAllowed(ctx, item) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s data source", api.ProductName(), d.printableName()),
			fmt.Sprintf("The %s data source can't be used when BT_API_HOST is configured for a %s site.", d.printableName(), api.ProductName()),
		)
		return
	}
	var state TDataSource
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	filter := api.MakeFilterMap(state)

	tflog.Debug(ctx, "🙀 list with filter", map[string]interface{}{
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
	tflog.Debug(ctx, "🙀 ListItems got data", map[string]interface{}{
		"data": string(rb),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to list %s items", d.printableName()),
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

		tflog.Debug(ctx, "🐉 TF Object is now copied", map[string]interface{}{
			"object": itemState,
		})

		for i := 0; i < itemStateObj.NumField(); i++ {
			field := itemStateObj.Field(i)
			fieldName := itemStateObj.Type().Field(i).Name
			m := field.MethodByName("IsUnknown")
			mCallable := m.Interface().(func() bool)
			if mCallable() {
				tflog.Debug(ctx, fmt.Sprintf("👻 IsUnknown? [%s][%v]", fieldName, mCallable()))
			}
		}

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

func (d *apiDataSource[TDataSource, TApi, TTf]) printableName() string {
	var tmp TApi
	name := reflect.TypeOf(tmp).String()
	parts := strings.Split(name, ".")

	return fmt.Sprintf("%s_list", api.ToSnakeCase(parts[len(parts)-1]))
}
