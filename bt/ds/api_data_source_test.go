package ds

import (
	"context"
	"net/http"
	"testing"

	"terraform-provider-sra/api"
	"terraform-provider-sra/api/apimock"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockClient(srv *apimock.Server) *api.APIClient {
	return &api.APIClient{RootURL: srv.URL, BaseURL: srv.URL + apimock.BasePath, HTTPClient: srv.Client()}
}

// readDataSource runs d.Read against client with the given config attributes
// (every other attribute null) and returns the resulting state.
func readDataSource(t *testing.T, d datasource.DataSource, client *api.APIClient, config map[string]tftypes.Value) (tfsdk.State, diag.Diagnostics) {
	t.Helper()
	ctx := context.Background()

	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, &datasource.ConfigureResponse{})

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())
	typ := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	vals := map[string]tftypes.Value{}
	for name, attrType := range typ.AttributeTypes {
		if v, ok := config[name]; ok {
			vals[name] = v
		} else {
			vals[name] = tftypes.NewValue(attrType, nil)
		}
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, nil)}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, vals)}}, &resp)
	return resp.State, resp.Diagnostics
}

func jumpGroupRows(n int) []apimock.Row {
	rows := make([]apimock.Row, n)
	for i := range rows {
		rows[i] = apimock.Row{"id": i + 1, "name": "group", "code_name": "group", "comments": ""}
	}
	return rows
}

// List data sources used to stop at the appliance's first page of 100.
func TestListDataSourceReturnsEveryPage(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Rows("jump-group", jumpGroupRows(250)...)

	state, diags := readDataSource(t, newJumpGroupDataSource(), mockClient(srv), nil)
	require.False(t, diags.HasError(), "%v", diags)

	var model jumpGroupDataSourceModel
	require.False(t, state.Get(context.Background(), &model).HasError())
	require.Len(t, model.Items, 250)
	assert.Equal(t, "1", model.Items[0].ID.ValueString(), "items[0] is still the first row")
	assert.Equal(t, 3, srv.Calls(http.MethodGet, "jump-group"))
}

// A list that keeps changing under the read is an error, never the rows that
// happened to be read: a short list in for_each plans destroys.
func TestListDataSourceUnstableListIsAnError(t *testing.T) {
	t.Parallel()
	srv := apimock.New()
	defer srv.Close()
	srv.Set("jump-group", &apimock.Collection{
		Rows: jumpGroupRows(201),
		OnPage: func(c *apimock.Collection, page int) {
			if page == 2 {
				c.Rows = c.Rows[1:]
			}
		},
	})

	state, diags := readDataSource(t, newJumpGroupDataSource(), mockClient(srv), nil)
	require.True(t, diags.HasError())
	assert.Equal(t, "Unable to list jump_group_list items", diags.Errors()[0].Summary())
	assert.Contains(t, diags.Errors()[0].Detail(),
		"The list of jump_group objects changed while it was being read, in 3 consecutive attempts (last: 100 unique of 199 reported)")
	assert.True(t, state.Raw.IsNull(), "no items reach state")
}
