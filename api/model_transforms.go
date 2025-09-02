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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
)

type NetworkConfig struct {
	IPAddresses IPAddressConfig `json:"ip_addresses" tfsdk:"ip_addresses"`
	Ports       types.Object    `json:"ports,omitempty" tfsdk:"ports"`
	Protocol    types.String    `json:"protocol" tfsdk:"protocol"`
}

type IPAddressConfig struct {
	CIDR  types.String `json:"cidr,omitempty" tfsdk:"cidr"`
	List  types.List   `json:"list,omitempty" tfsdk:"list"`
	Range types.Object `json:"range,omitempty" tfsdk:"range"`
}

type PortConfig struct {
	List  types.List   `json:"list,omitempty" tfsdk:"list"`
	Range types.Object `json:"range,omitempty" tfsdk:"range"`
}

type IPRange struct {
	Start types.String `json:"start" tfsdk:"start"`
	End   types.String `json:"end" tfsdk:"end"`
}

type PortRange struct {
	Start types.Int64 `json:"start" tfsdk:"start"`
	End   types.Int64 `json:"end" tfsdk:"end"`
}

// Then convert using proper struct
// func convertToStruct(obj tftypes.Value) (*NetworkConfig, error) {
// 	var config NetworkConfig
// 	err := tftypes.ValueAs(obj, &config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &config, nil
// }

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
			isNull := mCallable()
			m = tfField.MethodByName("IsUnknown")
			mCallable = m.Interface().(func() bool)
			isUnknown := mCallable()
			tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI FilterRules IsNull=%v IsUnknown=%v", isNull, isUnknown))
			if isNull || isUnknown {
				tflog.Debug(ctx, "🍻 copyTFtoAPI skipping FilterRules because it is null or unknown")
				continue
			}

			listVal := tfField.Interface().(types.List)
			tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI FilterRules raw list type: %+v", listVal))

			var config []NetworkConfig

			diag := listVal.ElementsAs(ctx, &config, true)
			if diag.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("💥 copyTFtoAPI FilterRules ValueAs error: %v", diag))
				continue
			}

			tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI using %+v", config))
			outList := make([]map[string]interface{}, 0, len(config))
			for _, c := range config {
				m := make(map[string]interface{})

				// ip_addresses: check CIDR, list, range (all are terraform types)
				if !(c.IPAddresses.CIDR.IsNull() || c.IPAddresses.CIDR.IsUnknown()) && c.IPAddresses.CIDR.ValueString() != "" {
					m["ip_addresses"] = map[string]interface{}{"cidr": c.IPAddresses.CIDR.ValueString()}
				} else if !(c.IPAddresses.List.IsNull() || c.IPAddresses.List.IsUnknown()) {
					var ips []string
					if err := c.IPAddresses.List.ElementsAs(ctx, &ips, false); err == nil {
						arr := make([]interface{}, 0, len(ips))
						for _, s := range ips {
							arr = append(arr, s)
						}
						m["ip_addresses"] = map[string]interface{}{"list": arr}
					}
				} else if !(c.IPAddresses.Range.IsNull() || c.IPAddresses.Range.IsUnknown()) {
					var rng struct {
						Start types.String `tfsdk:"start"`
						End   types.String `tfsdk:"end"`
					}
					_ = c.IPAddresses.Range.As(ctx, &rng, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
					if !(rng.Start.IsNull() || rng.Start.IsUnknown()) && !(rng.End.IsNull() || rng.End.IsUnknown()) {
						m["ip_addresses"] = map[string]interface{}{"range": map[string]interface{}{"start": rng.Start.ValueString(), "end": rng.End.ValueString()}}
					}
				}

				// ports
				if !(c.Ports.IsNull() || c.Ports.IsUnknown()) {
					var ports struct {
						List  types.List   `tfsdk:"list"`
						Range types.Object `tfsdk:"range"`
					}
					_ = c.Ports.As(ctx, &ports, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
					if !(ports.List.IsNull() || ports.List.IsUnknown()) {
						var pvals []int64
						if err := ports.List.ElementsAs(ctx, &pvals, false); err == nil {
							parr := make([]interface{}, 0, len(pvals))
							for _, p := range pvals {
								parr = append(parr, p)
							}
							m["ports"] = map[string]interface{}{"list": parr}
						}
					} else if !(ports.Range.IsNull() || ports.Range.IsUnknown()) {
						var pr struct {
							Start types.Int64 `tfsdk:"start"`
							End   types.Int64 `tfsdk:"end"`
						}
						_ = ports.Range.As(ctx, &pr, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
						if !(pr.Start.IsNull() || pr.Start.IsUnknown()) && !(pr.End.IsNull() || pr.End.IsUnknown()) {
							m["ports"] = map[string]interface{}{"range": map[string]interface{}{"start": pr.Start.ValueInt64(), "end": pr.End.ValueInt64()}}
						}
					}
				}

				// protocol
				if !(c.Protocol.IsNull() || c.Protocol.IsUnknown()) && c.Protocol.ValueString() != "" {
					m["protocol"] = strings.ToUpper(c.Protocol.ValueString())
				} else {
					m["protocol"] = "ANY"
				}

				outList = append(outList, m)
			}

			newBytes, err := json.Marshal(outList)
			if err != nil {
				tflog.Debug(ctx, fmt.Sprintf("💥 copyTFtoAPI marshalled filter_rules JSON error: %v", err))
				continue
			}
			tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI marshalled filter_rules JSON len=%d, [%s]", len(newBytes), newBytes))
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

			// If destination is pointer to string and TF value is empty string, skip setting so field is omitted (omitempty) instead of sending "" (API may reject empty string as invalid).
			if apiTypeField.Type.Elem().Kind() == reflect.String && tfObjField.Type.String() == "types.String" {
				val := tfField.Interface().(types.String)
				if val.ValueString() == "" {
					tflog.Debug(ctx, fmt.Sprintf("🍻 copyTFtoAPI skipping empty string pointer field %s", fieldName))
					continue
				}
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
			field.SetString(val.ValueString())
		case reflect.Int:
			val := tfField.Interface().(types.Int64)
			field.SetInt(val.ValueInt64())
		case reflect.Bool:
			val := tfField.Interface().(types.Bool)
			field.SetBool(val.ValueBool())

		case reflect.Slice:
			// Special-case for json.RawMessage on the API model (backed by []byte)
			// if tfObjField.Name == "FilterRules" {
			// 	val := tfField.Interface().(types.String)
			// 	// If null/unknown/empty, leave API nil/empty
			// 	if val.IsNull() || val.IsUnknown() || val.ValueString() == "" {
			// 		// leave as nil/empty
			// 	} else {
			// 		var list []map[string]interface{}
			// 		if err := json.Unmarshal([]byte(val.ValueString()), &list); err == nil {
			// 			var outList []map[string]interface{}
			// 			for _, item := range list {
			// 				if v, ok := item["ip_addresses"]; ok {
			// 					switch vv := v.(type) {
			// 					case []interface{}:
			// 						var cidrs []string
			// 						var plain []interface{}
			// 						for _, elem := range vv {
			// 							if s, ok := elem.(string); ok && strings.Contains(s, "/") {
			// 								cidrs = append(cidrs, s)
			// 							} else {
			// 								plain = append(plain, elem)
			// 							}
			// 						}
			// 						if len(cidrs) > 0 && len(plain) == 0 {
			// 							for _, c := range cidrs {
			// 								ni := make(map[string]interface{}, len(item))
			// 								for k, v := range item {
			// 									if k == "ip_addresses" {
			// 										continue
			// 									}
			// 									ni[k] = v
			// 								}
			// 								ni["ip_addresses"] = map[string]interface{}{"cidr": c}
			// 								if p, ok := ni["protocol"]; ok {
			// 									if ps, ok := p.(string); ok {
			// 										ni["protocol"] = strings.ToUpper(ps)
			// 									}
			// 								} else {
			// 									ni["protocol"] = "ANY"
			// 								}
			// 								if pp, ok := ni["ports"]; ok {
			// 									switch pvv := pp.(type) {
			// 									case []interface{}:
			// 										ni["ports"] = map[string]interface{}{"list": pvv}
			// 									case map[string]interface{}:
			// 									default:
			// 										ni["ports"] = map[string]interface{}{"list": []interface{}{}}
			// 									}
			// 								} else {
			// 									ni["ports"] = map[string]interface{}{"list": []interface{}{}}
			// 								}
			// 								outList = append(outList, ni)
			// 							}
			// 							continue
			// 						}
			// 						if len(cidrs) > 0 && len(plain) > 0 {
			// 							ni := make(map[string]interface{}, len(item))
			// 							for k, v := range item {
			// 								if k == "ip_addresses" {
			// 									continue
			// 								}
			// 								ni[k] = v
			// 							}
			// 							ni["ip_addresses"] = map[string]interface{}{"list": plain}
			// 							if p, ok := ni["protocol"]; ok {
			// 								if ps, ok := p.(string); ok {
			// 									ni["protocol"] = strings.ToUpper(ps)
			// 								}
			// 							} else {
			// 								ni["protocol"] = "ANY"
			// 							}
			// 							if pp, ok := ni["ports"]; ok {
			// 								switch pvv := pp.(type) {
			// 								case []interface{}:
			// 									ni["ports"] = map[string]interface{}{"list": pvv}
			// 								case map[string]interface{}:
			// 								default:
			// 									ni["ports"] = map[string]interface{}{"list": []interface{}{}}
			// 								}
			// 							} else {
			// 								ni["ports"] = map[string]interface{}{"list": []interface{}{}}
			// 							}
			// 							outList = append(outList, ni)
			// 							for _, c := range cidrs {
			// 								ci := make(map[string]interface{}, len(item))
			// 								for k, v := range item {
			// 									if k == "ip_addresses" {
			// 										continue
			// 									}
			// 									ci[k] = v
			// 								}
			// 								ci["ip_addresses"] = map[string]interface{}{"cidr": c}
			// 								if p, ok := ci["protocol"]; ok {
			// 									if ps, ok := p.(string); ok {
			// 										ci["protocol"] = strings.ToUpper(ps)
			// 									}
			// 								} else {
			// 									ci["protocol"] = "ANY"
			// 								}
			// 								if pp, ok := ci["ports"]; ok {
			// 									switch pvv := pp.(type) {
			// 									case []interface{}:
			// 										ci["ports"] = map[string]interface{}{"list": pvv}
			// 									case map[string]interface{}:
			// 									default:
			// 										ci["ports"] = map[string]interface{}{"list": []interface{}{}}
			// 									}
			// 								} else {
			// 									ci["ports"] = map[string]interface{}{"list": []interface{}{}}
			// 								}
			// 								outList = append(outList, ci)
			// 							}
			// 							continue
			// 						}
			// 						item["ip_addresses"] = map[string]interface{}{"list": vv}
			// 					case string:
			// 						if strings.Contains(vv, "/") {
			// 							item["ip_addresses"] = map[string]interface{}{"cidr": vv}
			// 						} else {
			// 							item["ip_addresses"] = map[string]interface{}{"list": []interface{}{vv}}
			// 						}
			// 					case map[string]interface{}:
			// 					}
			// 				}
			// 				if v, ok := item["ports"]; ok {
			// 					switch vv := v.(type) {
			// 					case []interface{}:
			// 						item["ports"] = map[string]interface{}{"list": vv}
			// 					case map[string]interface{}:
			// 					}
			// 				}
			// 				if v, ok := item["protocol"]; ok {
			// 					if s, ok := v.(string); ok {
			// 						item["protocol"] = strings.ToUpper(s)
			// 					}
			// 				}
			// 				outList = append(outList, item)
			// 			}
			// 			newBytes, _ := json.Marshal(outList)
			// 			field.Set(reflect.ValueOf([]byte(newBytes)))
			// 		} else {
			// 			field.Set(reflect.ValueOf([]byte("null")))
			// 		}
			// 	}
			// }

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
		case reflect.Struct:
			if tfObjField.Name == "KeyInfo" {
				attrTypes := map[string]attr.Type{
					"encoded_info":   types.StringType,
					"filename":       types.StringType,
					"installer_path": types.StringType,
				}
				if setToNil {
					*(*types.Object)(tfObj.Field(i).Addr().UnsafePointer()) = types.ObjectNull(attrTypes)
					break
				}
				vals := map[string]attr.Value{
					"encoded_info":   types.StringNull(),
					"filename":       types.StringNull(),
					"installer_path": types.StringNull(),
				}
				if f := field.FieldByName("EncodedInfo"); f.IsValid() && f.Kind() == reflect.String && f.Len() > 0 {
					vals["encoded_info"] = types.StringValue(f.String())
				}
				if f := field.FieldByName("Filename"); f.IsValid() && f.Kind() == reflect.String && f.Len() > 0 {
					vals["filename"] = types.StringValue(f.String())
				}
				if f := field.FieldByName("InstallerPath"); f.IsValid() && f.Kind() == reflect.String && f.Len() > 0 {
					vals["installer_path"] = types.StringValue(f.String())
				}
				ov, diag := types.ObjectValue(attrTypes, vals)
				if diag.HasError() {
					*(*types.Object)(tfObj.Field(i).Addr().UnsafePointer()) = types.ObjectNull(attrTypes)
				} else {
					*(*types.Object)(tfObj.Field(i).Addr().UnsafePointer()) = ov
				}
				break
			}
			// Unhandled struct types will be ignored here.
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
						// Unmarshal into primitive maps so json.Unmarshal decodes strings/numbers
						// rather than trying to decode into Terraform framework types.
						var apiConfig []map[string]interface{}
						if err := json.Unmarshal(rawBytes, &apiConfig); err != nil {
							tflog.Warn(ctx, fmt.Sprintf("Failed to unmarshal FilterRules JSON into primitive maps: %v", err))
							// Fallback: keep raw JSON as string in TF state so user can inspect it
							*(*types.String)(tfObj.Field(i).Addr().UnsafePointer()) = types.StringValue(string(rawBytes))
						} else {
							// Create the TF list value and set it
							elemType := types.ObjectType{
								AttrTypes: map[string]attr.Type{
									"ip_addresses": types.ObjectType{
										AttrTypes: map[string]attr.Type{
											"list": types.ListType{ElemType: types.StringType},
											"cidr": types.StringType,
											"range": types.ObjectType{
												AttrTypes: map[string]attr.Type{
													"start": types.StringType,
													"end":   types.StringType,
												},
											},
										},
									},
									"ports": types.ObjectType{
										AttrTypes: map[string]attr.Type{
											"list": types.ListType{ElemType: types.Int64Type},
											"range": types.ObjectType{
												AttrTypes: map[string]attr.Type{
													"start": types.Int64Type,
													"end":   types.Int64Type,
												},
											},
										},
									},
									"protocol": types.StringType,
								},
							}
							tflog.Debug(ctx, fmt.Sprintf("🎯 copyAPItoTF creating FilterRules list with primitive value %+v", apiConfig))
							elems := make([]attr.Value, 0, len(apiConfig))
							for idx, itm := range apiConfig {
								// Build attr.Values explicitly to avoid framework trying to coerce map[string]interface{} -> ObjectType
								// ip_addresses
								ipObjType := elemType.AttrTypes["ip_addresses"].(types.ObjectType)
								ipAttrTypes := ipObjType.AttrTypes
								ipVals := make(map[string]attr.Value, len(ipAttrTypes))

								// defaults null
								ipVals["cidr"] = types.StringNull()
								ipVals["list"] = types.ListNull(types.StringType)
								rangeType := ipAttrTypes["range"].(types.ObjectType)
								ipVals["range"] = types.ObjectNull(rangeType.AttrTypes)

								if rawIP, ok := itm["ip_addresses"]; ok {
									if ipMap, ok := rawIP.(map[string]interface{}); ok {
										if cidrRaw, ok := ipMap["cidr"].(string); ok && cidrRaw != "" {
											ipVals["cidr"] = types.StringValue(cidrRaw)
										}
										if listRaw, ok := ipMap["list"].([]interface{}); ok {
											strs := make([]string, 0, len(listRaw))
											for _, v := range listRaw {
												if s, ok := v.(string); ok {
													strs = append(strs, s)
												}
											}
											if len(strs) > 0 {
												if lv, ld := types.ListValueFrom(ctx, types.StringType, strs); !ld.HasError() {
													ipVals["list"] = lv
												}
											}
										}
										if rangeRaw, ok := ipMap["range"].(map[string]interface{}); ok {
											startStr, _ := rangeRaw["start"].(string)
											endStr, _ := rangeRaw["end"].(string)
											if startStr != "" && endStr != "" {
												rVals := map[string]attr.Value{"start": types.StringValue(startStr), "end": types.StringValue(endStr)}
												if rObj, rDiag := types.ObjectValue(rangeType.AttrTypes, rVals); !rDiag.HasError() {
													ipVals["range"] = rObj
												}
											}
										}
									}
								}
								ipObj, ipDiag := types.ObjectValue(ipAttrTypes, ipVals)
								if ipDiag.HasError() {
									tflog.Warn(ctx, fmt.Sprintf("Failed to create ip_addresses object for FilterRules[%d]: %v", idx, ipDiag))
									continue
								}

								// ports (optional)
								portsObjType := elemType.AttrTypes["ports"].(types.ObjectType)
								portsAttrTypes := portsObjType.AttrTypes
								portsVals := make(map[string]attr.Value, len(portsAttrTypes))
								portsVals["list"] = types.ListNull(types.Int64Type)
								portRangeType := portsAttrTypes["range"].(types.ObjectType)
								portsVals["range"] = types.ObjectNull(portRangeType.AttrTypes)
								portsPresent := false
								if rawPorts, ok := itm["ports"]; ok {
									if pMap, ok := rawPorts.(map[string]interface{}); ok {
										if listRaw, ok := pMap["list"].([]interface{}); ok {
											ints := make([]int64, 0, len(listRaw))
											for _, v := range listRaw {
												switch x := v.(type) {
												case float64:
													ints = append(ints, int64(x))
												case int:
													ints = append(ints, int64(x))
												case int64:
													ints = append(ints, x)
												}
											}
											if len(ints) > 0 {
												if lv, ld := types.ListValueFrom(ctx, types.Int64Type, ints); !ld.HasError() {
													portsVals["list"] = lv
													portsPresent = true
												}
											}
										}
										if rangeRaw, ok := pMap["range"].(map[string]interface{}); ok {
											var startI, endI *int64
											if v, ok := rangeRaw["start"]; ok {
												switch x := v.(type) {
												case float64:
													v2 := int64(x)
													startI = &v2
												case int:
													v2 := int64(x)
													startI = &v2
												case int64:
													v2 := x
													startI = &v2
												}
											}
											if v, ok := rangeRaw["end"]; ok {
												switch x := v.(type) {
												case float64:
													v2 := int64(x)
													endI = &v2
												case int:
													v2 := int64(x)
													endI = &v2
												case int64:
													v2 := x
													endI = &v2
												}
											}
											if startI != nil && endI != nil {
												rVals := map[string]attr.Value{"start": types.Int64Value(*startI), "end": types.Int64Value(*endI)}
												if rObj, rDiag := types.ObjectValue(portRangeType.AttrTypes, rVals); !rDiag.HasError() {
													portsVals["range"] = rObj
													portsPresent = true
												}
											}
										}
									}
								}
								var portsObj attr.Value
								if portsPresent {
									if pObj, pDiag := types.ObjectValue(portsAttrTypes, portsVals); !pDiag.HasError() {
										portsObj = pObj
									} else {
										portsObj = types.ObjectNull(portsAttrTypes)
									}
								} else {
									portsObj = types.ObjectNull(portsAttrTypes)
								}

								// protocol
								prot := "ANY"
								if pRaw, ok := itm["protocol"].(string); ok && pRaw != "" {
									prot = strings.ToUpper(pRaw)
								}
								protocolVal := types.StringValue(prot)

								valMap := map[string]attr.Value{"ip_addresses": ipObj, "ports": portsObj, "protocol": protocolVal}
								objVal, objDiag := types.ObjectValue(elemType.AttributeTypes(), valMap)
								if objDiag.HasError() {
									tflog.Warn(ctx, fmt.Sprintf("Failed to create ObjectValue for FilterRules[%d]: %v", idx, objDiag))
									continue
								}
								elems = append(elems, objVal)
							}
							listVal, listDiags := types.ListValueFrom(ctx, elemType, elems)
							if listDiags.HasError() {
								tflog.Warn(ctx, fmt.Sprintf("Failed to create FilterRules list value: %v", listDiags))
							}
							*(*types.List)(tfObj.Field(i).Addr().UnsafePointer()) = listVal
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
