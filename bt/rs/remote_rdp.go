package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &remoteRDPResource{}
	_ resource.ResourceWithConfigure   = &remoteRDPResource{}
	_ resource.ResourceWithImportState = &remoteRDPResource{}
	// _ resource.ResourceWithModifyPlan  = &remoteRDPResource{}
)

func newRemoteRDPResource() resource.Resource {
	return &remoteRDPResource{}
}

type remoteRDPResource struct {
	apiResource[api.RemoteRDP, models.RemoteRDP]
}

// We must define the schema for each resource individually. Anything that can be supplied by the API response
// needs to be marked as "Computed", even if we translate from "null" on a POST to an empty string
func (r *remoteRDPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Remote RDP Jump Item.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("shared"),
				Validators: jumpGroupTypeValidator(),
			},
			"quality": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("video"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"low", "performance", "performance_quality", "quality", "video", "lossless"}...),
				},
			},
			"console": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"ignore_untrusted": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"rdp_username": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"domain": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"secure_app_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"", "none", "remote_app", "remote_desktop_agent", "remote_desktop_agent_credentials"}...),
				},
			},
			"remote_app_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"remote_app_params": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"remote_exe_path": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"remote_exe_params": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"target_system": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"credential_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"session_forensics": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "This field only applies to PRA",
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
	}
}
