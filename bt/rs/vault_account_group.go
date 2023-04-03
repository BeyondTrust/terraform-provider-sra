package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"
	"terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	mapset "github.com/deckarep/golang-set/v2"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &vaultAccountGroupResource{}
	_ resource.ResourceWithConfigure   = &vaultAccountGroupResource{}
	_ resource.ResourceWithImportState = &vaultAccountGroupResource{}
	// _ resource.ResourceWithModifyPlan  = &vaultAccountGroupResource{}
)

func newVaultAccountGroupResource() resource.Resource {
	return &vaultAccountGroupResource{}
}

type vaultAccountGroupResource struct {
	apiResource[api.VaultAccountGroup, models.VaultAccountGroup]
}

func (r *vaultAccountGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	jiaSchema := accountJumpItemAssociationSchema()

	var tfDefault types.Object
	tfDefault, _ = types.ObjectValue(
		map[string]attr.Type{"filter_type": types.StringType, "criteria": types.ObjectType{}},
		map[string]attr.Value{"filter_type": types.StringValue("any_jump_items"), "criteria": types.ObjectNull(map[string]attr.Type{})},
	)

	jiaSchema.Default = objectdefault.StaticValue(tfDefault)

	resp.Schema = schema.Schema{
		Description: "Manages a Vault Account Group.\n\nFor descriptions of individual fields, please see the Configuration API documentation on your SRA Appliance",
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
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"account_policy": schema.StringAttribute{
				Optional: true,
			},

			"jump_item_association": jiaSchema,
			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Account Group is a member of",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"role": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf([]string{"inject", "inject_and_checkout"}...),
							},
						},
					},
				},
			},
		},
	}
}

func (r *vaultAccountGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH reading state")

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	{
		// Jump Item Associations

		var apiSub api.AccountGroupJumpItemAssociation
		var tfObj types.Object
		diags := req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !tfObj.IsNull() {
			diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		apiSub.ID = &id
		tflog.Info(ctx, fmt.Sprintf("🙀 Reading API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data": apiSub,
		})

		item, err := api.GetItemEndpoint[api.AccountGroupJumpItemAssociation](r.ApiClient, apiSub.Endpoint())

		if item != nil && !tfObj.IsNull() {
			rb, _ := json.Marshal(item)
			tflog.Info(ctx, "🙀 got item", map[string]interface{}{
				"data": string(rb),
			})

			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading item",
					"Unexpected reading item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
			diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), item)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

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

		var gpList []api.GroupPolicyVaultAccountGroup
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range gpList {
			m.AccountGroupID = &id
			tflog.Info(ctx, "🌈 Reading item", map[string]interface{}{
				"read": m,
			})

			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), id)
			item, err := api.GetItemEndpoint[api.GroupPolicyVaultAccountGroup](r.ApiClient, endpoint)
			item.GroupPolicyID = m.GroupPolicyID

			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading item's group policy memberships",
					fmt.Sprintf("Unexpected reading membership of item ID [%d][%s]\n%s", id, endpoint, err.Error()),
				)
				return
			}
			gpList[i] = *item
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), gpList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *vaultAccountGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 SSH updating plan")

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	{
		// Jump Item Association

		var apiSub api.AccountGroupJumpItemAssociation
		var tfObj types.Object
		diags := req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !tfObj.IsNull() {
			diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		apiSub.ID = &id
		tflog.Info(ctx, fmt.Sprintf("🙀 Updating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data": apiSub,
		})

		var tfStateObj types.Object
		diags = req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfStateObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var item *api.AccountGroupJumpItemAssociation
		var err error
		if tfStateObj.IsNull() {
			item, err = api.CreateItem(r.ApiClient, apiSub)
		} else {
			item, err = api.UpdateItemEndpoint(r.ApiClient, apiSub, apiSub.Endpoint())
		}

		rb, _ := json.Marshal(item)
		tflog.Info(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading item",
				"Unexpected creating item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}
		diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), item)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	{
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var gpList []api.GroupPolicyVaultAccountGroup
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

		var stateGPList []api.GroupPolicyVaultAccountGroup
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
			m.AccountGroupID = &id
			tflog.Info(ctx, "🌈 Deleting item", map[string]interface{}{
				"add":     m,
				"gp":      m.GroupPolicyID,
				"account": m.AccountGroupID,
			})
			err := api.DeleteItemEndpoint[api.GroupPolicyVaultAccountGroup](r.ApiClient, m.Endpoint())

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected deleting membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
		}

		results := []api.GroupPolicyVaultAccountGroup{}
		for m := range toAdd.Iterator().C {
			m.AccountGroupID = &id
			tflog.Info(ctx, "🌈 Adding item", map[string]interface{}{
				"add":     m,
				"gp":      m.GroupPolicyID,
				"account": m.AccountGroupID,
			})
			_, err := api.CreateItem(r.ApiClient, m)
			// item := api.GroupPolicyVaultAccountGroup{
			// 	GroupPolicyID:  m.GroupPolicyID,
			// 	AccountGroupID: res.AccountGroupID,
			// 	Role:           res.Role,
			// }
			tflog.Info(ctx, "🌈 Added item", map[string]interface{}{
				"add": m,
			})

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected adding membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
			results = append(results, m)
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}
