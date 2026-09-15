package rs

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &groupPolicyResource{}
	_ resource.ResourceWithConfigure   = &groupPolicyResource{}
	_ resource.ResourceWithImportState = &groupPolicyResource{}
	_ resource.ResourceWithModifyPlan  = &groupPolicyResource{}
)

const groupPolicyMaximumID int64 = 2147483647

func newGroupPolicyResource() resource.Resource {
	return &groupPolicyResource{}
}

type groupPolicyResource struct {
	apiResource[api.GroupPolicy, models.GroupPolicy]
}

func (r *groupPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Group Policy for either Privileged Remote Access (PRA) or Remote Support (RS). Product-specific attributes must only be configured for the matching appliance type.\n\nOn PRA, enabling a Jump permission requires effective endpoint access: `perm_access_allowed` must be `true`, and `access_perm_status` must not be `not_defined`. The appliance otherwise normalizes enabled Jump permissions back to `false`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The numeric identifier assigned to the group policy by the appliance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the group policy.",
				Validators: []validator.String{
					stringvalidator.UTF8LengthBetween(1, 255),
				},
			},

			"perm_share_other_team":                   defaultGroupPolicyBool("Whether users may share sessions with teams to which they do not belong."),
			"perm_session_idle_timeout":               groupPolicyIdleTimeoutAttribute("The session inactivity timeout in seconds. -1 uses the site-wide setting and 0 disables the timeout."),
			"perm_extended_availability_mode_allowed": defaultGroupPolicyBool("Whether users may enable extended availability mode."),
			"perm_edit_external_key":                  defaultGroupPolicyBool("Whether users may edit the external key."),
			"perm_collaborate":                        defaultGroupPolicyBool("Whether users may show their screen to other users or representatives."),
			"perm_collaborate_control":                defaultGroupPolicyBool("Whether users may give control while showing their screen."),
			"perm_jump_client":                        defaultGroupPolicyBool("Whether users may use Jump Clients."),
			"perm_local_jump":                         defaultGroupPolicyBool("Whether users may use Local Jump."),
			"perm_remote_jump":                        defaultGroupPolicyBool("Whether users may use Remote Jump."),
			"perm_remote_vnc":                         defaultGroupPolicyBool("Whether users may use Remote VNC."),
			"perm_remote_rdp":                         defaultGroupPolicyBool("Whether users may use Remote RDP."),
			"perm_shell_jump":                         defaultGroupPolicyBool("Whether users may use Shell Jump."),
			"default_jump_item_role_id":               groupPolicyRoleAttribute("The default Jump Item Role ID."),
			"private_jump_item_role_id":               groupPolicyRoleAttribute("The personal Jump Item Role ID."),
			"inferior_jump_item_role_id":              groupPolicyRoleAttribute("The teams Jump Item Role ID."),
			"unassigned_jump_item_role_id":            groupPolicyRoleAttribute("The system Jump Item Role ID."),

			"perm_access_allowed": optionalComputedGroupPolicyBool("Whether users may access endpoints. This field only applies to PRA."),
			"access_perm_status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this policy defines user permissions and whether lower-priority policies may override them. This field only applies to PRA.",
				Validators: []validator.String{
					stringvalidator.OneOf("not_defined", "defined", "final"),
				},
			},
			"perm_invite_external_user":              optionalComputedGroupPolicyBool("Whether users may invite external users. This field only applies to PRA."),
			"perm_web_jump":                          optionalComputedGroupPolicyBool("Whether users may use Web Jump. This field only applies to PRA."),
			"perm_protocol_tunnel":                   optionalComputedGroupPolicyBool("Whether users may use Protocol Tunnel Jump. This field only applies to PRA."),
			"perm_sd_static_port_for_external_tools": optionalComputedGroupPolicyBool("Whether users may use a static port and username for external-tool sessions. This field only applies to PRA."),

			"perm_support_allowed": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The level of remote support users may provide. This field only applies to RS.",
				Validators: []validator.String{
					stringvalidator.OneOf("not_allowed", "full_support"),
				},
			},
			"rep_perm_status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this policy defines representative permissions and whether lower-priority policies may override them. This field only applies to RS.",
				Validators: []validator.String{
					stringvalidator.OneOf("not_defined", "defined", "final"),
				},
			},
			"perm_generate_session_key":    optionalComputedGroupPolicyBool("Whether representatives may generate session keys. This field only applies to RS."),
			"perm_send_ios_profiles":       optionalComputedGroupPolicyBool("Whether representatives may generate access keys for sending iOS profiles. This field only applies to RS."),
			"perm_accept_team_sessions":    optionalComputedGroupPolicyBool("Whether representatives may manually accept sessions from a team queue. This field only applies to RS."),
			"perm_transfer_other_team":     optionalComputedGroupPolicyBool("Whether representatives may transfer sessions to other teams. This field only applies to RS."),
			"perm_invite_external_rep":     optionalComputedGroupPolicyBool("Whether representatives may invite external representatives. This field only applies to RS."),
			"perm_next_session_button":     optionalComputedGroupPolicyBool("Whether representatives may use Get Next Session. This field only applies to RS."),
			"perm_disable_auto_assignment": optionalComputedGroupPolicyBool("Whether representatives may opt out of automatic session assignment. This field only applies to RS."),
			"perm_routing_idle_timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The idle time in seconds after which sessions are no longer assigned. This field only applies to RS.",
				Validators: []validator.Int64{
					int64validator.OneOf(0, 180, 300, 600, 900, 1200, 1800, 2700, 3600),
				},
			},
			"auto_assignment_max_sessions": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The maximum number of sessions a representative may have before new sessions are not assigned. This field only applies to RS.",
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"perm_support_button_personal_deploy":     optionalComputedGroupPolicyBool("Whether representatives may deploy and manage personal Support Buttons. This field only applies to RS."),
			"perm_support_button_team_manage":         optionalComputedGroupPolicyBool("Whether representatives may manage team Support Buttons. This field only applies to RS."),
			"perm_support_button_change_public_sites": optionalComputedGroupPolicyBool("Whether representatives may change the Public Portal for Support Buttons. This field only applies to RS."),
			"perm_support_button_team_deploy":         optionalComputedGroupPolicyBool("Whether representatives may deploy team Support Buttons. This field only applies to RS."),
			"perm_local_vnc":                          optionalComputedGroupPolicyBool("Whether representatives may use Local VNC. This field only applies to RS."),
			"perm_local_rdp":                          optionalComputedGroupPolicyBool("Whether representatives may use Local RDP. This field only applies to RS."),
			"perm_vpro":                               optionalComputedGroupPolicyBool("Whether representatives may use Intel vPro. This field only applies to RS."),
			"perm_console_idle_timeout":               groupPolicyProductIdleTimeoutAttribute("The Representative Console inactivity timeout in seconds. This field only applies to RS."),
		},
	}
}

func (r *groupPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.ApiClient == nil {
		return
	}

	var config models.GroupPolicy
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, field := range configuredGroupPolicyFieldsForOtherProduct(config, r.ApiClient.Product) {
		requiredProduct := "Privileged Remote Access (PRA)"
		if strings.EqualFold(field.product, api.ProductRS) {
			requiredProduct = "Remote Support (RS)"
		}
		resp.Diagnostics.AddAttributeError(
			path.Root(field.name),
			"Group Policy Attribute Is Not Supported By This Appliance",
			fmt.Sprintf("The %q attribute only applies to %s group policies, but the configured appliance is %s.", field.name, requiredProduct, r.ApiClient.ProductName()),
		)
	}

	if !r.ApiClient.IsPRA() {
		return
	}

	var plan models.GroupPolicy
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabledPermissions, accessConflict, statusConflict := praGroupPolicyAccessConflicts(plan)
	if accessConflict {
		resp.Diagnostics.AddAttributeError(
			path.Root("perm_access_allowed"),
			"PRA Jump Permissions Require Endpoint Access",
			fmt.Sprintf("Set perm_access_allowed to true when enabling these Jump permissions: %s. PRA normalizes them to false when endpoint access is disabled.", strings.Join(enabledPermissions, ", ")),
		)
	}
	if statusConflict {
		resp.Diagnostics.AddAttributeError(
			path.Root("access_perm_status"),
			"PRA Jump Permissions Require Defined Access Permissions",
			fmt.Sprintf("access_perm_status cannot be not_defined when enabling these Jump permissions: %s. Use defined or final, or omit the attribute and allow the API to select its documented default.", strings.Join(enabledPermissions, ", ")),
		)
	}
}

func (r *groupPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 32)
	if err != nil || id < 1 {
		resp.Diagnostics.AddError(
			"Invalid Group Policy Import ID",
			fmt.Sprintf("Group Policy resources must be imported using a numeric ID from 1 through %d; received %q.", groupPolicyMaximumID, req.ID),
		)
		return
	}

	r.apiResource.ImportState(ctx, req, resp)
}

type configuredGroupPolicyField struct {
	name    string
	product string
}

func configuredGroupPolicyFieldsForOtherProduct(config models.GroupPolicy, product string) []configuredGroupPolicyField {
	configValue := reflect.ValueOf(config)
	configType := reflect.TypeOf(config)
	var fields []configuredGroupPolicyField

	for i := 0; i < configValue.NumField(); i++ {
		structField := configType.Field(i)
		fieldProduct := structField.Tag.Get("sraproduct")
		if fieldProduct == "" || strings.EqualFold(fieldProduct, product) {
			continue
		}

		value, ok := configValue.Field(i).Interface().(attr.Value)
		if !ok || value.IsNull() || value.IsUnknown() {
			continue
		}

		fields = append(fields, configuredGroupPolicyField{
			name:    structField.Tag.Get("tfsdk"),
			product: fieldProduct,
		})
	}

	return fields
}

func praGroupPolicyAccessConflicts(plan models.GroupPolicy) (enabledPermissions []string, accessConflict bool, statusConflict bool) {
	permissions := []struct {
		name  string
		value types.Bool
	}{
		{name: "perm_jump_client", value: plan.PermJumpClient},
		{name: "perm_local_jump", value: plan.PermLocalJump},
		{name: "perm_remote_jump", value: plan.PermRemoteJump},
		{name: "perm_remote_vnc", value: plan.PermRemoteVnc},
		{name: "perm_remote_rdp", value: plan.PermRemoteRdp},
		{name: "perm_shell_jump", value: plan.PermShellJump},
		{name: "perm_web_jump", value: plan.PermWebJump},
		{name: "perm_protocol_tunnel", value: plan.PermProtocolTunnel},
		{name: "perm_sd_static_port_for_external_tools", value: plan.PermSdStaticPortForExternalTools},
	}

	for _, permission := range permissions {
		if !permission.value.IsNull() && !permission.value.IsUnknown() && permission.value.ValueBool() {
			enabledPermissions = append(enabledPermissions, permission.name)
		}
	}

	if len(enabledPermissions) == 0 {
		return enabledPermissions, false, false
	}

	accessConflict = plan.PermAccessAllowed.IsNull() || plan.PermAccessAllowed.IsUnknown() || !plan.PermAccessAllowed.ValueBool()
	statusConflict = !plan.AccessPermStatus.IsNull() && !plan.AccessPermStatus.IsUnknown() && plan.AccessPermStatus.ValueString() == "not_defined"
	return enabledPermissions, accessConflict, statusConflict
}

func defaultGroupPolicyBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(false),
		Description: description,
	}
}

func optionalComputedGroupPolicyBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
	}
}

func groupPolicyIdleTimeoutAttribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(-1),
		Description: description,
		Validators: []validator.Int64{
			groupPolicyIdleTimeoutValidator(),
		},
	}
}

func groupPolicyProductIdleTimeoutAttribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		Validators: []validator.Int64{
			groupPolicyIdleTimeoutValidator(),
		},
	}
}

func groupPolicyIdleTimeoutValidator() validator.Int64 {
	return int64validator.OneOf(-1, 0, 300, 600, 900, 1800, 3600, 7200, 14400, 28800, 43200, 86400)
}

func groupPolicyRoleAttribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(1),
		Description: description,
		Validators: []validator.Int64{
			int64validator.Between(1, groupPolicyMaximumID),
		},
	}
}
