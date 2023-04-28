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
							Required: true,
						},
						"description": schema.StringAttribute{
							Required: true,
						},
						"perm_add": schema.BoolAttribute{
							Optional: true,
						},
						"perm_assign_jump_group": schema.BoolAttribute{
							Optional: true,
						},
						"perm_remove": schema.BoolAttribute{
							Optional: true,
						},
						"perm_start": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_tag": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_comments": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_jump_policy": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_session_policy": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_identity": schema.BoolAttribute{
							Optional: true,
						},
						"perm_edit_behavior": schema.BoolAttribute{
							Optional: true,
						},
						"perm_view_jump_item_report": schema.BoolAttribute{
							Optional: true,
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
