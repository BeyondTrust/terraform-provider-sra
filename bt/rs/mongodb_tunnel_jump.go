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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &mongodbTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &mongodbTunnelJumpResource{}
	_ resource.ResourceWithImportState = &mongodbTunnelJumpResource{}
)

func newMongodbTunnelJumpResource() resource.Resource { return &mongodbTunnelJumpResource{} }

type mongodbTunnelJumpResource struct {
	apiResource[api.MongodbTunnelJump, models.MongodbTunnelJump]
}

func (r *mongodbTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a MongoDB Tunnel Jump Item.\n\nNOTE: PRA only. Requires PRA 26.1.1 or later.\n\n" + tunnelJumpNotAssociableNote,
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":              schema.StringAttribute{Required: true},
			"jumpoint_id":       schema.Int64Attribute{Required: true},
			"hostname":          schema.StringAttribute{Required: true},
			"jump_group_id":     schema.Int64Attribute{Required: true},
			"jump_group_type":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("shared"), Validators: jumpGroupTypeValidator()},
			"tag":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"comments":          schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"jump_policy_id":    schema.Int64Attribute{Optional: true},
			"session_policy_id": schema.Int64Attribute{Optional: true},
			"tunnel_listen_address": schema.StringAttribute{
				Description:   tunnelListenAddressDescription + tunnelListenAddressLocalAddressNote,
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{tunnelListenAddressValidator{}},
				PlanModifiers: []planmodifier.String{tunnelListenAddressPlan{}},
			},
			"username": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"database": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"auth_database": schema.StringAttribute{
				Description: "Database to authenticate against. When blank, the appliance uses `database`, then `admin`. " +
					"Setting a value requires PRA 26.3.1 or later; older appliances reject the field.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
		},
	}
}

// Update sends an emptied auth_database, which the generic Update would drop.
func (r *mongodbTunnelJumpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.update(ctx, req, resp, clearAuthDatabase)
}

// clearAuthDatabase sends auth_database as "" when the configuration clears a
// value the appliance holds. CopyTFtoAPI drops an empty *string, so without this
// the field is left out of the PATCH and the old value stays. State can only be
// non-empty on 26.3.1 or later, where "" is accepted, so this never sends the
// key to an appliance that rejects it. An unknown plan value never clears.
func clearAuthDatabase(state, plan models.MongodbTunnelJump, item *api.MongodbTunnelJump) {
	if isKnown(plan.AuthDatabase) && plan.AuthDatabase.ValueString() == "" && isSet(state.AuthDatabase) {
		empty := ""
		item.AuthDatabase = &empty
	}
}
