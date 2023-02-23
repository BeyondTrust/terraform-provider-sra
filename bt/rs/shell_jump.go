package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"terraform-provider-beyondtrust-sra/api"
	. "terraform-provider-beyondtrust-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &shellJumpResource{}
	_ resource.ResourceWithConfigure   = &shellJumpResource{}
	_ resource.ResourceWithImportState = &shellJumpResource{}
	_ resource.ResourceWithModifyPlan  = &shellJumpResource{}
)

func newShellJumpResource() resource.Resource {
	return &shellJumpResource{}
}

type shellJumpResource struct {
	apiResource[shellJumpResource, api.ShellJump, ShellJumpModel]
}

func (r *shellJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"jumpoint_id": schema.Int64Attribute{
				Required: true,
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"protocol": schema.StringAttribute{
				Required: true,
			},
			"port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"jump_group_id": schema.Int64Attribute{
				Required: true,
			},
			"jump_group_type": schema.StringAttribute{
				Required: true,
			},
			"terminal": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"keep_alive": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"comments": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"jump_policy_id": schema.Int64Attribute{
				Optional: true,
			},
			"session_policy_id": schema.Int64Attribute{
				Optional: true,
			},
		},
	}
}

func (r *shellJumpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Info(ctx, "Starting plan modification")
	if req.Plan.Raw.IsNull() {
		tflog.Info(ctx, "No plan to modify")
		return
	}
	var plan ShellJumpModel
	diags := req.Plan.Get(ctx, &plan)
	tflog.Info(ctx, "Read plan")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Info(ctx, "Error reading plan")
		return
	}
	if plan.Terminal.ValueString() == "" {
		tflog.Info(ctx, "plan.Terminal was null, setting default")
		plan.Terminal = types.StringValue("xterm")
	}
	if plan.Port.ValueInt64() == 0 {
		if plan.Protocol.ValueString() != "ssh" {
			tflog.Info(ctx, "plan.Port was null, setting default as 23")
			plan.Port = types.Int64Value(23)
		} else {
			tflog.Info(ctx, "plan.Port was null, setting default as 22")
			plan.Port = types.Int64Value(22)
		}
	}

	// Just convert null values to the empty default representations
	plan.Username = types.StringValue(plan.Username.ValueString())
	plan.KeepAlive = types.Int64Value(plan.KeepAlive.ValueInt64())
	plan.Tag = types.StringValue(plan.Tag.ValueString())
	plan.Comments = types.StringValue(plan.Comments.ValueString())

	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Finished modification")
}

func (r *shellJumpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ShellJumpModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	item := api.ShellJump{
		Name:          plan.Name.ValueString(),
		JumpointID:    int(plan.JumpointID.ValueInt64()),
		Hostname:      plan.Hostname.ValueString(),
		Protocol:      plan.Protocol.ValueString(),
		JumpGroupID:   int(plan.JumpGroupID.ValueInt64()),
		JumpGroupType: plan.JumpGroupType.ValueString(),

		Terminal:  plan.Terminal.ValueString(),
		Port:      int(plan.Port.ValueInt64()),
		Username:  plan.Username.ValueString(),
		KeepAlive: int(plan.KeepAlive.ValueInt64()),
		Tag:       plan.Tag.ValueString(),
		Comments:  plan.Comments.ValueString(),
	}

	if !plan.JumpPolicyID.IsNull() {
		val := int(plan.JumpPolicyID.ValueInt64())
		item.JumpPolicyID = &val
	}
	if !plan.SessionPolicyID.IsNull() {
		val := int(plan.SessionPolicyID.ValueInt64())
		item.SessionPolicyID = &val
	}

	rb, _ := json.Marshal(item)
	tflog.Info(ctx, "🙀 executing shell jump post", map[string]interface{}{
		"data": string(rb),
	})
	shellJump, err := api.CreateItem(r.apiClient, item)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating shell jump item",
			"Unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(int(*shellJump.ID)))
	plan.Name = types.StringValue(shellJump.Name)
	plan.JumpointID = types.Int64Value(int64(shellJump.JumpointID))
	plan.Hostname = types.StringValue(shellJump.Hostname)
	plan.Protocol = types.StringValue(shellJump.Protocol)
	plan.JumpGroupID = types.Int64Value(int64(shellJump.JumpGroupID))
	plan.JumpGroupType = types.StringValue(shellJump.JumpGroupType)
	plan.KeepAlive = types.Int64Value(int64(shellJump.KeepAlive))
	plan.Tag = types.StringValue(shellJump.Tag)
	plan.Comments = types.StringValue(shellJump.Comments)
	plan.Username = types.StringValue(shellJump.Username)
	plan.Port = types.Int64Value(int64(shellJump.Port))
	plan.Terminal = types.StringValue(shellJump.Terminal)

	if shellJump.JumpPolicyID != nil {
		plan.JumpPolicyID = types.Int64Value(int64(*shellJump.JumpPolicyID))
	}
	if shellJump.SessionPolicyID != nil {
		plan.SessionPolicyID = types.Int64Value(int64(*shellJump.SessionPolicyID))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// func (r *shellJumpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
// 	var state ShellJumpModel
// 	diags := req.State.Get(ctx, &state)
// 	resp.Diagnostics.Append(diags...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	id, _ := strconv.Atoi(state.ID.ValueString())
// 	shellJump, err := api.GetItem[api.ShellJump](r.apiClient, id)
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"Error creating shell jump item",
// 			"Unexpected reading shell jump ID ["+strconv.Itoa(id)+"]: "+err.Error(),
// 		)
// 		return
// 	}

// 	state.ID = types.StringValue(strconv.Itoa(int(*shellJump.ID)))
// 	state.Name = types.StringValue(shellJump.Name)
// 	state.JumpointID = types.Int64Value(int64(shellJump.JumpointID))
// 	state.Hostname = types.StringValue(shellJump.Hostname)
// 	state.Protocol = types.StringValue(shellJump.Protocol)
// 	state.Port = types.Int64Value(int64(shellJump.Port))
// 	state.JumpGroupID = types.Int64Value(int64(shellJump.JumpGroupID))
// 	state.JumpGroupType = types.StringValue(shellJump.JumpGroupType)
// 	state.Terminal = types.StringValue(shellJump.Terminal)
// 	state.KeepAlive = types.Int64Value(int64(shellJump.KeepAlive))
// 	state.Tag = types.StringValue(shellJump.Tag)
// 	state.Comments = types.StringValue(shellJump.Comments)
// 	state.Username = types.StringValue(shellJump.Username)

// 	if shellJump.JumpPolicyID != nil {
// 		state.JumpPolicyID = types.Int64Value(int64(*shellJump.JumpPolicyID))
// 	}
// 	if shellJump.SessionPolicyID != nil {
// 		state.SessionPolicyID = types.Int64Value(int64(*shellJump.SessionPolicyID))
// 	}
// 	diags = resp.State.Set(ctx, &state)
// 	resp.Diagnostics.Append(diags...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}
// }

func (r *shellJumpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ShellJumpModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, _ := strconv.Atoi(plan.ID.ValueString())
	item := api.ShellJump{
		ID:            &id,
		Name:          plan.Name.ValueString(),
		JumpointID:    int(plan.JumpointID.ValueInt64()),
		Hostname:      plan.Hostname.ValueString(),
		Protocol:      plan.Protocol.ValueString(),
		JumpGroupID:   int(plan.JumpGroupID.ValueInt64()),
		JumpGroupType: plan.JumpGroupType.ValueString(),

		Terminal:  plan.Terminal.ValueString(),
		Port:      int(plan.Port.ValueInt64()),
		Username:  plan.Username.ValueString(),
		KeepAlive: int(plan.KeepAlive.ValueInt64()),
		Tag:       plan.Tag.ValueString(),
		Comments:  plan.Comments.ValueString(),
	}

	if !plan.JumpPolicyID.IsNull() {
		val := int(plan.JumpPolicyID.ValueInt64())
		item.JumpPolicyID = &val
	}
	if !plan.SessionPolicyID.IsNull() {
		val := int(plan.SessionPolicyID.ValueInt64())
		item.SessionPolicyID = &val
	}

	rb, _ := json.Marshal(item)
	tflog.Info(ctx, "🙀 executing shell jump update", map[string]interface{}{
		"data": string(rb),
	})
	shellJump, err := api.UpdateItem(r.apiClient, item)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating shell jump item with id [%d]", id),
			"Unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(int(*shellJump.ID)))
	plan.Name = types.StringValue(shellJump.Name)
	plan.JumpointID = types.Int64Value(int64(shellJump.JumpointID))
	plan.Hostname = types.StringValue(shellJump.Hostname)
	plan.Protocol = types.StringValue(shellJump.Protocol)
	plan.JumpGroupID = types.Int64Value(int64(shellJump.JumpGroupID))
	plan.JumpGroupType = types.StringValue(shellJump.JumpGroupType)
	plan.KeepAlive = types.Int64Value(int64(shellJump.KeepAlive))
	plan.Tag = types.StringValue(shellJump.Tag)
	plan.Comments = types.StringValue(shellJump.Comments)
	plan.Username = types.StringValue(shellJump.Username)
	plan.Port = types.Int64Value(int64(shellJump.Port))
	plan.Terminal = types.StringValue(shellJump.Terminal)

	if shellJump.JumpPolicyID != nil {
		plan.JumpPolicyID = types.Int64Value(int64(*shellJump.JumpPolicyID))
	}
	if shellJump.SessionPolicyID != nil {
		plan.SessionPolicyID = types.Int64Value(int64(*shellJump.SessionPolicyID))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *shellJumpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Starting delete")
	var state ShellJumpModel
	diags := req.State.Get(ctx, &state)
	tflog.Info(ctx, "got state")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Info(ctx, "error getting state")
		return
	}
	tflog.Info(ctx, "deleting")

	id, _ := strconv.Atoi(state.ID.ValueString())
	err := api.DeleteItem[api.ShellJump](r.apiClient, id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting sell jump with ID [%d]", id),
			"Could not delete item, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *shellJumpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
