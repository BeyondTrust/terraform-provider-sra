package ds

import (
	"context"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"
	. "terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &shellJumpDataSource{}
	_ datasource.DataSourceWithConfigure = &shellJumpDataSource{}
)

func NewShellJumpDataSource() datasource.DataSource {
	return &shellJumpDataSource{}
}

type shellJumpDataSource struct {
	apiClient *api.APIClient
}

type shellJumpDataSourceModel struct {
	Items []ShellJumpModel `tfsdk:"items"`
}

func (d *shellJumpDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shell_jump_item"
}

func (d *shellJumpDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
					},
				},
			},
		},
	}
}

func (d *shellJumpDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state shellJumpDataSourceModel

	// items, err := d.apiClient.ListShellJumpItems()
	items, err := api.ListItems[api.ShellJump](d.apiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list shell jump items",
			err.Error(),
		)
		return
	}

	for _, item := range items {
		shellJumpState := ShellJumpModel{
			ID:            types.StringValue(strconv.Itoa(int(*item.ID))),
			Name:          types.StringValue(item.Name),
			JumpointID:    types.Int64Value(int64(item.JumpointID)),
			Hostname:      types.StringValue(item.Hostname),
			Protocol:      types.StringValue(item.Protocol),
			Port:          types.Int64Value(int64(item.Port)),
			JumpGroupID:   types.Int64Value(int64(item.JumpGroupID)),
			JumpGroupType: types.StringValue(item.JumpGroupType),
			Terminal:      types.StringValue(item.Terminal),
			KeepAlive:     types.Int64Value(int64(item.KeepAlive)),
			Tag:           types.StringValue(item.Tag),
			Comments:      types.StringValue(item.Comments),
			Username:      types.StringValue(item.Username),
		}
		if item.JumpPolicyID != nil {
			shellJumpState.JumpPolicyID = types.Int64Value(int64(*item.JumpPolicyID))
		}
		if item.SessionPolicyID != nil {
			shellJumpState.SessionPolicyID = types.Int64Value(int64(*item.SessionPolicyID))
		}

		state.Items = append(state.Items, shellJumpState)
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *shellJumpDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.apiClient = req.ProviderData.(*api.APIClient)
}
