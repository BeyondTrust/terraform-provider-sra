package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

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

	// Because of the way the PHP code handles changing memberships, those
	// operations cannot be done in parallel. We use this mutex to ensure
	// we deal with membership updates one at a time
	agMembershipMutex sync.Mutex
)

func newVaultAccountGroupResource() resource.Resource {
	return &vaultAccountGroupResource{}
}

type vaultAccountGroupResource struct {
	apiResource[api.VaultAccountGroup, models.VaultAccountGroup]
}

func (r *vaultAccountGroupResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	jiaSchema := accountJumpItemAssociationSchema()

	criteriaDefaultType := types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
		"shared_jump_groups": types.SetType{}.WithElementType(types.Int64Type),
		"host":               types.SetType{}.WithElementType(types.StringType),
		"name":               types.SetType{}.WithElementType(types.StringType),
		"tag":                types.SetType{}.WithElementType(types.StringType),
		"comment":            types.SetType{}.WithElementType(types.StringType)})

	jiDefaultType := types.SetType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
		"id":   types.Int64Type,
		"type": types.StringType,
	}))

	jiDefault := types.SetValueMust(jiDefaultType.ElementType(), []attr.Value{})

	// var tfDefault types.Object
	tfDefault := types.ObjectValueMust(
		map[string]attr.Type{"filter_type": types.StringType, "criteria": criteriaDefaultType, "jump_items": jiDefaultType},
		map[string]attr.Value{
			"filter_type": types.StringValue("any_jump_items"),
			// "criteria": types.ObjectValueMust(criteriaDefaultType.AttributeTypes(), map[string]attr.Value{
			// 	"shared_jump_groups": types.SetNull(types.Int64Type),
			// 	"host":               types.SetNull(types.StringType),
			// 	"name":               types.SetNull(types.StringType),
			// 	"tag":                types.SetNull(types.StringType),
			// 	"comment":            types.SetNull(types.StringType),
			// }),
			"criteria":   types.ObjectNull(criteriaDefaultType.AttributeTypes()),
			"jump_items": jiDefault},
	)

	// tfDefault, _ = types.ObjectValueFrom(ctx, map[string]attr.Type{"filter_type": types.StringType, "criteria": criteriaDefaultType, "jump_items": jiDefaultType}, map[string]any{})

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

func (r *vaultAccountGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "🤬 Account Group creating plan")

	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	updateJIA := func() {
		// Jump Item Association

		var apiSub api.AccountGroupJumpItemAssociation
		var tfObj types.Object
		diags := req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if tfObj.IsNull() {
			return
		}

		diags = tfObj.As(ctx, &apiSub, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		apiSub.ID = &id
		tflog.Debug(ctx, fmt.Sprintf("🙀 Updating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data": apiSub,
		})

		var tfStateObj types.Object
		diags = req.Plan.GetAttribute(ctx, path.Root("jump_item_association"), &tfStateObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if apiSub.Criteria == nil {
			apiSub.Criteria = &api.JumpItemAssociationCriteria{}
		}

		var item *api.AccountGroupJumpItemAssociation
		var err error
		item, err = api.UpdateItemEndpoint(r.ApiClient, apiSub, apiSub.Endpoint())

		rb, _ := json.Marshal(item)
		tflog.Debug(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Account Group Jump Item Associations",
				"Unexpected value for ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}
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

		var gpList []api.GroupPolicyVaultAccountGroup
		if tfGPList.IsNull() {
			return
		}
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		setGPList := mapset.NewSet(gpList...)

		tflog.Trace(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
			"add": setGPList,

			"tf":   tfGPList,
			"list": gpList,
		})

		agMembershipMutex.Lock()
		defer agMembershipMutex.Unlock()

		results := []api.GroupPolicyVaultAccountGroup{}
		needsProvision := mapset.NewSet[string]()
		for m := range setGPList.Iterator().C {
			m.AccountGroupID = &id
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
			needsProvision.Add(*m.GroupPolicyID)
		}

		for id := range needsProvision.Iter() {
			p := api.GroupPolicyProvision{
				GroupPolicyID: &id,
			}
			_, err := api.CreateItem(r.ApiClient, p)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error provisioning item's group policy memberships",
					"Unexpected response provisioning membership of item ID ["+*p.GroupPolicyID+"]: "+err.Error(),
				)
				return
			}
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateGP()
}

func (r *vaultAccountGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.apiResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "🤬 Account Group reading state")

	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	readJIA := func() {
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
		tflog.Debug(ctx, fmt.Sprintf("🙀 Reading API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data": apiSub,
		})

		item, err := api.GetItemEndpoint[api.AccountGroupJumpItemAssociation](r.ApiClient, apiSub.Endpoint())

		if item != nil && !tfObj.IsNull() {
			rb, _ := json.Marshal(item)
			tflog.Debug(ctx, "🙀 got item", map[string]interface{}{
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

		var gpList []api.GroupPolicyVaultAccountGroup
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range gpList {
			m.AccountGroupID = &id
			tflog.Trace(ctx, "🌈 Reading item", map[string]interface{}{
				"read": m,
			})
			gpId := *m.GroupPolicyID

			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), id)
			item, err := api.GetItemEndpoint[api.GroupPolicyVaultAccountGroup](r.ApiClient, endpoint)

			if err != nil {
				tflog.Debug(ctx, "🌈 Error reading item item, skipping", map[string]interface{}{
					"read":  m,
					"error": err,
				})
			} else if item != nil {
				tflog.Trace(ctx, "🌈 Read item", map[string]interface{}{
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

func (r *vaultAccountGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "🤬 Account group updating plan")

	var tfId types.String
	req.Plan.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	updateJIA := func() {
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
		tflog.Debug(ctx, fmt.Sprintf("🙀 Updating API with ID %d [%s]", *apiSub.ID, apiSub.Endpoint()), map[string]interface{}{
			"data": apiSub,
		})

		var tfStateObj types.Object
		diags = req.State.GetAttribute(ctx, path.Root("jump_item_association"), &tfStateObj)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if apiSub.Criteria == nil {
			apiSub.Criteria = &api.JumpItemAssociationCriteria{}
		}

		var item *api.AccountGroupJumpItemAssociation
		var err error
		if tfStateObj.IsNull() {
			item, err = api.CreateItem(r.ApiClient, apiSub)
		} else {
			item, err = api.UpdateItemEndpoint(r.ApiClient, apiSub, apiSub.Endpoint())
		}

		rb, _ := json.Marshal(item)
		tflog.Trace(ctx, "🙀 got item", map[string]interface{}{
			"data": string(rb),
		})

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Account Group Jump Item Associations",
				"Unexpected value for ID ["+strconv.Itoa(id)+"]: "+err.Error(),
			)
			return
		}
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

		toAdd, toRemove, noChange := api.DiffGPAccountGroupLists(gpList, stateGPList)

		tflog.Trace(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
			"add":      fmt.Sprintf("%+v", toAdd),
			"remove":   fmt.Sprintf("%+v", toRemove),
			"noChange": fmt.Sprintf("%+v", noChange),

			"tfPlan":    fmt.Sprintf("%+v", tfGPList),
			"tfState":   fmt.Sprintf("%+v", tfGPStateList),
			"planList":  fmt.Sprintf("%+v", gpList),
			"stateList": fmt.Sprintf("%+v", stateGPList),
		})

		if tfGPList.IsNull() && tfGPStateList.IsNull() {
			return
		}

		agMembershipMutex.Lock()
		defer agMembershipMutex.Unlock()

		needsProvision := mapset.NewSet[string]()
		for m := range toRemove.Iterator().C {
			m.AccountGroupID = &id
			tflog.Trace(ctx, "🌈 Deleting item", map[string]interface{}{
				"add":     m,
				"gp":      m.GroupPolicyID,
				"account": m.AccountGroupID,
			})
			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), *m.AccountGroupID)
			err := api.DeleteItemEndpoint[api.GroupPolicyVaultAccountGroup](r.ApiClient, endpoint)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating item's group policy memberships",
					"Unexpected deleting membership of item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
				)
				return
			}
			needsProvision.Add(*m.GroupPolicyID)
		}

		results := noChange.ToSlice()
		for m := range toAdd.Iterator().C {
			m.AccountGroupID = &id
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
			needsProvision.Add(*m.GroupPolicyID)
		}

		for id := range needsProvision.Iter() {
			p := api.GroupPolicyProvision{
				GroupPolicyID: &id,
			}
			_, err := api.CreateItem(r.ApiClient, p)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error provisioning item's group policy memberships",
					"Unexpected response provisioning membership of item ID ["+*p.GroupPolicyID+"]: "+err.Error(),
				)
				return
			}
		}

		diags = resp.State.SetAttribute(ctx, path.Root("group_policy_memberships"), results)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateGP()
}
