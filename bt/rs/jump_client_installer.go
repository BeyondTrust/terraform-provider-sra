package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	_ resource.Resource                = &jumpClientInstallerResource{}
	_ resource.ResourceWithConfigure   = &jumpClientInstallerResource{}
	_ resource.ResourceWithImportState = &jumpClientInstallerResource{}
	_ resource.ResourceWithModifyPlan  = &jumpClientInstallerResource{}
)

func newJumpClientInstallerResource() resource.Resource {
	return &jumpClientInstallerResource{}
}

type jumpClientInstallerResource struct {
	apiResource[api.JumpClientInstaller, models.JumpClientInstaller]
}

func (r *jumpClientInstallerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Jump Client Installer.

		*NOTE*: It is not recommended to use any installers managed by Terraform outside of Terraform
		and any automated provisioning based on the Terraform configuration. Installers will be
		deleted and recreated in response to any changes to the Terraform configuration, which
		will invalidate any existing copies of the installer.

		For descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance.
`,
		Attributes: jciSchema,
	}
}

func (r jumpClientInstallerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Mark all attributes in the schema that can be supplied as requiring a full replacement
	// of the item since we don't allow modification of existing installers
	for attr, val := range jciSchema {
		if !val.IsRequired() && !val.IsOptional() {
			continue
		}
		resp.RequiresReplace = resp.RequiresReplace.Append(
			path.Root(attr),
		)
	}

	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.InDebugfo(ctx, "No plan to modify")
		return
	}
	var plan models.JumpClientInstaller
	diags := req.Plan.Get(ctx, &plan)
	tflog.Debug(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Error reading plan")
		return
	}

	if api.IsPRA() {
		if plan.SessionPolicyID.IsUnknown() {
			plan.SessionPolicyID = types.Int64Null()
		}
		if plan.AllowOverrideSessionPolicy.IsUnknown() {
			plan.AllowOverrideSessionPolicy = types.BoolValue(false)
		}
	} else if api.IsRS() {
		if plan.AttendedSessionPolicyID.IsUnknown() {
			plan.AttendedSessionPolicyID = types.Int64Null()
		}
		if plan.AllowOverrideAttendedSessionPolicy.IsUnknown() {
			plan.AllowOverrideAttendedSessionPolicy = types.BoolValue(false)
		}
		if plan.UnattendedSessionPolicyID.IsUnknown() {
			plan.UnattendedSessionPolicyID = types.Int64Null()
		}
		if plan.AllowOverrideUnattendedSessionPolicy.IsUnknown() {
			plan.AllowOverrideUnattendedSessionPolicy = types.BoolValue(false)
		}
		if plan.IsQuiet.IsUnknown() {
			plan.IsQuiet = types.BoolValue(false)
		}
	}

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished modification")
}

var jciSchema = map[string]schema.Attribute{
	"id": schema.StringAttribute{
		Computed: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"valid_duration": schema.Int64Attribute{
		Optional: true,
	},
	"name": schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString(""),
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
	"jump_policy_id": schema.Int64Attribute{
		Optional: true,
	},
	"connection_type": schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString("active"),
		Validators: []validator.String{
			stringvalidator.OneOf([]string{"active", "passive"}...),
		},
	},
	"expiration_timestamp": schema.StringAttribute{
		Computed: true,
	},
	"max_offline_minutes": schema.Int64Attribute{
		Optional: true,
		Computed: true,
		Default:  int64default.StaticInt64(0),
	},
	"elevate_install": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(true),
	},
	"elevate_prompt": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(true),
	},
	"allow_override_jump_policy": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
	},
	"allow_override_jump_group": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
	},
	"allow_override_name": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
	},
	"allow_override_tag": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
	},
	"allow_override_comments": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
	},
	"allow_override_max_offline_minutes": schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(false),
		Description: "If true, the max offline minutes can be specified during installation, which will override the max offline minutes specified in this API call.",
	},
	"installer_id": schema.StringAttribute{
		Computed: true,
	},
	"key_info": schema.StringAttribute{
		Computed: true,
	},

	"session_policy_id": schema.Int64Attribute{
		Optional:    true,
		Description: "This field only applies to PRA",
	},
	"allow_override_session_policy": schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "This field only applies to PRA",
	},

	// RS Attributes
	"attended_session_policy_id": schema.Int64Attribute{
		Optional:    true,
		Description: "This field only applies to RS",
	},
	"unattended_session_policy_id": schema.Int64Attribute{
		Optional:    true,
		Description: "This field only applies to RS",
	},
	"allow_override_attended_session_policy": schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "This field only applies to RS",
	},
	"allow_override_unattended_session_policy": schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "This field only applies to RS",
	},
	"is_quiet": schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "This field only applies to RS",
	},
}
