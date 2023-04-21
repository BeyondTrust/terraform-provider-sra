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
	_ datasource.DataSource              = &jumpPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &jumpPolicyDataSource{}
	_                                    = &jumpPolicyDataSourceModel{}
)

func newJumpPolicyDataSource() datasource.DataSource {
	return &jumpPolicyDataSource{}
}

type jumpPolicyDataSource struct {
	apiDataSource[jumpPolicyDataSourceModel, api.JumpPolicy, models.JumpPolicy]
}

type jumpPolicyDataSourceModel struct {
	Items    []models.JumpPolicy `tfsdk:"items"`
	CodeName types.String        `tfsdk:"code_name" filter:"code_name"`
}

func (d *jumpPolicyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a list of Jump Policies.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
						"display_name": schema.StringAttribute{
							Required: true,
						},
						"code_name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"schedule_enabled": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"schedule_strict": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"session_start_notification": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"session_end_notification": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"notification_email_addresses": schema.SetAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
						},
						"notification_display_name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"notification_email_language": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"ticket_id_required": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"approval_required": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"approval_max_duration": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
						"approval_scope": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"approval_email_addresses": schema.SetAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
						},
						"approval_user_ids": schema.SetAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
						},
						"approval_display_name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"approval_email_language": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"recordings_disabled": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"code_name": schema.StringAttribute{
				Description: "Filter the list for Jumpoints with a matching \"code_name\"",
				Optional:    true,
			},
		},
	}
}
