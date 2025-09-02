package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"terraform-provider-sra/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Factory function to return the list of all resource–generating factories to the main provider
// Add new resource factory functions here.
func ResourceList() []func() resource.Resource {
	return []func() resource.Resource{
		newJumpGroupResource,
		newJumpointResource,

		newProtocolTunnelJumpResource,
		newRemoteRDPResource,
		newRemoteVNCResource,
		newShellJumpResource,
		newWebJumpResource,
		newJumpClientInstallerResource,
		newPostgreSQLTunnelJumpResource,
		newMySQLTunnelJumpResource,
		newNetworkTunnelJumpResource,

		newVaultAccountGroupResource,
		newVaultAccountPolicyResource,
		newVaultSSHAccountResource,
		newVaultUsernamePasswordAccountResource,
		newVaultTokenAccountResource,
	}
}

// The base type that allows the other generic functions in this file to apply to the actual implementations.
// The actual resource struct must compose this struct to get all the functionality defined in this
// file. This has 2 generic types defined tha must be supplied. The first is the type of the
// API model, the second is the type of the Terraform model
type apiResource[TApi api.APIResource, TTf any] struct {
	ApiClient *api.APIClient
}

// Generic Configure function for resource providers. It simply maps the ProviderData as the API client on the resource
func (r *apiResource[TApi, TTf]) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil || r == nil {
		return
	}

	r.ApiClient = req.ProviderData.(*api.APIClient)
}

// Generic Metadata implementation. It reads the type name of the resource type provided and derives the public facing resource
// name from that. It does this by dropping "Resource" from the type name and converting the rest to snake_case, which is
// prefixed with "sra_". For example, shellJumpResource is publicly exposed as sra_shell_jump
func (r *apiResource[TApi, TTf]) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s", req.ProviderTypeName, r.printableName())
	tflog.Debug(ctx, fmt.Sprintf("🥃 Registered provider name [%s]", resp.TypeName))
}

/*
The following are generic implementations of Create, Read, Update and Delete, which satisfy the basic requirements for a Terraform
resource. They work with the api client by using the API model type that is provided by the resource implementations. The API
client uses this type to infer the endpoints to query.

Largely the flow of these methods are all:
  1. Read the plan or state from the request, which is read into the Terraform model type.
  2. Convert this Terraform model to an API model (using reflect)
  3. Make the appropriate API request
  4. Copy the API response back to a Terraform model
  5. Set the updated Terraform model as the new plan or state in the response
  * also checks for errors when appropriate along the way

The conversion between API and Terraform modules are necessary because:
  * json encoding relies on the fields having standard Go types
  * terraform relies on its own type wrappers as the field types

For the conversion to work, some conventions **must** be followed:
  * the API model and the Terraform model must have the exact same fields, and the names must match exactly
  	* order of fields in the definition should not be important
  * Types should map correctly, that is a API model "string" should map to Terraform's "types.String"

Additionally, for Terraform to be happy:
  * If a field can be null in a response from the server, it should be a pointer to the type on the API model
    * Additionally, specify the omitempty hint on the json tag for the field
  * If a field can be null in a POST/PATCH request but will have some non-null value in the response,
    this should be mapped as a non-null type in the API model
  * For fields where we will supply a default value for fields not specified by the user, the defaults must be
    set on the plan by the resource in ModifyPlan. See shell_jump.go for specifics
*/

func (r *apiResource[TApi, TTf]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var item TApi
	if !api.IsProductAllowed(ctx, item) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", api.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), api.ProductName()),
		)
		return
	}

	var plan TTf
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("🤬 create plan [%v]", plan))

	tfObj := reflect.ValueOf(&plan).Elem()
	apiObj := reflect.ValueOf(&item).Elem()
	api.CopyTFtoAPI(ctx, tfObj, apiObj)

	rb, _ := json.Marshal(item)
	tflog.Debug(ctx, "🙀 executing item post", map[string]interface{}{
		"data": string(rb),
	})
	newItem, err := api.CreateItem(r.ApiClient, item)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating item",
			fmt.Sprintf("Unexpected error: [%s][%s]", err.Error(), string(rb)),
		)
		return
	}
	apiType := reflect.TypeOf(newItem).Elem()
	newApiObj := reflect.ValueOf(newItem).Elem()
	api.CopyAPItoTF(ctx, newApiObj, tfObj, apiType)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[TApi, TTf]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintln("Reading"))
	var testItem TApi
	if !api.IsProductAllowed(ctx, testItem) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", api.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), api.ProductName()),
		)
		return
	}

	var state TTf
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("🤬 read state [%v]", state))
	tfObj := reflect.ValueOf(&state).Elem()
	tfId := tfObj.FieldByName("ID").Interface().(types.String)
	id, _ := strconv.Atoi(tfId.ValueString())
	item, err := api.GetItem[TApi](r.ApiClient, &id)

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
	apiType := reflect.TypeOf(item).Elem()
	apiObj := reflect.ValueOf(item).Elem()
	api.CopyAPItoTF(ctx, apiObj, tfObj, apiType)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[TApi, TTf]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var item TApi
	if !api.IsProductAllowed(ctx, item) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", api.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), api.ProductName()),
		)
		return
	}

	var plan TTf
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("🤬 update plan [%v]", plan))

	tfObj := reflect.ValueOf(&plan).Elem()
	apiObj := reflect.ValueOf(&item).Elem()
	api.CopyTFtoAPI(ctx, tfObj, apiObj)

	rb, _ := json.Marshal(item)
	tflog.Debug(ctx, "🙀 executing item update", map[string]interface{}{
		"data": string(rb),
	})
	newItem, err := api.UpdateItem(r.ApiClient, item)
	if err != nil {
		tfId := tfObj.FieldByName("ID").Interface().(types.String)
		id, _ := strconv.Atoi(tfId.ValueString())
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating item with id [%d]", id),
			"Unexpected error: "+err.Error(),
		)
		return
	}

	newApiObj := reflect.ValueOf(newItem).Elem()
	apiType := reflect.TypeOf(newItem).Elem()
	api.CopyAPItoTF(ctx, newApiObj, tfObj, apiType)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[TApi, TTf]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting delete")
	var item TApi
	if !api.IsProductAllowed(ctx, item) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", api.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), api.ProductName()),
		)
		return
	}

	var state TTf
	diags := req.State.Get(ctx, &state)
	tflog.Debug(ctx, "got state")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "error getting state")
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("🤬 delete state [%v]", state))
	tflog.Debug(ctx, "deleting")

	tfObj := reflect.ValueOf(&state).Elem()
	tfId := tfObj.FieldByName("ID").Interface().(types.String)
	id, _ := strconv.Atoi(tfId.ValueString())
	err := api.DeleteItem[TApi](r.ApiClient, &id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting item with ID [%d]", id),
			"Could not delete item, unexpected error: "+err.Error(),
		)
		return
	}
}

// Generic ImportState implementation that just imports by ID
func (r *apiResource[TApi, TTf]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var item TApi
	if !api.IsProductAllowed(ctx, item) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", api.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), api.ProductName()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d *apiResource[TApi, TTf]) printableName() string {
	var tmp TApi
	name := reflect.TypeOf(tmp).String()
	parts := strings.Split(name, ".")

	return api.ToSnakeCase(parts[len(parts)-1])
}

// Jump Group type validator
func jumpGroupTypeValidator() []validator.String {
	return []validator.String{
		stringvalidator.OneOf([]string{"shared", "personal"}...),
	}
}

func accountJumpItemAssociationSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"filter_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"any_jump_items", "no_jump_items", "criteria"}...),
				},
			},
			"criteria": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"shared_jump_groups": schema.SetAttribute{
						ElementType: types.Int64Type,
						Optional:    true,
						Computed:    true,
						Default:     setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{})),
					},
					"host": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
					},
					"name": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
					},
					"tag": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
					},
					"comment": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
					},
				},
			},
			"jump_items": schema.SetNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  setdefault.StaticValue(types.SetValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.Int64Type, "type": types.StringType}}, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Required: true,
						},
						"type": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}
}
