package ds

import (
	"context"
	"reflect"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = &shellJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &shellJumpDataSource{}
)

func newShellJumpDataSource() datasource.DataSource {
	return &shellJumpDataSource{}
}

type shellJumpDataSource struct {
	apiDataSource[api.ShellJump, models.ShellJumpModel]
}

type shellJumpDataSourceModel struct {
	Items []models.ShellJumpModel `tfsdk:"items"`
	models.ShellJumpModel
}

func (d *shellJumpDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state shellJumpDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	typ := reflect.TypeOf(state)
	ste := reflect.ValueOf(state)
	var filter = make(map[string]string)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i).Tag.Get("filter")
		if f != "" {
			filter[f] = ste.Field(i).String()
		}
	}

	items := d.doFilteredRead(ctx, req, resp, filter)

	if items == nil {
		return
	}

	state.Items = items

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *shellJumpDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	sjAttributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
			Required: false,
			Optional: false,
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"jumpoint_id": schema.Int64Attribute{
			Required: true,
		},
		"hostname": schema.StringAttribute{
			Required: true,
		},
		"protocol": schema.StringAttribute{
			Required: true,
		},
		"port": schema.Int64Attribute{
			Optional: true,
		},
		"jump_group_id": schema.Int64Attribute{
			Required: true,
		},
		"jump_group_type": schema.StringAttribute{
			Required: true,
		},
		"terminal": schema.StringAttribute{
			Optional: true,
		},
		"keep_alive": schema.Int64Attribute{
			Optional: true,
		},
		"tag": schema.StringAttribute{
			Optional: true,
		},
		"comments": schema.StringAttribute{
			Optional: true,
		},
		"jump_policy_id": schema.Int64Attribute{
			Optional: true,
		},
		"username": schema.StringAttribute{
			Optional: true,
		},
		"session_policy_id": schema.Int64Attribute{
			Optional: true,
		},
	}

	resp.Schema = schema.Schema{
		Attributes: sjAttributes,
	}

	resp.Schema.Attributes["item"] = schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: sjAttributes,
		},
	}
}
