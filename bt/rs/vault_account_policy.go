package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &vaultAccountPolicyResource{}
	_ resource.ResourceWithConfigure   = &vaultAccountPolicyResource{}
	_ resource.ResourceWithImportState = &vaultAccountPolicyResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultAccountPolicyResource{}
)

func newVaultAccountPolicyResource() resource.Resource {
	return &vaultAccountPolicyResource{}
}

type vaultAccountPolicyResource struct {
	apiResource[api.VaultAccountPolicy, models.VaultAccountPolicy]
}

func (r *vaultAccountPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Vault Account Policy.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"auto_rotate_credentials": schema.BoolAttribute{
				Required: true,
			},
			"allow_simultaneous_checkout": schema.BoolAttribute{
				Required: true,
			},
			"scheduled_password_rotation": schema.BoolAttribute{
				Required: true,
			},
			"maximum_password_age": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}
