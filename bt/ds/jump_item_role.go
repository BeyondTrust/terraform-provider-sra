package ds

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &jumpItemRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &jumpItemRoleDataSource{}
	_                                    = &jumpItemRoleDataSourceModel{}
)

func newJumpItemRoleDataSource() datasource.DataSource {
	return &jumpItemRoleDataSource{}
}

type jumpItemRoleDataSource struct {
	apiDataSource[jumpItemRoleDataSourceModel, api.JumpItemRole, models.JumpItemRole]
}

type jumpItemRoleDataSourceModel struct {
	Items []models.JumpItemRole `tfsdk:"items"`
	Name  types.String          `tfsdk:"name" filter:"name"`
}

func (d *jumpItemRoleDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Jump Item Roles.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"perm_add": schema.BoolAttribute{
							Computed: true,
						},
						"perm_assign_jump_group": schema.BoolAttribute{
							Computed: true,
						},
						"perm_remove": schema.BoolAttribute{
							Computed: true,
						},
						"perm_start": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_tag": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_comments": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_jump_policy": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_session_policy": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_identity": schema.BoolAttribute{
							Computed: true,
						},
						"perm_edit_behavior": schema.BoolAttribute{
							Computed: true,
						},
						"perm_view_jump_item_report": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the list for roles matching \"name\"",
				Optional:    true,
			},
		},
	}
}
