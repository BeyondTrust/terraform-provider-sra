package rs

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &jumpGroupResource{}
	_ resource.ResourceWithConfigure   = &jumpGroupResource{}
	_ resource.ResourceWithImportState = &jumpGroupResource{}
	_ resource.ResourceWithModifyPlan  = &jumpGroupResource{}

	// Because of the way the PHP code handles changing memberships, those
	// operations cannot be done in parallel. We use this mutex to ensure
	// we deal with membership updates one at a time
	jgMembershipMutex sync.Mutex
)

func newJumpGroupResource() resource.Resource {
	return &jumpGroupResource{}
}

type jumpGroupResource struct {
	apiResource[api.JumpGroup, models.JumpGroup]
}

func (r *jumpGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"group_policy_memberships": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_policy_id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the Group Policy this Account is a member of",
						},
						"jump_item_role_id": schema.Int64Attribute{
							Description: `The ID of the Jump Item Role that applies to this membership. Omitting or 0 means "User's Default"`,
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(0),
						},
						"jump_policy_id": schema.Int64Attribute{
							Description: `The ID of the Jump Policy that applies to this membership. Omitting or 0 means "Set on Jump Items"

This field only applies to PRA`,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (r *jumpGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Debug(ctx, "No plan to modify")
		return
	}

	/*
		Here we are setting some things that get defaults if they are not supplied.
	*/
	var tfGPList types.Set
	diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !tfGPList.IsNull() {
		var planList []models.GroupPolicyJumpGroup
		diags = tfGPList.ElementsAs(ctx, &planList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range planList {
			if api.IsPRA() {
				if m.JumpPolicyID.IsNull() || m.JumpPolicyID.IsUnknown() {
					m.JumpPolicyID = types.Int64Value(0)
				}
			} else {
				m.JumpPolicyID = types.Int64Null()
			}
			planList[i] = m
		}

		diags = resp.Plan.SetAttribute(ctx, path.Root("group_policy_memberships"), planList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Debug(ctx, "Finished modification")
}

func (r *jumpGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apiResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	updateGP := func() {
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var gpList []api.GroupPolicyJumpGroup
		if tfGPList.IsNull() {
			return
		}
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		toAdd := mapset.NewSet(gpList...)

		tflog.Trace(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
			"add": toAdd,

			"tf":   tfGPList,
			"list": gpList,
		})

		jgMembershipMutex.Lock()
		defer jgMembershipMutex.Unlock()

		results := []api.GroupPolicyJumpGroup{}
		needsProvision := mapset.NewSet[string]()
		for m := range toAdd.Iterator().C {
			m.JumpGroupID = &id
			tflog.Trace(ctx, "🌈 Adding item", map[string]interface{}{
				"add":         fmt.Sprintf("%+v", m),
				"gp":          m.GroupPolicyID,
				"jump group":  m.JumpGroupID,
				"jump policy": m.JumpPolicyID,
			})
			item, err := api.CreateItem(r.ApiClient, m)

			if err != nil {
				resp.Diagnostics.AddError(
					"Error adding item's group policy memberships",
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

func (r *jumpGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

		var gpList []api.GroupPolicyJumpGroup
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, m := range gpList {
			m.JumpGroupID = &id
			tflog.Trace(ctx, "🌈 Reading item", map[string]interface{}{
				"read": m,
			})
			gpId := *m.GroupPolicyID

			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), id)
			item, err := api.GetItemEndpoint[api.GroupPolicyJumpGroup](r.ApiClient, endpoint)

			if err != nil {
				tflog.Trace(ctx, "🌈 Error reading item item, skipping", map[string]interface{}{
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
}

func (r *jumpGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apiResource.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var tfId types.String
	req.State.GetAttribute(ctx, path.Root("id"), &tfId)
	id, _ := strconv.Atoi(tfId.ValueString())

	updateGP := func() {
		// Group Policy Memberships

		var tfGPList types.Set
		diags := req.Plan.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var gpList []api.GroupPolicyJumpGroup
		if tfGPList.IsNull() {
			return
		}
		diags = tfGPList.ElementsAs(ctx, &gpList, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var tfGPStateList types.Set
		diags = req.State.GetAttribute(ctx, path.Root("group_policy_memberships"), &tfGPStateList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var stateGPList []api.GroupPolicyJumpGroup
		if !tfGPStateList.IsNull() {
			diags = tfGPStateList.ElementsAs(ctx, &stateGPList, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		toAdd, toRemove, noChange := api.DiffGPJumpItemLists(gpList, stateGPList)

		tflog.Trace(ctx, "🌈 Updating group policy memberships", map[string]interface{}{
			"add":    toAdd,
			"remove": toRemove,

			"tf":    tfGPList,
			"list":  gpList,
			"state": stateGPList,
		})

		jgMembershipMutex.Lock()
		defer jgMembershipMutex.Unlock()

		needsProvision := mapset.NewSet[string]()
		for m := range toRemove.Iterator().C {
			m.JumpGroupID = &id
			tflog.Trace(ctx, "🌈 Deleting item", map[string]interface{}{
				"add":        m,
				"gp":         m.GroupPolicyID,
				"jump group": m.JumpGroupID,
			})
			endpoint := fmt.Sprintf("%s/%d", m.Endpoint(), *m.JumpGroupID)
			err := api.DeleteItemEndpoint[api.GroupPolicyJumpGroup](r.ApiClient, endpoint)

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
			m.JumpGroupID = &id
			tflog.Trace(ctx, "🌈 Adding item", map[string]interface{}{
				"add":         fmt.Sprintf("%+v", m),
				"gp":          m.GroupPolicyID,
				"jump group":  m.JumpGroupID,
				"jump policy": m.JumpPolicyID,
			})
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
