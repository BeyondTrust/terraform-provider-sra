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

func ResourceList() []func() resource.Resource {
	return []func() resource.Resource{
		newShellJumpResource,
	}
}

// resource Type, api model type, tf model type
type apiResource[T any, TApi api.APIResource, TTf any] struct {
	apiClient *api.APIClient
}

func (r *apiResource[T, TApi, TTf]) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil || r == nil {
		return
	}

	r.apiClient = req.ProviderData.(*api.APIClient)
}

func (r *apiResource[T, TApi, TTf]) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	var tmp T
	name := fmt.Sprintf("%s", reflect.TypeOf(tmp))
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

func copyTFtoAPI(ctx context.Context, tfObj reflect.Value, apiObj reflect.Value) {
	for i := 0; i < tfObj.NumField(); i++ {
		fieldName := tfObj.Type().Field(i).Name
		field := apiObj.FieldByName(fieldName)
		tfField := tfObj.Field(i)
		tflog.Info(ctx, "🍺 copyTFtoAPI field "+fieldName)

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

func (r *apiResource[T, TApi, TTf]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func copyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value) {
	for i := 0; i < tfObj.NumField(); i++ {
		fieldName := tfObj.Type().Field(i).Name
		field := apiObj.FieldByName(fieldName)
		tflog.Info(ctx, "🍺 copyAPItoTF field "+fieldName)

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
