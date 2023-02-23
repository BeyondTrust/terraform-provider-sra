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
	if req.ProviderData == nil {
		return
	}

	client := req.ProviderData.(*api.APIClient)

	tflog.Info(ctx, fmt.Sprintln("💥 ", reflect.TypeOf(r)))

	if r == nil {
		tflog.Info(ctx, fmt.Sprintln("💥 r is nil"))
		return
	}
	r.apiClient = client

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
			"Error creating shell jump item",
			"Unexpected reading shell jump ID ["+strconv.Itoa(id)+"]: "+err.Error(),
		)
		return
	}

	apiObj := reflect.ValueOf(item).Elem()
	for i := 0; i < tfObj.NumField(); i++ {
		fieldName := tfObj.Type().Field(i).Name
		field := apiObj.FieldByName(fieldName)

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
		case reflect.Int64:
			*(*types.Int64)(tfObj.Field(i).Addr().UnsafePointer()) = types.Int64Value(field.Int())
		default:

		}
	}
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
