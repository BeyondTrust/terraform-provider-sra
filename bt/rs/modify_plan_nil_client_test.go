package rs

import (
	"context"
	"terraform-provider-sra/bt/models"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// A3: apiResource.Configure leaves ApiClient nil when ProviderData is nil
// (e.g. Configure was never called, or the provider itself failed to
// configure). These four resources call r.ApiClient.IsPRA()/IsRS() directly
// in ModifyPlan, which would otherwise panic on a nil pointer dereference.
// Each subtest builds a real, schema-valid plan (all-null attributes) so the
// resource's own Get/GetAttribute calls succeed and execution actually
// reaches the guarded IsPRA()/IsRS() call.
func TestModifyPlan_NilApiClient(t *testing.T) {
	ctx := context.Background()

	t.Run("jump_client_installer", func(t *testing.T) {
		var schemaResp resource.SchemaResponse
		(&jumpClientInstallerResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		sch := schemaResp.Schema

		keyInfoType, ok := sch.Attributes["key_info"].GetType().(types.ObjectType)
		assert.True(t, ok)

		model := models.JumpClientInstaller{
			KeyInfo: types.ObjectNull(keyInfoType.AttrTypes),
		}
		plan := tfsdk.Plan{Schema: sch}
		diags := plan.Set(ctx, &model)
		assert.False(t, diags.HasError(), "%v", diags)

		r := jumpClientInstallerResource{}
		resp := &resource.ModifyPlanResponse{}
		assert.NotPanics(t, func() {
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan}, resp)
		})
	})

	t.Run("remote_rdp", func(t *testing.T) {
		var schemaResp resource.SchemaResponse
		(&remoteRDPResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		sch := schemaResp.Schema

		var model models.RemoteRDP // all scalar fields; zero value is null
		plan := tfsdk.Plan{Schema: sch}
		diags := plan.Set(ctx, &model)
		assert.False(t, diags.HasError(), "%v", diags)

		r := &remoteRDPResource{}
		resp := &resource.ModifyPlanResponse{}
		assert.NotPanics(t, func() {
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan}, resp)
		})
	})

	t.Run("jumpoint", func(t *testing.T) {
		var schemaResp resource.SchemaResponse
		(&jumpointResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		sch := schemaResp.Schema

		gpmType, ok := sch.Attributes["group_policy_memberships"].GetType().(types.SetType)
		assert.True(t, ok)

		model := models.Jumpoint{
			GroupPolicyMemberships: types.SetNull(gpmType.ElemType),
		}
		plan := tfsdk.Plan{Schema: sch}
		diags := plan.Set(ctx, &model)
		assert.False(t, diags.HasError(), "%v", diags)

		r := &jumpointResource{}
		resp := &resource.ModifyPlanResponse{}
		assert.NotPanics(t, func() {
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan}, resp)
		})
	})

	t.Run("jump_group", func(t *testing.T) {
		// jump_group.go's ModifyPlan only reads the group_policy_memberships
		// attribute (not the whole model), so the narrower gp-membership test
		// schema/scaffolding is sufficient here. IsPRA() is only called when
		// the set is non-empty, so it must carry at least one membership.
		sch := gpTestSchema()
		plan := tfsdk.Plan{Schema: sch, Raw: gpRaw([]tftypes.Value{gpMember("7")})}

		r := &jumpGroupResource{}
		resp := &resource.ModifyPlanResponse{}
		assert.NotPanics(t, func() {
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan}, resp)
		})
	})
}
