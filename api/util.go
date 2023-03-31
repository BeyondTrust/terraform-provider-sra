package api

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

/*
These functions do the actual copying from TF -> API and API -> TF. They take a context parameter first
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

func CopyTFtoAPI(ctx context.Context, tfObj reflect.Value, apiObj reflect.Value) {
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
			m = tfField.MethodByName("IsUnknown")
			mCallable = m.Interface().(func() bool)
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
		case reflect.Bool:
			val := tfField.Interface().(types.Bool)
			field.SetBool(val.ValueBool())
		default:
			panic("Unknown encoded type in struct: " + field.Kind().String())
		}
	}
}

func CopyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value, apiType reflect.Type) {
	tflog.Info(ctx, fmt.Sprintf("🍺 copyAPItoTF source obj [%+v] ", apiObj))
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
			tflog.Info(ctx, fmt.Sprintf("🥃 ID [%d]", val))
			*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(strconv.Itoa(int(val)))
			continue
		}

		sraTag := tfObj.Type().Field(i).Tag.Get("sra")
		if sraTag != "" && slices.Contains(strings.Split(sraTag, ","), "persist_state") {
			continue
		}
		setToNil := false
		fieldKind := field.Kind()
		if fieldKind == reflect.Pointer {
			if field.IsNil() {
				setToNil = true
				fieldKind = apiType.Field(i).Type.Elem().Kind()
			} else {
				field = field.Elem()
				fieldKind = field.Kind()
			}
		}

		switch fieldKind {
		case reflect.String:
			if setToNil {
				*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringNull()
			} else {
				*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(field.String())
			}
		case reflect.Int:
			if setToNil {
				*(*types.Int64)(tfObj.Field(i).Addr().UnsafePointer()) = types.Int64Null()
			} else {
				*(*types.Int64)(tfObj.Field(i).Addr().UnsafePointer()) = types.Int64Value(field.Int())
			}
		case reflect.Bool:
			if setToNil {
				*(*types.Bool)(tfObj.Field(i).Addr().UnsafePointer()) = types.BoolNull()
			} else {
				*(*types.Bool)(tfObj.Field(i).Addr().UnsafePointer()) = types.BoolValue(field.Bool())
			}
		default:
			panic("Unknown encoded type in struct: " + field.Kind().String())
		}
	}
}

func MakeFilterMap(ctx context.Context, source any) map[string]string {

	typ := reflect.TypeOf(source)
	ste := reflect.ValueOf(source)
	var filter = make(map[string]string)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i).Tag.Get("filter")
		if f != "" {
			fld := ste.Field(i)
			final := ""
			switch fld.FieldByName("value").Kind() {
			case reflect.String:
				v := fld.Interface().(types.String)
				if !v.IsNull() {
					final = v.ValueString()
				}
			case reflect.Int64:
				v := fld.Interface().(types.Int64)
				if !v.IsNull() {
					final = strconv.Itoa(int(v.ValueInt64()))
				}
			}
			if final != "" {
				tflog.Info(ctx, fmt.Sprintf("🚀 %s=%s", f, final))
				filter[f] = final
			}
		}
	}

	return filter
}
