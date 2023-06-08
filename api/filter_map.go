package api

import (
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func MakeFilterMap(source any) map[string]string {
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
				filter[f] = final
			}
		}
	}

	return filter
}
