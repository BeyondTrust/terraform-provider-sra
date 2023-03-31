package rs

import (
	"context"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithConfigure   = &vaultUsernamePasswordAccountResource{}
	_ resource.ResourceWithImportState = &vaultUsernamePasswordAccountResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultUsernamePasswordAccountResource{}
)

func newVaultUsernamePasswordAccountResource() resource.Resource {
	return &vaultUsernamePasswordAccountResource{}
}

type vaultUsernamePasswordAccountResource struct {
	apiResource[api.VaultUsernamePasswordAccount, models.VaultUsernamePasswordAccount]
}

func (r *vaultUsernamePasswordAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Vault Username/Password Account.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString("username_password"),
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"personal": schema.BoolAttribute{
				Computed: true,
			},
			"owner_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"account_group_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
			},
			"account_policy": schema.StringAttribute{
				Optional: true,
			},
			"username": schema.StringAttribute{
				Required: true,
			},
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"last_checkout_timestamp": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}
