package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

			"jump_item_association": accountJumpItemAssociationSchema(),
			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Account is a member of",
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

func (r *vaultUsernamePasswordAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 User/Pass creating plan")

	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	updateJIA := func() {
		// Jump Item Association
		var apiSub api.AccountJumpItemAssociation
		var tfObj types.Object
		diags := req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if tfObj.IsNull() {
			return
		}

		if tfObj.IsUnknown() {
			diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), types.ObjectNull(tfObj.AttributeTypes(ctx)))
			resp.Diagnostics.Append(diags...)
			return
		}

		diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		apiSub.ID = &id
		tflog.Info(ctx, fmt.Sprintf("🙀 Creating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data": apiSub,
		})

		item, err := api.CreateItem(r.ApiClient, apiSub)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating User/Pass Account Jump Item Association",
				"Unexpected value for ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}

		rb, _ := json.Marshal(item)
		tflog.Info(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})
		diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), item)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateJIA()

	updateGP := func() {
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var gpList []api.GroupPolicyVaultAccount
		if !tfGPList.IsNull() {
			diags = tfGPList.ElementsAs(ctx, &gpList, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		setGPList := mapset.NewSet(gpList...)

		tflog.Info(ctx, "🌈 Adding group policy memberships", map[string]interface{}{
			"add": setGPList,

			"tf":   tfGPList,
			"list": gpList,
		})

		results := []api.GroupPolicyVaultAccount{}
		for m := range setGPList.Iterator().C {
			m.AccountID = &id
			item, err := api.CreateItem(r.ApiClient, m)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected adding membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
			item.GroupPolicyID = m.GroupPolicyID
			results = append(results, *item)
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateGP()
}

func (r *vaultUsernamePasswordAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 User/Pass reading state")

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	readJIA := func() {
		// Jump Item Association

		var apiSub api.AccountJumpItemAssociation
		var tfObj types.Object
		diags := req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		planIsGone := tfObj.IsNull() || tfObj.IsUnknown()

		if !planIsGone {
			diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		apiSub.ID = &id
		tflog.Info(ctx, fmt.Sprintf("🙀 Reading API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data":          apiSub,
			"planIsNull":    tfObj.IsNull(),
			"planIsUnknown": tfObj.IsUnknown(),
		})

		item, err := api.GetItemEndpoint[api.AccountJumpItemAssociation](r.ApiClient, apiSub.Endpoint())

		if item == nil && (planIsGone || apiSub.FilterType == "") {
			var empty api.AccountJumpItemAssociation
			diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), empty)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			return
		}

		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading item",
				"Unexpected reading item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}

		rb, _ := json.Marshal(item)
		tflog.Info(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})
		diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), item)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readJIA()

	readGP := func() {
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

		var gpList []api.GroupPolicyVaultAccount
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range gpList {
			m.AccountID = &id
			tflog.Info(ctx, "🌈 Reading item", map[string]interface{}{
				"read": m,
			})
			gpId := *m.GroupPolicyID

			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), id)
			item, err := api.GetItemEndpoint[api.GroupPolicyVaultAccount](r.ApiClient, endpoint)

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

	readGP()
}

func (r *vaultUsernamePasswordAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "🤬 User/Pass updating plan")
	var tfId types.String
	req.Plan.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	updateJIA := func() {
		// Jump Item Association

		var apiSub api.AccountJumpItemAssociation
		var tfObj types.Object
		diags := req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		planIsGone := tfObj.IsNull() || tfObj.IsUnknown()

		if !planIsGone {
			diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		var tfStateObj types.Object
		diags = req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfStateObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateIsGone := tfStateObj.IsNull() || tfStateObj.IsUnknown()

		apiSub.ID = &id
		tflog.Info(ctx, fmt.Sprintf("🤷🏻‍♂️ Updating User/Pass Jump Associations with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data":           apiSub,
			"planIsNull":     tfObj.IsNull(),
			"planIsUnknown":  tfObj.IsUnknown(),
			"stateIsNull":    tfStateObj.IsNull(),
			"stateIsUnknown": tfStateObj.IsUnknown(),
		})

		var item *api.AccountJumpItemAssociation
		var err error
		if !stateIsGone && planIsGone {
			tflog.Info(ctx, fmt.Sprintf("🦠 Deleting item %v", apiSub))
			err = api.DeleteItemEndpoint[api.AccountJumpItemAssociation](r.ApiClient, apiSub.Endpoint())
		} else if stateIsGone {
			tflog.Info(ctx, fmt.Sprintf("🦠 Creating item %v", apiSub))
			item, err = api.CreateItem(r.ApiClient, apiSub)
		} else {
			tflog.Info(ctx, fmt.Sprintf("🦠 Updating item %v", apiSub))
			item, err = api.UpdateItemEndpoint(r.ApiClient, apiSub, apiSub.Endpoint())
		}

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating User/Pass Account Jump Item Association",
				"Unexpected value for ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}

		if item != nil {
			tflog.Info(ctx, fmt.Sprintf("🦠 Setting item in plan %v", item))
			rb, _ := json.Marshal(item)
			tflog.Info(ctx, "🙀 got item", map[string]interface{}{
				"data": string(rb),
			})
			diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), item)
		} else {
			var empty api.AccountJumpItemAssociation
			tflog.Info(ctx, fmt.Sprintf("🦠 Setting empty item in plan %v", empty))
			diags = resp.State.SetAttribute(ctx, path.Root("jump_item_association"), empty)
		}
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateJIA()

	updateGP := func() {
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var gpList []api.GroupPolicyVaultAccount
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

		var stateGPList []api.GroupPolicyVaultAccount
		if !tfGPStateList.IsNull() {
			diags = tfGPStateList.ElementsAs(ctx, &stateGPList, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		if tfGPList.IsNull() && tfGPStateList.IsNull() {
			return
		}

		toAdd, toRemove, noChange := api.DiffGPAccountLists(gpList, stateGPList)

		tflog.Info(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
			"add":    toAdd,
			"remove": toRemove,

			"tf":    tfGPList,
			"list":  gpList,
			"state": stateGPList,
		})

		for m := range toRemove.Iterator().C {
			m.AccountID = &id
			tflog.Info(ctx, "🌈 Deleting item", map[string]interface{}{
				"add":     m,
				"gp":      m.GroupPolicyID,
				"account": m.AccountID,
			})
			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), *m.AccountID)
			err := api.DeleteItemEndpoint[api.GroupPolicyVaultAccount](r.ApiClient, endpoint)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected deleting membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
		}

		results := noChange.ToSlice()
		for m := range toAdd.Iterator().C {
			m.AccountID = &id
			item, err := api.CreateItem(r.ApiClient, m)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected adding membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
			item.GroupPolicyID = m.GroupPolicyID
			results = append(results, *item)
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateGP()
}
