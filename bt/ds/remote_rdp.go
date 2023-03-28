package ds

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &remoteRDPDataSource{}
	_ datasource.DataSourceWithConfigure = &remoteRDPDataSource{}
)

func newRemoteRDPDataSource() datasource.DataSource {
	return &remoteRDPDataSource{}
}

type remoteRDPDataSource struct {
	apiDataSource[remoteRDPDataSourceModel, api.RemoteRDP, models.RemoteRDPModel]
}

type remoteRDPDataSourceModel struct {
	Items         []models.RemoteRDPModel `tfsdk:"items"`
	Name          types.String            `tfsdk:"name" filter:"name"`
	JumpointID    types.Int64             `tfsdk:"jumpoint_id" filter:"jumpoint_id"`
	Hostname      types.String            `tfsdk:"hostname" filter:"hostname"`
	JumpGroupID   types.Int64             `tfsdk:"jump_group_id" filter:"jump_group_id"`
	JumpGroupType types.String            `tfsdk:"jump_group_type" filter:"jump_group_type"`
	EndpointID    types.Int64             `tfsdk:"endpoint_id" filter:"endpoint_id"`
	Tag           types.String            `tfsdk:"tag" filter:"tag"`
}

func (d *remoteRDPDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Shell Jump Items.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
						"jump_group_id": schema.Int64Attribute{
							Required: true,
						},
						"jump_group_type": schema.StringAttribute{
							Required: true,
						},
						"quality": schema.StringAttribute{
							Optional: true,
						},
						"console": schema.BoolAttribute{
							Optional: true,
						},
						"ignore_untrusted": schema.BoolAttribute{
							Optional: true,
						},
						"tag": schema.StringAttribute{
							Optional: true,
						},
						"comments": schema.StringAttribute{
							Optional: true,
						},
						"rdp_username": schema.StringAttribute{
							Optional: true,
						},
						"domain": schema.StringAttribute{
							Optional: true,
						},
						"secure_app_type": schema.StringAttribute{
							Optional: true,
						},
						"remote_app_name": schema.StringAttribute{
							Optional: true,
						},
						"remote_app_params": schema.StringAttribute{
							Optional: true,
						},
						"remote_exe_path": schema.StringAttribute{
							Optional: true,
						},
						"remote_exe_params": schema.StringAttribute{
							Optional: true,
						},
						"target_system": schema.StringAttribute{
							Optional: true,
						},
						"credential_type": schema.StringAttribute{
							Optional: true,
						},
						"session_forensics": schema.BoolAttribute{
							Optional: true,
						},
						"jump_policy_id": schema.Int64Attribute{
							Optional: true,
						},
						"session_policy_id": schema.Int64Attribute{
							Optional: true,
						},
						"endpoint_id": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter the list for items matching \"name\"",
				Optional:    true,
			},
			"jumpoint_id": schema.Int64Attribute{
				Description: "Filter the list for items with a matching \"jumpoint_id\"",
				Optional:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"hostname\"",
				Optional:    true,
			},
			"jump_group_id": schema.Int64Attribute{
				Description: "Filter the list for items with a matching \"jump_group_id\"",
				Optional:    true,
			},
			"jump_group_type": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"jump_group_type\"",
				Optional:    true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Filter the list for items with a matching \"endpoint_id\"",
				Optional:    true,
			},
			"tag": schema.StringAttribute{
				Description: "Filter the list for items with a matching \"tag\"",
				Optional:    true,
			},
		},
	}
}
