package api

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
)

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
		tfObjField := tfObj.Type().Field(i)
		fieldName := tfObjField.Name
		apiTypeField, found := apiObj.Type().FieldByName(fieldName)
		if !found || apiTypeField.Tag.Get("sraapi") != "" {
			// This attribute must be manually mapped to a different API object
			continue
		}
		prod := tfObjField.Tag.Get("sraproduct")
		if prod != "" {
			tflog.Debug(ctx, fmt.Sprintf("🍻 🔥 copyTFtoAPI check product for %s [%s][%s][%v]", fieldName, prod, product, strings.EqualFold(prod, product)))
			if !strings.EqualFold(prod, product) {
				tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI field Skipping %s as it's for [%s]", fieldName, prod))
				continue
			}
		}
		field := apiObj.FieldByName(fieldName)
		tfField := tfObj.Field(i)
		tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI field %s [apiKind=%s tfType=%s apiType=%s tfKind=%s]",
			fieldName, field.Kind(), tfObjField.Type.String(), apiTypeField.Type.String(), tfField.Kind()))

		// Special handling for structured FilterRules (TF: list<object> -> API: json.RawMessage)
		if tfObjField.Name == "FilterRules" {
			// tfField should be a types.List
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

			listVal := tfField.Interface().(types.List)
			var rawList []map[string]interface{}
			// ElementsAs will convert the TF nested objects into Go maps/slices
			_ = listVal.ElementsAs(ctx, &rawList, false)
			// reuse existing normalization logic from the string-based path
			var outList []map[string]interface{}
			for _, item := range rawList {
				if v, ok := item["ip_addresses"]; ok {
					switch vv := v.(type) {
					case []interface{}:
						var cidrs []string
						var plain []interface{}
						for _, elem := range vv {
							if s, ok := elem.(string); ok && strings.Contains(s, "/") {
								cidrs = append(cidrs, s)
							} else {
								plain = append(plain, elem)
							}
						}
						if len(cidrs) > 0 && len(plain) == 0 {
							for _, c := range cidrs {
								ni := make(map[string]interface{}, len(item))
								for k, v := range item {
									if k == "ip_addresses" {
										continue
									}
									ni[k] = v
								}
								ni["ip_addresses"] = map[string]interface{}{"cidr": c}
								if p, ok := ni["protocol"]; ok {
									if ps, ok := p.(string); ok {
										ni["protocol"] = strings.ToUpper(ps)
									}
								} else {
									ni["protocol"] = "ANY"
								}
								if pp, ok := ni["ports"]; ok {
									switch pvv := pp.(type) {
									case []interface{}:
										ni["ports"] = map[string]interface{}{"list": pvv}
									case map[string]interface{}:
										// ok
									default:
										ni["ports"] = map[string]interface{}{"list": []interface{}{}}
									}
								} else {
									ni["ports"] = map[string]interface{}{"list": []interface{}{}}
								}
								outList = append(outList, ni)
							}
							continue
						}
						if len(cidrs) > 0 && len(plain) > 0 {
							ni := make(map[string]interface{}, len(item))
							for k, v := range item {
								if k == "ip_addresses" {
									continue
								}
								ni[k] = v
							}
							ni["ip_addresses"] = map[string]interface{}{"list": plain}
							if p, ok := ni["protocol"]; ok {
								if ps, ok := p.(string); ok {
									ni["protocol"] = strings.ToUpper(ps)
								}
							} else {
								ni["protocol"] = "ANY"
							}
							if pp, ok := ni["ports"]; ok {
								switch pvv := pp.(type) {
								case []interface{}:
									ni["ports"] = map[string]interface{}{"list": pvv}
								case map[string]interface{}:
									// ok
								default:
									ni["ports"] = map[string]interface{}{"list": []interface{}{}}
								}
							} else {
								ni["ports"] = map[string]interface{}{"list": []interface{}{}}
							}
							outList = append(outList, ni)
							for _, c := range cidrs {
								ci := make(map[string]interface{}, len(item))
								for k, v := range item {
									if k == "ip_addresses" {
										continue
									}
									ci[k] = v
								}
								ci["ip_addresses"] = map[string]interface{}{"cidr": c}
								if p, ok := ci["protocol"]; ok {
									if ps, ok := p.(string); ok {
										ci["protocol"] = strings.ToUpper(ps)
									}
								} else {
									ci["protocol"] = "ANY"
								}
								if pp, ok := ci["ports"]; ok {
									switch pvv := pp.(type) {
									case []interface{}:
										ci["ports"] = map[string]interface{}{"list": pvv}
									case map[string]interface{}:
										// ok
									default:
										ci["ports"] = map[string]interface{}{"list": []interface{}{}}
									}
								} else {
									ci["ports"] = map[string]interface{}{"list": []interface{}{}}
								}
								outList = append(outList, ci)
							}
							continue
						}
						item["ip_addresses"] = map[string]interface{}{"list": vv}
					case string:
						if strings.Contains(vv, "/") {
							item["ip_addresses"] = map[string]interface{}{"cidr": vv}
						} else {
							item["ip_addresses"] = map[string]interface{}{"list": []interface{}{vv}}
						}
					case map[string]interface{}:
						// already in expected format
					}
				}
				if v, ok := item["ports"]; ok {
					switch vv := v.(type) {
					case []interface{}:
						item["ports"] = map[string]interface{}{"list": vv}
					case map[string]interface{}:
						// ok
					}
				}
				if v, ok := item["protocol"]; ok {
					if s, ok := v.(string); ok {
						item["protocol"] = strings.ToUpper(s)
					}
				}
				outList = append(outList, item)
			}
			newBytes, _ := json.Marshal(outList)
			var raw json.RawMessage = newBytes
			field.Set(reflect.ValueOf(&raw))
			continue
		}

		if fieldName == "ID" {
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

			val := tfField.Interface().(types.String)
			id, _ := strconv.Atoi(val.ValueString())
			idVal := reflect.ValueOf(&id)
			field.Set(idVal)
			continue
		}
		if field.Kind() == reflect.Pointer {
			m := tfField.MethodByName("IsNull")
			mCallable := m.Interface().(func() bool)
			if mCallable() {
				tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI field %s was null", fieldName))
				continue
			}
			m = tfField.MethodByName("IsUnknown")
			mCallable = m.Interface().(func() bool)
			if mCallable() {
				tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI field %s was unknown", fieldName))
				continue
			}

			if field.IsNil() {
				nestedKind := apiTypeField.Type.Elem()
				field.Set(reflect.New(nestedKind))
			}
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.String:
			val := tfField.Interface().(types.String)
			// Special case: filter_rules is stored as JSON on the API side as a raw message
			if tfObjField.Name == "FilterRules" {
				// If null/unknown/empty, leave API nil
				if val.IsNull() || val.IsUnknown() || val.ValueString() == "" {
					// leave as nil
				} else {
					// Parse the user-provided JSON and massage it into the API-expected shape.
					var list []map[string]interface{}
					if err := json.Unmarshal([]byte(val.ValueString()), &list); err == nil {
						var outList []map[string]interface{}
						for _, item := range list {
							// ip_addresses can be a bare array or string in TF examples; API expects an object
							if v, ok := item["ip_addresses"]; ok {
								switch vv := v.(type) {
								case []interface{}:
									// Partition elements into cidrs and plain IPs
									var cidrs []string
									var plain []interface{}
									for _, elem := range vv {
										if s, ok := elem.(string); ok && strings.Contains(s, "/") {
											cidrs = append(cidrs, s)
										} else {
											plain = append(plain, elem)
										}
									}
									if len(cidrs) > 0 && len(plain) == 0 {
										// all cidrs -> create one rule per cidr
										for _, c := range cidrs {
											ni := make(map[string]interface{}, len(item))
											for k, v := range item {
												if k == "ip_addresses" {
													continue
												}
												ni[k] = v
											}
											ni["ip_addresses"] = map[string]interface{}{"cidr": c}
											// normalize protocol
											if p, ok := ni["protocol"]; ok {
												if ps, ok := p.(string); ok {
													ni["protocol"] = strings.ToUpper(ps)
												}
											} else {
												ni["protocol"] = "ANY"
											}
											// normalize ports
											if pp, ok := ni["ports"]; ok {
												switch pvv := pp.(type) {
												case []interface{}:
													ni["ports"] = map[string]interface{}{"list": pvv}
												case map[string]interface{}:
													// ok
												default:
													ni["ports"] = map[string]interface{}{"list": []interface{}{}}
												}
											} else {
												ni["ports"] = map[string]interface{}{"list": []interface{}{}}
											}
											outList = append(outList, ni)
										}
										// continue to next original item
										continue
									}
									if len(cidrs) > 0 && len(plain) > 0 {
										// mixed -> keep a rule for plain list and separate rules for each cidr
										ni := make(map[string]interface{}, len(item))
										for k, v := range item {
											if k == "ip_addresses" {
												continue
											}
											ni[k] = v
										}
										ni["ip_addresses"] = map[string]interface{}{"list": plain}
										// normalize protocol and ports for ni
										if p, ok := ni["protocol"]; ok {
											if ps, ok := p.(string); ok {
												ni["protocol"] = strings.ToUpper(ps)
											}
										} else {
											ni["protocol"] = "ANY"
										}
										if pp, ok := ni["ports"]; ok {
											switch pvv := pp.(type) {
											case []interface{}:
												ni["ports"] = map[string]interface{}{"list": pvv}
											case map[string]interface{}:
												// ok
											default:
												ni["ports"] = map[string]interface{}{"list": []interface{}{}}
											}
										} else {
											ni["ports"] = map[string]interface{}{"list": []interface{}{}}
										}
										outList = append(outList, ni)
										for _, c := range cidrs {
											ci := make(map[string]interface{}, len(item))
											for k, v := range item {
												if k == "ip_addresses" {
													continue
												}
												ci[k] = v
											}
											ci["ip_addresses"] = map[string]interface{}{"cidr": c}
											// normalize protocol and ports for ci
											if p, ok := ci["protocol"]; ok {
												if ps, ok := p.(string); ok {
													ci["protocol"] = strings.ToUpper(ps)
												}
											} else {
												ci["protocol"] = "ANY"
											}
											if pp, ok := ci["ports"]; ok {
												switch pvv := pp.(type) {
												case []interface{}:
													ci["ports"] = map[string]interface{}{"list": pvv}
												case map[string]interface{}:
													// ok
												default:
													ci["ports"] = map[string]interface{}{"list": []interface{}{}}
												}
											} else {
												ci["ports"] = map[string]interface{}{"list": []interface{}{}}
											}
											outList = append(outList, ci)
										}
										continue
									}
									// no cidrs -> treat as list
									item["ip_addresses"] = map[string]interface{}{"list": vv}
								case string:
									if strings.Contains(vv, "/") {
										item["ip_addresses"] = map[string]interface{}{"cidr": vv}
									} else {
										item["ip_addresses"] = map[string]interface{}{"list": []interface{}{vv}}
									}
								case map[string]interface{}:
									// already in expected format
								}
							}
							// ports can be array or object
							if v, ok := item["ports"]; ok {
								switch vv := v.(type) {
								case []interface{}:
									item["ports"] = map[string]interface{}{"list": vv}
								case map[string]interface{}:
									// ok
								}
							}
							// protocol should be uppercased to match enum values
							if v, ok := item["protocol"]; ok {
								if s, ok := v.(string); ok {
									item["protocol"] = strings.ToUpper(s)
								}
							}
							outList = append(outList, item)
						}
						newBytes, _ := json.Marshal(outList)
						var raw json.RawMessage = newBytes
						field.Set(reflect.ValueOf(&raw))
					} else {
						// if invalid JSON, set empty to cause server-side error later
						empty := json.RawMessage([]byte("null"))
						field.Set(reflect.ValueOf(&empty))
					}
				}
			} else {
				field.SetString(val.ValueString())
			}
		case reflect.Int:
			val := tfField.Interface().(types.Int64)
			field.SetInt(val.ValueInt64())
		case reflect.Bool:
			val := tfField.Interface().(types.Bool)
			field.SetBool(val.ValueBool())

		case reflect.Slice:
			// Special-case for json.RawMessage on the API model (backed by []byte)
			if tfObjField.Name == "FilterRules" {
				val := tfField.Interface().(types.String)
				// If null/unknown/empty, leave API nil/empty
				if val.IsNull() || val.IsUnknown() || val.ValueString() == "" {
					// leave as nil/empty
				} else {
					var list []map[string]interface{}
					if err := json.Unmarshal([]byte(val.ValueString()), &list); err == nil {
						var outList []map[string]interface{}
						for _, item := range list {
							if v, ok := item["ip_addresses"]; ok {
								switch vv := v.(type) {
								case []interface{}:
									var cidrs []string
									var plain []interface{}
									for _, elem := range vv {
										if s, ok := elem.(string); ok && strings.Contains(s, "/") {
											cidrs = append(cidrs, s)
										} else {
											plain = append(plain, elem)
										}
									}
									if len(cidrs) > 0 && len(plain) == 0 {
										for _, c := range cidrs {
											ni := make(map[string]interface{}, len(item))
											for k, v := range item {
												if k == "ip_addresses" {
													continue
												}
												ni[k] = v
											}
											ni["ip_addresses"] = map[string]interface{}{"cidr": c}
											if p, ok := ni["protocol"]; ok {
												if ps, ok := p.(string); ok {
													ni["protocol"] = strings.ToUpper(ps)
												}
											} else {
												ni["protocol"] = "ANY"
											}
											if pp, ok := ni["ports"]; ok {
												switch pvv := pp.(type) {
												case []interface{}:
													ni["ports"] = map[string]interface{}{"list": pvv}
												case map[string]interface{}:
												default:
													ni["ports"] = map[string]interface{}{"list": []interface{}{}}
												}
											} else {
												ni["ports"] = map[string]interface{}{"list": []interface{}{}}
											}
											outList = append(outList, ni)
										}
										continue
									}
									if len(cidrs) > 0 && len(plain) > 0 {
										ni := make(map[string]interface{}, len(item))
										for k, v := range item {
											if k == "ip_addresses" {
												continue
											}
											ni[k] = v
										}
										ni["ip_addresses"] = map[string]interface{}{"list": plain}
										if p, ok := ni["protocol"]; ok {
											if ps, ok := p.(string); ok {
												ni["protocol"] = strings.ToUpper(ps)
											}
										} else {
											ni["protocol"] = "ANY"
										}
										if pp, ok := ni["ports"]; ok {
											switch pvv := pp.(type) {
											case []interface{}:
												ni["ports"] = map[string]interface{}{"list": pvv}
											case map[string]interface{}:
											default:
												ni["ports"] = map[string]interface{}{"list": []interface{}{}}
											}
										} else {
											ni["ports"] = map[string]interface{}{"list": []interface{}{}}
										}
										outList = append(outList, ni)
										for _, c := range cidrs {
											ci := make(map[string]interface{}, len(item))
											for k, v := range item {
												if k == "ip_addresses" {
													continue
												}
												ci[k] = v
											}
											ci["ip_addresses"] = map[string]interface{}{"cidr": c}
											if p, ok := ci["protocol"]; ok {
												if ps, ok := p.(string); ok {
													ci["protocol"] = strings.ToUpper(ps)
												}
											} else {
												ci["protocol"] = "ANY"
											}
											if pp, ok := ci["ports"]; ok {
												switch pvv := pp.(type) {
												case []interface{}:
													ci["ports"] = map[string]interface{}{"list": pvv}
												case map[string]interface{}:
												default:
													ci["ports"] = map[string]interface{}{"list": []interface{}{}}
												}
											} else {
												ci["ports"] = map[string]interface{}{"list": []interface{}{}}
											}
											outList = append(outList, ci)
										}
										continue
									}
									item["ip_addresses"] = map[string]interface{}{"list": vv}
								case string:
									if strings.Contains(vv, "/") {
										item["ip_addresses"] = map[string]interface{}{"cidr": vv}
									} else {
										item["ip_addresses"] = map[string]interface{}{"list": []interface{}{vv}}
									}
								case map[string]interface{}:
								}
							}
							if v, ok := item["ports"]; ok {
								switch vv := v.(type) {
								case []interface{}:
									item["ports"] = map[string]interface{}{"list": vv}
								case map[string]interface{}:
								}
							}
							if v, ok := item["protocol"]; ok {
								if s, ok := v.(string); ok {
									item["protocol"] = strings.ToUpper(s)
								}
							}
							outList = append(outList, item)
						}
						newBytes, _ := json.Marshal(outList)
						field.Set(reflect.ValueOf([]byte(newBytes)))
					} else {
						field.Set(reflect.ValueOf([]byte("null")))
					}
				}
			}

		default:
			// Log detailed runtime information instead of panicking so we can
			// capture what unexpected kinds/types are encountered during tests.
			tflog.Error(ctx, fmt.Sprintf("🍻 copyTFtoAPI unknown kind for field %s => apiKind=%s tfType=%s apiType=%s tfKind=%s",
				tfObjField.Name, field.Kind().String(), tfObjField.Type.String(), apiTypeField.Type.String(), tfField.Kind().String()))
			// Avoid crashing the provider during tests; skip this field so the request can proceed
			// (we'll use the logs to implement proper handling for the type seen).
			continue
		}
	}
}

func CopyAPItoTF(ctx context.Context, apiObj reflect.Value, tfObj reflect.Value, apiType reflect.Type) {
	tflog.Debug(ctx, fmt.Sprintf("🍺 copyAPItoTF source obj [%+v] [%v]", apiObj, IsRS()))
	for i := 0; i < tfObj.NumField(); i++ {
		tfObjField := tfObj.Type().Field(i)
		fieldName := tfObjField.Name
		apiTypeField, found := apiType.FieldByName(fieldName)
		if !found || apiTypeField.Tag.Get("sraapi") != "" {
			// This attribute must be manually mapped to a different API object
			continue
		}
		field := apiObj.FieldByName(fieldName)
		tflog.Debug(ctx, "🍺 copyAPItoTF field "+fieldName)

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
			tflog.Debug(ctx, fmt.Sprintf("🥃 ID [%d]", val))
			*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(strconv.Itoa(int(val)))
			continue
		}

		sraTag := tfObj.Type().Field(i).Tag.Get("sra")
		if sraTag != "" && slices.Contains(strings.Split(sraTag, ","), "persist_state") {
			continue
		}
		setToNil := false
		fieldKind := field.Kind()
		prod := tfObjField.Tag.Get("sraproduct")
		if prod != "" && !strings.EqualFold(prod, product) {
			tflog.Debug(ctx, fmt.Sprintf("🍺 copyAPItoTF field setting %s to nil as it's for [%s]", fieldName, prod))
			setToNil = true
			fieldKind = apiTypeField.Type.Elem().Kind()
		} else if fieldKind == reflect.Pointer {
			if field.IsNil() {
				setToNil = true
				fieldKind = apiTypeField.Type.Elem().Kind()
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
				// Special-case for FilterRules which may be a json.RawMessage on the API model
				if tfObjField.Name == "FilterRules" {
					// field is a reflect.Value referencing the RawMessage (string in JSON)
					rawBytes := []byte(field.String())
					if len(rawBytes) == 0 {
						*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringNull()
					} else {
						*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(string(rawBytes))
					}
				} else {
					*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(field.String())
				}
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
		case reflect.Slice:
			if setToNil {
				*(*types.Set)(tfObj.Field(i).Addr().UnsafePointer()) = types.SetNull(types.StringType)
			} else if field.Len() == 0 {
				*(*types.Set)(tfObj.Field(i).Addr().UnsafePointer()) = types.SetValueMust(types.StringType, []attr.Value{})
			} else {
				// If this is the FilterRules field, the API will provide []byte (json.RawMessage). Convert back to a string.
				if tfObjField.Name == "FilterRules" {
					rawBytes := field.Bytes()
					if len(rawBytes) == 0 {
						*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringNull()
					} else {
						// Try to unmarshal the API-returned JSON into a list and convert into TF nested objects
						var apiList []map[string]interface{}
						if err := json.Unmarshal(rawBytes, &apiList); err == nil {
							var elems []attr.Value
							for _, it := range apiList {
								// build attribute map for TF object
								attrMap := map[string]attr.Value{}
								// ip_addresses -> []string
								var ips []attr.Value
								if v, ok := it["ip_addresses"]; ok {
									switch vv := v.(type) {
									case map[string]interface{}:
										if cidr, ok := vv["cidr"]; ok {
											ips = append(ips, types.StringValue(fmt.Sprintf("%v", cidr)))
										} else if lst, ok := vv["list"]; ok {
											if arr, ok := lst.([]interface{}); ok {
												for _, e := range arr {
													ips = append(ips, types.StringValue(fmt.Sprintf("%v", e)))
												}
											}
										}
									case []interface{}:
										for _, e := range vv {
											ips = append(ips, types.StringValue(fmt.Sprintf("%v", e)))
										}
									case string:
										ips = append(ips, types.StringValue(vv))
									}
								}
								attrMap["ip_addresses"] = types.ListValueMust(types.StringType, ips)
								// ports -> nested object. API may return either {"list": [...]} or {"range": {"start": N, "end": M}}
								if v, ok := it["ports"]; ok {
									switch pv := v.(type) {
									case map[string]interface{}:
										// If it has a "list" key, convert elements to Int64 list
										if lst, ok := pv["list"]; ok {
											if arr, ok := lst.([]interface{}); ok {
												var pvals []attr.Value
												for _, e := range arr {
													// JSON numbers may unmarshal as float64
													switch vnum := e.(type) {
													case float64:
														pvals = append(pvals, types.Int64Value(int64(vnum)))
													case int:
														pvals = append(pvals, types.Int64Value(int64(vnum)))
													case int64:
														pvals = append(pvals, types.Int64Value(vnum))
													case string:
														// try parse
														i64, _ := strconv.ParseInt(vnum, 10, 64)
														pvals = append(pvals, types.Int64Value(i64))
													default:
														pvals = append(pvals, types.Int64Value(0))
													}
												}
												portsObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.Int64Type}, "range": types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}}}, map[string]attr.Value{"list": types.ListValueMust(types.Int64Type, pvals)})
												attrMap["ports"] = portsObj
											}
										} else if rng, ok := pv["range"]; ok {
											// Expect object with start and end
											if rmap, ok := rng.(map[string]interface{}); ok {
												var startVal, endVal attr.Value = types.Int64Null(), types.Int64Null()
												if s, ok := rmap["start"]; ok {
													switch sv := s.(type) {
													case float64:
														startVal = types.Int64Value(int64(sv))
													case int64:
														startVal = types.Int64Value(sv)
													case string:
														i64, _ := strconv.ParseInt(sv, 10, 64)
														startVal = types.Int64Value(i64)
													}
												}
												if e, ok := rmap["end"]; ok {
													switch ev := e.(type) {
													case float64:
														endVal = types.Int64Value(int64(ev))
													case int64:
														endVal = types.Int64Value(ev)
													case string:
														i64, _ := strconv.ParseInt(ev, 10, 64)
														endVal = types.Int64Value(i64)
													}
												}
												rangeObj := types.ObjectValueMust(map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}, map[string]attr.Value{"start": startVal, "end": endVal})
												portsObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.Int64Type}, "range": types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}}}, map[string]attr.Value{"range": rangeObj})
												attrMap["ports"] = portsObj
											}
										}
									case []interface{}:
										var pvals []attr.Value
										for _, e := range pv {
											switch vnum := e.(type) {
											case float64:
												pvals = append(pvals, types.Int64Value(int64(vnum)))
											case int:
												pvals = append(pvals, types.Int64Value(int64(vnum)))
											case int64:
												pvals = append(pvals, types.Int64Value(vnum))
											case string:
												i64, _ := strconv.ParseInt(vnum, 10, 64)
												pvals = append(pvals, types.Int64Value(i64))
											default:
												pvals = append(pvals, types.Int64Value(0))
											}
										}
										portsObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.Int64Type}, "range": types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}}}, map[string]attr.Value{"list": types.ListValueMust(types.Int64Type, pvals)})
										attrMap["ports"] = portsObj

									case string:
										// single value string -> try parse as int
										i64, _ := strconv.ParseInt(pv, 10, 64)
										portsObj := types.ObjectValueMust(map[string]attr.Type{"list": types.ListType{ElemType: types.Int64Type}, "range": types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.Int64Type, "end": types.Int64Type}}}, map[string]attr.Value{"list": types.ListValueMust(types.Int64Type, []attr.Value{types.Int64Value(i64)})})
										attrMap["ports"] = portsObj
									}
								}
								// protocol
								if v, ok := it["protocol"]; ok {
									if s, ok := v.(string); ok {
										attrMap["protocol"] = types.StringValue(s)
									}
								}
								// create object
								obj := types.ObjectValueMust(map[string]attr.Type{"ip_addresses": types.ListType{ElemType: types.StringType}, "ports": types.ListType{ElemType: types.StringType}, "protocol": types.StringType}, attrMap)
								elems = append(elems, obj)
							}
							// set the TF list
							listVal := types.ListValueMust(types.ObjectType{}, elems)
							// write into TF field
							*(*types.List)(tfObj.Field(i).Addr().UnsafePointer()) = listVal
						} else {
							// fallback: preserve raw API bytes as a string
							*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(string(rawBytes))
						}
					}
				} else {
					var goList []string
					rgList := reflect.ValueOf(&goList).Elem()
					rgList.Set(reflect.MakeSlice(field.Type(), field.Len(), field.Cap()))
					for j := 0; j < field.Len(); j++ {
						switch field.Index(j).Kind() {
						case reflect.String:
							rgList.Index(j).SetString(field.Index(j).Interface().(string))
						default:
							panic("Unhandled set type: " + field.Index(j).Kind().String())
						}
					}

					v, err := types.SetValueFrom(ctx, types.StringType, goList)
					if err != nil {
						panic("Error converting go set to TF object: " + err.Errors()[0].Detail())
					}
					*(*types.Set)(tfObj.Field(i).Addr().UnsafePointer()), _ = types.SetValueFrom(ctx, types.StringType, v)
				}
			}
		default:
			panic("Unknown encoded type in struct: " + field.Kind().String())
		}
	}
}
