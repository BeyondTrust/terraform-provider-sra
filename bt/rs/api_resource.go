package rs

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
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
	if !api.IsProductAllowed(ctx, item, r.ApiClient.Product) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", r.ApiClient.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), r.ApiClient.ProductName()),
		)
		return
	}

	var plan TTf
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "🤬 create plan")

	tfObj := reflect.ValueOf(&plan).Elem()
	apiObj := reflect.ValueOf(&item).Elem()
	api.CopyTFtoAPI(ctx, tfObj, apiObj, r.ApiClient.Product)

	tflog.Debug(ctx, "🙀 executing item post", map[string]interface{}{
		"endpoint": item.Endpoint(),
	})
	newItem, err := api.CreateItem(r.ApiClient, item)
	if err != nil {
		// The request body is deliberately absent from this message. Diagnostics are
		// surfaced to the operator and copied into bug reports regardless of TF_LOG,
		// and for the vault account resources the body is a credential.
		resp.Diagnostics.AddError(
			"Error creating item",
			fmt.Sprintf("Unexpected error: [%s]", err.Error()),
		)
		return
	}
	if newItem == nil {
		// CreateItem returns (nil, nil) on a 204 No Content. No documented create
		// endpoint does this today, but if one did, there would be no server-
		// assigned fields (e.g. ID) to populate computed attributes from — fail
		// loudly rather than writing a half-null state.
		resp.Diagnostics.AddError(
			"Error creating item",
			fmt.Sprintf("%s create returned no content; computed attributes could not be populated", r.printableName()),
		)
		return
	}
	apiType := reflect.TypeOf(newItem).Elem()
	newApiObj := reflect.ValueOf(newItem).Elem()
	if err := api.CopyAPItoTF(ctx, newApiObj, tfObj, apiType, r.ApiClient.Product); err != nil {
		resp.Diagnostics.AddError(
			"Error converting API response",
			"Unexpected error converting API response to Terraform state: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[TApi, TTf]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintln("Reading"))
	var testItem TApi
	if !api.IsProductAllowed(ctx, testItem, r.ApiClient.Product) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", r.ApiClient.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), r.ApiClient.ProductName()),
		)
		return
	}

	var state TTf
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "🤬 read state")
	tfObj := reflect.ValueOf(&state).Elem()
	tfId := tfObj.FieldByName("ID").Interface().(types.String)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID",
			fmt.Sprintf("Could not parse resource ID [%s] as integer: %s", tfId.ValueString(), err.Error()),
		)
		return
	}
	item, err := api.GetItem[TApi](r.ApiClient, &id)

	logItem(ctx, "🙀 got item", item)

	if err != nil {
		if api.IsNotFound(err) {
			// The item was deleted out-of-band. Remove it from state so Terraform
			// plans to recreate it (or drop it) instead of failing the refresh.
			tflog.Debug(ctx, fmt.Sprintf("Item ID [%d] not found; removing from state", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading item",
			"Unexpected reading item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
		)
		return
	}
	apiType := reflect.TypeOf(item).Elem()
	apiObj := reflect.ValueOf(item).Elem()
	if err := api.CopyAPItoTF(ctx, apiObj, tfObj, apiType, r.ApiClient.Product); err != nil {
		resp.Diagnostics.AddError(
			"Error converting API response",
			"Unexpected error converting API response to Terraform state: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[TApi, TTf]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var item TApi
	if !api.IsProductAllowed(ctx, item, r.ApiClient.Product) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", r.ApiClient.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), r.ApiClient.ProductName()),
		)
		return
	}

	var plan TTf
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "🤬 update plan")

	tfObj := reflect.ValueOf(&plan).Elem()
	apiObj := reflect.ValueOf(&item).Elem()
	api.CopyTFtoAPI(ctx, tfObj, apiObj, r.ApiClient.Product)

	logItem(ctx, "🙀 executing item update", item)
	newItem, err := api.UpdateItem(r.ApiClient, item)
	if err != nil {
		tfId := tfObj.FieldByName("ID").Interface().(types.String)
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating item with id [%s]", tfId.ValueString()),
			"Unexpected error: "+err.Error(),
		)
		return
	}

	newApiObj := reflect.ValueOf(newItem).Elem()
	apiType := reflect.TypeOf(newItem).Elem()
	if err := api.CopyAPItoTF(ctx, newApiObj, tfObj, apiType, r.ApiClient.Product); err != nil {
		resp.Diagnostics.AddError(
			"Error converting API response",
			"Unexpected error converting API response to Terraform state: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[TApi, TTf]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting delete")
	var item TApi
	if !api.IsProductAllowed(ctx, item, r.ApiClient.Product) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", r.ApiClient.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), r.ApiClient.ProductName()),
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
	tflog.Debug(ctx, "🤬 delete state")
	tflog.Debug(ctx, "deleting")

	tfObj := reflect.ValueOf(&state).Elem()
	tfId := tfObj.FieldByName("ID").Interface().(types.String)
	id, err := strconv.Atoi(tfId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid resource ID",
			fmt.Sprintf("Could not parse resource ID [%s] as integer: %s", tfId.ValueString(), err.Error()),
		)
		return
	}
	err = api.DeleteItem[TApi](r.ApiClient, &id)
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
	if !api.IsProductAllowed(ctx, item, r.ApiClient.Product) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s can't be used with a %s resource", r.ApiClient.ProductName(), r.printableName()),
			fmt.Sprintf("The %s resource can't be used when BT_API_HOST is configured for a %s site.", r.printableName(), r.ApiClient.ProductName()),
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
// groupPolicyIDPattern matches the form the configuration API documents for a
// group policy ID. The spec types the path parameter as
// `integer, format: int32, minimum: 1` (openapi/bt-pra-configuration.openapi.yaml,
// the ObjectId parameter), and every example in this repo sources the value from
// `data.sra_group_policy_list.gp.items[0].id`.
//
// The attribute is a string on the Terraform side, so nothing previously stopped a
// configuration supplying something that is not an ID at all. That value is
// interpolated into the request path by the Endpoint() methods in api/models.go.
var groupPolicyIDPattern = regexp.MustCompile(`^[0-9]+$`)

// groupPolicyIDValidators constrains group_policy_id to that documented form.
//
// Used by every resource exposing the attribute, so the six declarations cannot
// drift apart. Pairs with url.PathEscape in the Endpoint() methods: this keeps
// non-conforming values out, and the escape means a path segment stays one segment
// regardless.
func groupPolicyIDValidators() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(
			groupPolicyIDPattern,
			"must be a numeric group policy ID, as returned by the sra_group_policy_list data source",
		),
	}
}

// logItem records that an item was handled, without recording the item.
//
// The generic paths carry every resource type, and build their item from the
// Terraform plan — so the struct holds whatever the configuration set, including
// write-only attributes. Logging the type keeps the trace useful for following a
// request through the provider; the values are not the provider's to write out.
func logItem(ctx context.Context, msg string, item any) {
	// Type only. Endpoint() is deliberately NOT called here: several
	// implementations dereference an ID that is not yet populated at the point
	// these logs fire (AccountGroupJumpItemAssociation.Endpoint does *a.ID), so
	// calling it turns a log line into a nil-pointer panic. A logging helper must
	// not be able to fail the operation it is describing.
	tflog.Debug(ctx, msg, map[string]interface{}{"type": fmt.Sprintf("%T", item)})
}

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
