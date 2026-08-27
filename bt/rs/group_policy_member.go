package rs

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &groupPolicyMemberResource{}
	_ resource.ResourceWithConfigure        = &groupPolicyMemberResource{}
	_ resource.ResourceWithConfigValidators = &groupPolicyMemberResource{}
	_ resource.ResourceWithImportState      = &groupPolicyMemberResource{}

	// The appliance requires membership changes and provisioning to be handled
	// serially. This also makes empty-body create ID discovery deterministic for
	// provider operations in the same process.
	groupPolicyMemberMutex sync.Mutex
)

const (
	groupPolicyMemberPageSize     = 100
	groupPolicyMemberMaximumPages = 10000
)

func newGroupPolicyMemberResource() resource.Resource {
	return &groupPolicyMemberResource{}
}

type groupPolicyMemberResource struct {
	apiResource[api.GroupPolicyMember, models.GroupPolicyMember]
}

func (r *groupPolicyMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	replaceInt64 := []planmodifier.Int64{int64planmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user or group membership in a BeyondTrust Group Policy. Memberships are immutable relationships: changing the policy, security provider, or member selector replaces the relationship. Exactly one of `user_id`, `distinguished_name`, or `group_name` must be configured.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The numeric member identifier assigned by the appliance within the Group Policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_policy_id": schema.StringAttribute{
				Required:      true,
				Description:   "The numeric identifier of the Group Policy.",
				PlanModifiers: replaceString,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[1-9][0-9]*$`), "must be a positive numeric Group Policy ID"),
				},
			},
			"security_provider_id": schema.Int64Attribute{
				Required:      true,
				Description:   "The numeric identifier of the security provider containing the member.",
				PlanModifiers: replaceInt64,
				Validators: []validator.Int64{
					int64validator.Between(1, groupPolicyMaximumID),
				},
			},
			"distinguished_name": schema.StringAttribute{
				Optional:      true,
				Description:   "The distinguished name of an LDAP user or group, or a PRA SCIM user.",
				PlanModifiers: replaceString,
				Validators: []validator.String{
					stringvalidator.UTF8LengthBetween(1, 1024),
				},
			},
			"group_name": schema.StringAttribute{
				Optional:      true,
				Description:   "The name of a SAML group, or a PRA SCIM group.",
				PlanModifiers: replaceString,
				Validators: []validator.String{
					stringvalidator.UTF8LengthBetween(1, 64),
				},
			},
			"user_id": schema.Int64Attribute{
				Optional:      true,
				Description:   "The numeric identifier of a local, LDAP, RADIUS, Kerberos, or SAML user.",
				PlanModifiers: replaceInt64,
				Validators: []validator.Int64{
					int64validator.Between(1, groupPolicyMaximumID),
				},
			},
		},
	}
}

func (r *groupPolicyMemberResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("user_id"),
			path.MatchRoot("distinguished_name"),
			path.MatchRoot("group_name"),
		),
	}
}

func (r *groupPolicyMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.GroupPolicyMember
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupPolicyID, ok := parseGroupPolicyMemberID(plan.GroupPolicyID, "group_policy_id", &resp.Diagnostics)
	if !ok {
		return
	}

	member := groupPolicyMemberFromTerraform(plan, groupPolicyID)
	groupPolicyMemberMutex.Lock()
	defer groupPolicyMemberMutex.Unlock()

	before, err := matchingGroupPolicyMembers(r.ApiClient, member)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Existing Group Policy Members", "Could not inspect the Group Policy before adding its member: "+err.Error())
		return
	}

	created, err := api.CreateItem(r.ApiClient, member)
	if err != nil {
		resp.Diagnostics.AddError("Error Adding Group Policy Member", "The appliance rejected the Group Policy membership: "+err.Error())
		return
	}

	memberID := 0
	if created != nil && created.ID != nil {
		memberID = *created.ID
	} else {
		memberID, err = discoverCreatedGroupPolicyMemberID(r.ApiClient, member, before)
		if err != nil {
			resp.Diagnostics.AddError("Error Discovering Group Policy Member ID", err.Error())
			return
		}
	}

	if err := provisionGroupPolicy(r.ApiClient, groupPolicyID); err != nil {
		resp.Diagnostics.AddError("Error Provisioning Group Policy Members", "The member was added, but the Group Policy could not be provisioned: "+err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(memberID))
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *groupPolicyMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.GroupPolicyMember
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupPolicyID, ok := parseGroupPolicyMemberID(state.GroupPolicyID, "group_policy_id", &resp.Diagnostics)
	if !ok {
		return
	}
	memberID, ok := parseGroupPolicyMemberID(state.ID, "id", &resp.Diagnostics)
	if !ok {
		return
	}

	endpoint := fmt.Sprintf("group-policy/%s/member/%s", groupPolicyID, memberID)
	member, err := api.GetItemEndpoint[api.GroupPolicyMember](r.ApiClient, endpoint)
	if err != nil {
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Group Policy Member", "Could not read the Group Policy member: "+err.Error())
		return
	}

	state.SecurityProviderID = types.Int64Value(int64(member.SecurityProviderID))
	if !groupPolicyMemberSelectorMatchesState(*member, state) {
		if !setGroupPolicyMemberSelectorFromAPI(&state, *member) {
			resp.Diagnostics.AddError("Invalid Group Policy Member Response", "The appliance returned a member without user_id, distinguished_name, or group_name.")
			return
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *groupPolicyMemberResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected Group Policy Member Update", "Group Policy memberships are immutable and should be replaced instead of updated.")
}

func (r *groupPolicyMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.GroupPolicyMember
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupPolicyID, ok := parseGroupPolicyMemberID(state.GroupPolicyID, "group_policy_id", &resp.Diagnostics)
	if !ok {
		return
	}
	memberID, ok := parseGroupPolicyMemberID(state.ID, "id", &resp.Diagnostics)
	if !ok {
		return
	}

	groupPolicyMemberMutex.Lock()
	defer groupPolicyMemberMutex.Unlock()

	endpoint := fmt.Sprintf("group-policy/%s/member/%s", groupPolicyID, memberID)
	if err := api.DeleteItemEndpoint[api.GroupPolicyMember](r.ApiClient, endpoint); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Removing Group Policy Member", "Could not remove the Group Policy member: "+err.Error())
		return
	}

	if err := provisionGroupPolicy(r.ApiClient, groupPolicyID); err != nil {
		resp.Diagnostics.AddError("Error Provisioning Group Policy Members", "The member was removed, but the Group Policy could not be provisioned: "+err.Error())
	}
}

func (r *groupPolicyMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Group Policy Member Import ID", fmt.Sprintf("Use <group_policy_id>/<member_id>, for example 9/42; received %q.", req.ID))
		return
	}

	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 32)
		if err != nil || id < 1 {
			resp.Diagnostics.AddError("Invalid Group Policy Member Import ID", fmt.Sprintf("Both parts of <group_policy_id>/<member_id> must be numeric IDs from 1 through %d; received %q.", groupPolicyMaximumID, req.ID))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_policy_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func parseGroupPolicyMemberID(value types.String, attributeName string, diags interface {
	AddAttributeError(path.Path, string, string)
}) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		diags.AddAttributeError(path.Root(attributeName), "Invalid Group Policy Member ID", attributeName+" must be a known positive numeric ID.")
		return "", false
	}
	id, err := strconv.ParseInt(value.ValueString(), 10, 32)
	if err != nil || id < 1 {
		diags.AddAttributeError(path.Root(attributeName), "Invalid Group Policy Member ID", fmt.Sprintf("%s must be a numeric ID from 1 through %d.", attributeName, groupPolicyMaximumID))
		return "", false
	}
	return strconv.FormatInt(id, 10), true
}

func groupPolicyMemberFromTerraform(plan models.GroupPolicyMember, groupPolicyID string) api.GroupPolicyMember {
	member := api.GroupPolicyMember{
		GroupPolicyID:      &groupPolicyID,
		SecurityProviderID: int(plan.SecurityProviderID.ValueInt64()),
	}
	if !plan.DistinguishedName.IsNull() && !plan.DistinguishedName.IsUnknown() {
		value := plan.DistinguishedName.ValueString()
		member.DistinguishedName = &value
	}
	if !plan.GroupName.IsNull() && !plan.GroupName.IsUnknown() {
		value := plan.GroupName.ValueString()
		member.GroupName = &value
	}
	if !plan.UserID.IsNull() && !plan.UserID.IsUnknown() {
		value := int(plan.UserID.ValueInt64())
		member.UserID = &value
	}
	return member
}

func matchingGroupPolicyMembers(client *api.APIClient, wanted api.GroupPolicyMember) ([]api.GroupPolicyMember, error) {
	all, err := listGroupPolicyMembers(client, wanted.Endpoint())
	if err != nil {
		return nil, err
	}

	matches := make([]api.GroupPolicyMember, 0, 1)
	for _, member := range all {
		if groupPolicyMembersMatch(member, wanted) {
			matches = append(matches, member)
		}
	}
	return matches, nil
}

func listGroupPolicyMembers(client *api.APIClient, endpoint string) ([]api.GroupPolicyMember, error) {
	all := make([]api.GroupPolicyMember, 0)
	seen := make(map[int]struct{})
	for page := 1; page <= groupPolicyMemberMaximumPages; page++ {
		items, err := api.ListItemsEndpoint[api.GroupPolicyMember](client, endpoint, map[string]string{
			"per_page":     strconv.Itoa(groupPolicyMemberPageSize),
			"current_page": strconv.Itoa(page),
		})
		if err != nil {
			return nil, err
		}

		newItems := 0
		for _, item := range items {
			if item.ID == nil {
				continue
			}
			if _, exists := seen[*item.ID]; exists {
				continue
			}
			seen[*item.ID] = struct{}{}
			all = append(all, item)
			newItems++
		}
		if len(items) < groupPolicyMemberPageSize {
			return all, nil
		}
		if newItems == 0 {
			return nil, fmt.Errorf("pagination for %s repeated a full page without new member IDs", endpoint)
		}
	}
	return nil, fmt.Errorf("pagination for %s exceeded %d pages", endpoint, groupPolicyMemberMaximumPages)
}

func discoverCreatedGroupPolicyMemberID(client *api.APIClient, wanted api.GroupPolicyMember, before []api.GroupPolicyMember) (int, error) {
	existingIDs := make(map[int]struct{}, len(before))
	for _, member := range before {
		if member.ID != nil {
			existingIDs[*member.ID] = struct{}{}
		}
	}

	after, err := matchingGroupPolicyMembers(client, wanted)
	if err != nil {
		return 0, err
	}
	newIDs := make([]int, 0, 1)
	for _, member := range after {
		if member.ID == nil {
			continue
		}
		if _, existed := existingIDs[*member.ID]; !existed {
			newIDs = append(newIDs, *member.ID)
		}
	}
	if len(newIDs) != 1 {
		return 0, fmt.Errorf("the create response contained no ID and the member list contained %d newly matching members; expected exactly one", len(newIDs))
	}
	return newIDs[0], nil
}

func groupPolicyMembersMatch(actual, wanted api.GroupPolicyMember) bool {
	if actual.SecurityProviderID != wanted.SecurityProviderID {
		return false
	}
	if wanted.UserID != nil {
		return actual.UserID != nil && *actual.UserID == *wanted.UserID
	}
	if wanted.DistinguishedName != nil {
		return actual.DistinguishedName != nil && strings.EqualFold(strings.TrimSpace(*actual.DistinguishedName), strings.TrimSpace(*wanted.DistinguishedName))
	}
	if wanted.GroupName != nil {
		return actual.GroupName != nil && *actual.GroupName == *wanted.GroupName
	}
	return false
}

func groupPolicyMemberSelectorMatchesState(actual api.GroupPolicyMember, state models.GroupPolicyMember) bool {
	if !state.UserID.IsNull() && !state.UserID.IsUnknown() {
		return actual.UserID != nil && int64(*actual.UserID) == state.UserID.ValueInt64()
	}
	if !state.DistinguishedName.IsNull() && !state.DistinguishedName.IsUnknown() {
		return actual.DistinguishedName != nil && strings.EqualFold(strings.TrimSpace(*actual.DistinguishedName), strings.TrimSpace(state.DistinguishedName.ValueString()))
	}
	if !state.GroupName.IsNull() && !state.GroupName.IsUnknown() {
		return actual.GroupName != nil && *actual.GroupName == state.GroupName.ValueString()
	}
	return false
}

func setGroupPolicyMemberSelectorFromAPI(state *models.GroupPolicyMember, member api.GroupPolicyMember) bool {
	state.UserID = types.Int64Null()
	state.DistinguishedName = types.StringNull()
	state.GroupName = types.StringNull()

	if member.UserID != nil {
		state.UserID = types.Int64Value(int64(*member.UserID))
		return true
	}
	if member.DistinguishedName != nil {
		state.DistinguishedName = types.StringValue(*member.DistinguishedName)
		return true
	}
	if member.GroupName != nil {
		state.GroupName = types.StringValue(*member.GroupName)
		return true
	}
	return false
}

func provisionGroupPolicy(client *api.APIClient, groupPolicyID string) error {
	provision := api.GroupPolicyProvision{GroupPolicyID: &groupPolicyID}
	_, err := api.CreateItem(client, provision)
	return err
}
