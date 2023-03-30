package rs

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

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
	_ resource.Resource                = &jumpointResource{}
	_ resource.ResourceWithConfigure   = &jumpointResource{}
	_ resource.ResourceWithImportState = &jumpointResource{}
	// _ resource.ResourceWithModifyPlan  = &jumpointResource{}
)

func newJumpointResource() resource.Resource {
	return &jumpointResource{}
}

type jumpointResource struct {
	apiResource[api.Jumpoint, models.JumpointModel]
}

func (r *jumpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Jump Group.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
			"code_name": schema.StringAttribute{
				Required: true,
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"platform": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"windows-x86", "linux-x86"}...),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"connected": schema.BoolAttribute{
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"clustered": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"shell_jump_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"external_jump_item_network_id": schema.StringAttribute{
				Optional: true,
			},
			"protocol_tunnel_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"rdp_service_account_id": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}
