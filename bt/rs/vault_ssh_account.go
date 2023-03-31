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
	_ resource.Resource                = &vaultSSHAccountResource{}
	_ resource.ResourceWithConfigure   = &vaultSSHAccountResource{}
	_ resource.ResourceWithImportState = &vaultSSHAccountResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultSSHAccountResource{}
)

func newVaultSSHAccountResource() resource.Resource {
	return &vaultSSHAccountResource{}
}

type vaultSSHAccountResource struct {
	apiResource[api.VaultSSHAccount, models.VaultSSHAccount]
}

func (r *vaultSSHAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Vault SSH Account.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Computed: true,
				Default:  stringdefault.StaticString("ssh"),
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
			"public_key": schema.StringAttribute{
				Computed: true,
			},
			"private_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"private_key_passphrase": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"private_key_public_cert": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"last_checkout_timestamp": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}
