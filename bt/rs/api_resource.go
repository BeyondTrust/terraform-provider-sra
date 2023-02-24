package rs

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"terraform-provider-beyondtrust-sra/api"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Factory function to return the list of all resource–generating factories to the main provider
// Add new resource factory functions here.
func ResourceList() []func() resource.Resource {
	return []func() resource.Resource{
		newShellJumpResource,
	}
}

// The base type that allows the other generic functions in this file to apply to the actual implementations.
// The actual resource struct must compose this struct to get all the functionality defined in this
// file. This has 3 generic types defined tha must be supplied. The first is the type of the
// resource provider itself, the second is the type of the API model, the third is the type
// of the Terraform model
type apiResource[T any, TApi api.APIResource, TTf any] struct {
	apiClient *api.APIClient
}

// Generic Configure function for resource providers. It simply maps the ProviderData as the API client on the resource
func (r *apiResource[T, TApi, TTf]) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil || r == nil {
		return
	}

	r.apiClient = req.ProviderData.(*api.APIClient)
}

// Generic Metadata implementation. It reads the type name of the resource type provided and derives the public facing resource
// name from that. It does this by dropping "Resource" from the type name and converting the rest to snake_case, which is
// prefixed with "bt_". For example, shellJumpResource is publicly exposed as bt_shell_jump
func (r *apiResource[T, TApi, TTf]) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	var tmp T
	name := reflect.TypeOf(tmp).String()
	parts := strings.Split(name, ".")

	resp.TypeName = fmt.Sprintf("%s_%s", req.ProviderTypeName, toSnakeCase(strings.ReplaceAll(parts[len(parts)-1], "Resource", "")))
	tflog.Info(ctx, fmt.Sprintf("🥃 Registered provider name [%s]", resp.TypeName))
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
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

func (r *apiResource[T, TApi, TTf]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TTf
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var item TApi

	tfObj := reflect.ValueOf(&plan).Elem()
	apiObj := reflect.ValueOf(&item).Elem()
	copyTFtoAPI(ctx, tfObj, apiObj)

	rb, _ := json.Marshal(item)
	tflog.Info(ctx, "🙀 executing jump item post", map[string]interface{}{
		"data": string(rb),
	})
	newItem, err := api.CreateItem(r.apiClient, item)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating jump item item",
			"Unexpected error: "+err.Error(),
		)
		return
	}

	newApiObj := reflect.ValueOf(newItem).Elem()
	copyAPItoTF(ctx, newApiObj, tfObj)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[T, TApi, TTf]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, fmt.Sprintln("Reading"))
	var state TTf
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfObj := reflect.ValueOf(&state).Elem()
	tfId := tfObj.FieldByName("ID").Interface().(types.String)
	id, _ := strconv.Atoi(tfId.ValueString())
	item, err := api.GetItem[TApi](r.apiClient, id)

	rb, _ := json.Marshal(item)
	tflog.Debug(ctx, "🙀 got item", map[string]interface{}{
		"data": string(rb),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating jump item item",
			"Unexpected reading jump item ID ["+strconv.Itoa(id)+"]: "+err.Error(),
		)
		return
	}

	apiObj := reflect.ValueOf(item).Elem()
	copyAPItoTF(ctx, apiObj, tfObj)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[T, TApi, TTf]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TTf
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var item TApi

	tfObj := reflect.ValueOf(&plan).Elem()
	apiObj := reflect.ValueOf(&item).Elem()
	copyTFtoAPI(ctx, tfObj, apiObj)

	rb, _ := json.Marshal(item)
	tflog.Info(ctx, "🙀 executing jump item update", map[string]interface{}{
		"data": string(rb),
	})
	newItem, err := api.UpdateItem(r.apiClient, item)
	if err != nil {
		tfId := tfObj.FieldByName("ID").Interface().(types.String)
		id, _ := strconv.Atoi(tfId.ValueString())
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating jump item item with id [%d]", id),
			"Unexpected error: "+err.Error(),
		)
		return
	}

	newApiObj := reflect.ValueOf(newItem).Elem()
	copyAPItoTF(ctx, newApiObj, tfObj)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *apiResource[T, TApi, TTf]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Starting delete")
	var state TTf
	diags := req.State.Get(ctx, &state)
	tflog.Info(ctx, "got state")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Info(ctx, "error getting state")
		return
	}
	tflog.Info(ctx, "deleting")

	tfObj := reflect.ValueOf(&state).Elem()
	tfId := tfObj.FieldByName("ID").Interface().(types.String)
	id, _ := strconv.Atoi(tfId.ValueString())
	err := api.DeleteItem[TApi](r.apiClient, id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting jump item with ID [%d]", id),
			"Could not delete item, unexpected error: "+err.Error(),
		)
		return
	}
}

// Generic ImportState implementation that just imports by ID
func (r *apiResource[T, TApi, TTf]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

/*
These next functions do the actual copying from TF -> API and API -> TF. They take a context parameter first
for logging purposes. The general idea for these is:

  1. loop over all the fields on one of the object (I chose to loop over the TF model's fields, but that's mostly arbitrary as they should have the same fields 😀)
  2. For each field:
	1. Get the field name so we can pull the same field from the API object (pulling by name in case the fields are in a different order… this is why names must match)
	2. Special handling for ID. Every model must have ID, and it is expected to be of type *int on the API model and base.String on the TF model.
		* This is because our IDs are actual numbers in the response json, but Terraform require IDs to be strings for the import command to work
		* ID can be null because the user does not specify an ID in POST requests
		* We allow null IDs on TF models but expect all API models to have an ID set
	3. Check to see if our API model describes this field as a pointer
		* if yes, check to see if the value is nil on the source model
		  * if nil, there is nothing to do
		  * if not nil, then replace "field" with the value of the pointer so we can set the value we're pointing to instead of the pointer itself
		    * if the destination is an API model, its pointer is likely nil, so we have to set the pointer to a new object of the appropriate type before dereferencing
	4. Set the value on the destination. This conversion is done based on the type of the API model field, because those are standard Go types that have reflect mappings
	    * Currently we only map int and string types. Other types will panic. Additional types will need to be added to the switch mappings as needed
*/

func copyTFtoAPI(ctx context.Context, tfObj reflect.Value, apiObj reflect.Value) {
	for i := 0; i < tfObj.NumField(); i++ {
		fieldName := tfObj.Type().Field(i).Name
		field := apiObj.FieldByName(fieldName)
		tfField := tfObj.Field(i)
		tflog.Trace(ctx, fmt.Sprintf("🍺 copyTFtoAPI field %s [%s]", fieldName, field.Kind()))

		if fieldName == "ID" {
			m := tfField.MethodByName("IsNull")
			mCallable := m.Interface().(func() bool)
			if !mCallable() {
				val := tfField.Interface().(types.String)
				id, _ := strconv.Atoi(val.ValueString())
				idVal := reflect.ValueOf(&id)
				field.Set(idVal)
			}
			continue
		}
		if field.Kind() == reflect.Pointer {
			m := tfField.MethodByName("IsNull")
			mCallable := m.Interface().(func() bool)
			if mCallable() {
				continue
			}
			if field.IsNil() {
				typeField, _ := apiObj.Type().FieldByName(fieldName)
				nestedKind := typeField.Type.Elem()
				field.Set(reflect.New(nestedKind))
			}
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.String:
			val := tfField.Interface().(types.String)
			field.SetString(val.ValueString())
		case reflect.Int:
			val := tfField.Interface().(types.Int64)
			field.SetInt(val.ValueInt64())
		default:
			panic("Unknown encoded type in struct: " + field.Kind().String())
		}
	}
}

func copyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value) {
	for i := 0; i < tfObj.NumField(); i++ {
		fieldName := tfObj.Type().Field(i).Name
		field := apiObj.FieldByName(fieldName)
		tflog.Trace(ctx, "🍺 copyAPItoTF field "+fieldName)

		// FIXME (maybe?) The reflect library doesn't have a nice wrapper method for setting
		// the Terraform types, and I didn't know enough about the other reflect
		// methods to set the pointer directly in a way that works. So these
		// ugly looking expressions get the raw pointer and set what it
		// points to with the proper value from the source model
		//
		// The unsafe pointer of the address of the field is a pointer to the TF type… we're setting
		// the dereferenced value of that. This is effectively what the nice reflect wrappers do
		// *(*types.String)(tfObj.Field(i).Addr().UnsafePointer())
		if fieldName == "ID" {
			val := field.Elem().Int()
			*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(strconv.Itoa(int(val)))
			continue
		}
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.String:
			*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(field.String())
		case reflect.Int:
			*(*types.Int64)(tfObj.Field(i).Addr().UnsafePointer()) = types.Int64Value(field.Int())
		default:
			panic("Unknown encoded type in struct: " + field.Kind().String())
		}
	}
}
