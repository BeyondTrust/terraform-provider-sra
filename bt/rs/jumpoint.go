package rs

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource                = &jumpointResource{}
	_ resource.ResourceWithConfigure   = &jumpointResource{}
	_ resource.ResourceWithImportState = &jumpointResource{}
	// _ resource.ResourceWithModifyPlan  = &jumpointResource{}
)

func newJumpointResource() resource.Resource {
	return &jumpointResource{}
}

type jumpointResource struct {
	apiResource[api.Jumpoint, models.Jumpoint]
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

			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Jumpoint is a member of",
						},
					},
				},
			},
		},
	}
}

func (r *jumpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	{
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.State.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if tfGPList.IsNull() {
			return
		}

		var gpList []api.GroupPolicyJumpoint
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range gpList {
			m.JumpointID = &id
			tflog.Info(ctx, "🌈 Reading item", map[string]interface{}{
				"read": m,
			})
			gpId := *m.GroupPolicyID

			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), id)
			item, err := api.GetItemEndpoint[api.GroupPolicyJumpoint](r.ApiClient, endpoint)

			if err != nil {
				tflog.Info(ctx, "🌈 Error reading item item, skipping", map[string]interface{}{
					"read":  m,
					"error": err,
				})
			} else if item != nil {
				tflog.Info(ctx, "🌈 Read item", map[string]interface{}{
					"read": *item,
				})
				item.GroupPolicyID = &gpId
				gpList[i] = *item
			}
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), gpList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *jumpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	{
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var gpList []api.GroupPolicyJumpoint
		if !tfGPList.IsNull() {
			diags = tfGPList.ElementsAs(ctx, &gpList, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		var tfGPStateList types.Set
		diags = req.State.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPStateList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var stateGPList []api.GroupPolicyJumpoint
		if !tfGPStateList.IsNull() {
			diags = tfGPStateList.ElementsAs(ctx, &stateGPList, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		setGPList := mapset.NewSet(gpList...)
		setGPStateList := mapset.NewSet(stateGPList...)

		toAdd := setGPList.Difference(setGPStateList)
		toRemove := setGPStateList.Difference(setGPList)

		tflog.Info(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
			"add":    toAdd,
			"remove": toRemove,

			"tf":    tfGPList,
			"list":  gpList,
			"state": stateGPList,
		})

		for m := range toRemove.Iterator().C {
			m.JumpointID = &id
			tflog.Info(ctx, "🌈 Deleting item", map[string]interface{}{
				"add":      m,
				"gp":       m.GroupPolicyID,
				"jumpoint": m.JumpointID,
			})
			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), *m.JumpointID)
			err := api.DeleteItemEndpoint[api.GroupPolicyJumpoint](r.ApiClient, endpoint)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected deleting membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
		}

		results := []api.GroupPolicyJumpoint{}
		for m := range toAdd.Iterator().C {
			m.JumpointID = &id
			item, err := api.CreateItem(r.ApiClient, m)
			item.GroupPolicyID = m.GroupPolicyID

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected adding membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
			results = append(results, *item)
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}
