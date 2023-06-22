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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"", "remote_app", "remote_desktop_agent", "remote_desktop_agent_credentials"}...),
				},
				Description: "This field only applies to PRA",
			},
			"remote_app_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"remote_app_params": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"remote_exe_path": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"remote_exe_params": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"target_system": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"credential_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "This field only applies to PRA",
			},
			"session_forensics": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
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

func (r *remoteRDPResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}
	var plan models.RemoteRDP
	diags := req.Plan.Get(ctx, &plan)
	tflog.Debug(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Error reading plan")
		return
	}
	/*
		Here we are setting some things that get defaults if they are not supplied.
	*/
	if api.IsPRA() {
		if plan.SecureAppType.IsNull() || plan.SecureAppType.IsUnknown() || plan.SecureAppType.ValueString() == "" {
			plan.SecureAppType = types.StringValue("")
			plan.RemoteAppName = types.StringValue("")
			plan.RemoteAppParams = types.StringValue("")
			plan.RemoteExePath = types.StringValue("")
			plan.TargetSystem = types.StringValue("")
			plan.CredentialType = types.StringValue("")
		} else {
			if plan.RemoteAppName.IsUnknown() {
				plan.RemoteAppName = types.StringValue("")
			}
			if plan.RemoteAppParams.IsUnknown() {
				plan.RemoteAppParams = types.StringValue("")
			}
			if plan.RemoteExePath.IsUnknown() {
				plan.RemoteExePath = types.StringValue("")
			}
			if plan.RemoteExeParams.IsUnknown() {
				plan.RemoteExeParams = types.StringValue("")
			}
			if plan.TargetSystem.IsUnknown() {
				plan.TargetSystem = types.StringValue("")
			}
			if plan.CredentialType.IsUnknown() {
				plan.CredentialType = types.StringValue("")
			}
		}

		if plan.SessionForensics.IsUnknown() || plan.SessionForensics.IsNull() {
			plan.SessionForensics = types.BoolValue(false)
		}
	} else {
		plan.SecureAppType = types.StringNull()
		plan.RemoteAppName = types.StringNull()
		plan.RemoteAppParams = types.StringNull()
		plan.RemoteExePath = types.StringNull()
		plan.TargetSystem = types.StringNull()
		plan.CredentialType = types.StringNull()
		plan.SessionForensics = types.BoolNull()
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification")
}
