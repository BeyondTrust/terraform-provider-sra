package rs

import (
	"context"
	"terraform-provider-sra/api"
	"terraform-provider-sra/bt/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func applyNetworkTunnelValidate(ctx context.Context, plan *models.NetworkTunnelJump) diag.Diagnostics {
	var diags diag.Diagnostics
	// filter_rules is now a structured list attribute in the TF model.
	if plan.FilterRules.IsNull() || plan.FilterRules.IsUnknown() {
		diags.Append(diag.NewErrorDiagnostic("filter_rules is required", "NetworkTunnelJump requires filter_rules with at least one rule"))
		return diags
	}

	// Try decoding into TF runtime objects then convert each to a generic map.
	var rules []map[string]interface{}
	var objValues []types.Object
	if err := plan.FilterRules.ElementsAs(ctx, &objValues, false); err == nil {
		// basic checks
		if len(objValues) == 0 {
			diags.Append(diag.NewErrorDiagnostic("filter_rules is required", "NetworkTunnelJump requires at least one filter rule"))
			return diags
		}
		if len(objValues) > 50 {
			diags.Append(diag.NewErrorDiagnostic("filter_rules too large", "NetworkTunnelJump filter_rules must contain at most 50 rules"))
			return diags
		}

		for _, obj := range objValues {
			// 1) Try decoding into a struct that includes Ports
			var withPorts struct {
				IpAddresses types.List   `tfsdk:"ip_addresses"`
				Ports       types.Object `tfsdk:"ports"`
			}
			if d := obj.As(ctx, &withPorts, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); !d.HasError() {
				if withPorts.IpAddresses.IsNull() || withPorts.IpAddresses.IsUnknown() {
					diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include ip_addresses"))
					return diags
				}
				m := make(map[string]interface{})
				m["ip_addresses"] = withPorts.IpAddresses
				if !withPorts.Ports.IsNull() && !withPorts.Ports.IsUnknown() {
					// Try decoding as list-only
					var listOnly struct {
						List types.List `tfsdk:"list"`
					}
					if ld := withPorts.Ports.As(ctx, &listOnly, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); !ld.HasError() {
						if !(listOnly.List.IsNull() || listOnly.List.IsUnknown()) {
							var portVals []int64
							if err := listOnly.List.ElementsAs(ctx, &portVals, false); err != nil || len(portVals) == 0 {
								diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports.list must be a non-empty list of port numbers"))
								return diags
							}
							for _, p := range portVals {
								if p < 1 || p > 65535 {
									diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "port values must be between 1 and 65535"))
									return diags
								}
							}
						}
						m["ports"] = withPorts.Ports
					} else {
						// Try decoding as range-only
						var rangeOnly struct {
							Range struct {
								Start types.Int64 `tfsdk:"start"`
								End   types.Int64 `tfsdk:"end"`
							} `tfsdk:"range"`
						}
						if rd := withPorts.Ports.As(ctx, &rangeOnly, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); !rd.HasError() {
							if !(rangeOnly.Range.Start.IsNull() || rangeOnly.Range.Start.IsUnknown() || rangeOnly.Range.End.IsNull() || rangeOnly.Range.End.IsUnknown()) {
								start := rangeOnly.Range.Start.ValueInt64()
								end := rangeOnly.Range.End.ValueInt64()
								if start < 1 || end < 1 || start > 65535 || end > 65535 {
									diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "range start/end must be between 1 and 65535"))
									return diags
								}
								if start > end {
									diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "range start must be <= range end"))
									return diags
								}
							}
							m["ports"] = withPorts.Ports
						} else {
							diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "unable to parse ports object"))
							return diags
						}
					}
				}
				rules = append(rules, m)
				continue
			}

			// 2) Try decoding into a struct with only ip_addresses (works when ports absent)
			var noPorts struct {
				IpAddresses types.List `tfsdk:"ip_addresses"`
			}
			if d := obj.As(ctx, &noPorts, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); !d.HasError() {
				if noPorts.IpAddresses.IsNull() || noPorts.IpAddresses.IsUnknown() {
					diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include ip_addresses"))
					return diags
				}
				m := make(map[string]interface{})
				m["ip_addresses"] = noPorts.IpAddresses
				rules = append(rules, m)
				continue
			}

			// 3) Fallback: decode into map[string]attr.Value and preserve attr.Values
			var mdAttr map[string]attr.Value
			if d := obj.As(ctx, &mdAttr, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "unable to parse filter_rules object"))
				return diags
			}
			m := make(map[string]interface{})
			if v, ok := mdAttr["ip_addresses"]; ok {
				m["ip_addresses"] = v
			}
			if v, ok := mdAttr["ports"]; ok {
				// try to validate ports when present in the fallback path
				if pvObj, ok := v.(types.Object); ok {
					var portsStruct struct {
						List  types.List `tfsdk:"list"`
						Range struct {
							Start types.Int64 `tfsdk:"start"`
							End   types.Int64 `tfsdk:"end"`
						} `tfsdk:"range"`
					}
					if pd := pvObj.As(ctx, &portsStruct, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); pd.HasError() {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "unable to parse ports object"))
						return diags
					}
					// reuse same checks as above
					listPresent := !(portsStruct.List.IsNull() || portsStruct.List.IsUnknown())
					rangePresent := !(portsStruct.Range.Start.IsNull() || portsStruct.Range.Start.IsUnknown() || portsStruct.Range.End.IsNull() || portsStruct.Range.End.IsUnknown())
					if listPresent && rangePresent {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports must contain either 'list' or 'range', not both"))
						return diags
					}
				}
				m["ports"] = v
			}
			rules = append(rules, m)
		}
	}
	for _, r := range rules {
		v, ok := r["ip_addresses"]
		if !ok {
			diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include ip_addresses"))
			return diags
		}
		switch vv := v.(type) {
		case []interface{}:
			if len(vv) == 0 {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include at least one ip_address"))
				return diags
			}
		case string:
			if vv == "" {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include at least one ip_address"))
				return diags
			}
		case []string:
			if len(vv) == 0 {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include at least one ip_address"))
				return diags
			}
		case []attr.Value:
			if len(vv) == 0 {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include at least one ip_address"))
				return diags
			}
		case attr.Value:
			if lv, ok := vv.(types.List); ok {
				var ips []string
				if err := lv.ElementsAs(ctx, &ips, false); err != nil || len(ips) == 0 {
					diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "each filter_rules entry must include at least one ip_address"))
					return diags
				}
			} else {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ip_addresses must be a list of strings"))
				return diags
			}
		default:
			diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ip_addresses must be a list of strings"))
			return diags
		}
		// validate ports if present
		if pv, ok := r["ports"]; ok {
			// pv may be an attr.Value (types.Object) or types.Object
			var objVal types.Object
			switch t := pv.(type) {
			case types.Object:
				objVal = t
			case attr.Value:
				if vobj, ok := t.(types.Object); ok {
					objVal = vobj
				} else {
					diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports must be an object"))
					return diags
				}
			default:
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports must be an object"))
				return diags
			}

			// decode ports object into a map of attr.Values
			var portsMap map[string]attr.Value
			if d := objVal.As(ctx, &portsMap, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "unable to parse ports object"))
				return diags
			}

			_, hasList := portsMap["list"]
			_, hasRange := portsMap["range"]
			if hasList && hasRange {
				diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports must contain either 'list' or 'range', not both"))
				return diags
			}

			if hasList {
				if lv, ok := portsMap["list"].(types.List); ok {
					var portVals []int64
					if err := lv.ElementsAs(ctx, &portVals, false); err != nil || len(portVals) == 0 {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports.list must be a non-empty list of port numbers"))
						return diags
					}
					for _, p := range portVals {
						if p < 1 || p > 65535 {
							diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "port values must be between 1 and 65535"))
							return diags
						}
					}
				} else {
					diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports.list must be a list of integers"))
					return diags
				}
			}

			if hasRange {
				if rv, ok := portsMap["range"].(types.Object); ok {
					var rangeMap map[string]attr.Value
					if d := rv.As(ctx, &rangeMap, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}); d.HasError() {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "unable to parse ports.range object"))
						return diags
					}
					startV, sok := rangeMap["start"]
					endV, eok := rangeMap["end"]
					if !sok || !eok {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports.range must include start and end"))
						return diags
					}
					sv, sOk := startV.(types.Int64)
					ev, eOk := endV.(types.Int64)
					if !sOk || !eOk {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "range start/end must be integers"))
						return diags
					}
					start := sv.ValueInt64()
					end := ev.ValueInt64()
					if start < 1 || end < 1 || start > 65535 || end > 65535 {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "range start/end must be between 1 and 65535"))
						return diags
					}
					if start > end {
						diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "range start must be <= range end"))
						return diags
					}
				} else {
					diags.Append(diag.NewErrorDiagnostic("filter_rules invalid", "ports.range must be an object"))
					return diags
				}
			}
		}
	}

	return diags
}

// These throw away variable declarations are to allow the compiler to
// enforce compliance to these interfaces
var (
	_ resource.Resource                = &networkTunnelJumpResource{}
	_ resource.ResourceWithConfigure   = &networkTunnelJumpResource{}
	_ resource.ResourceWithImportState = &networkTunnelJumpResource{}
)

func newNetworkTunnelJumpResource() resource.Resource { return &networkTunnelJumpResource{} }

type networkTunnelJumpResource struct {
	apiResource[api.NetworkTunnelJump, models.NetworkTunnelJump]
}

func (r *networkTunnelJumpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Network Tunnel Jump Item. NOTE: PRA only.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":              schema.StringAttribute{Required: true},
			"hostname":          schema.StringAttribute{Required: true},
			"jumpoint_id":       schema.Int64Attribute{Required: true},
			"jump_group_id":     schema.Int64Attribute{Required: true},
			"jump_group_type":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("shared")},
			"tag":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"comments":          schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"jump_policy_id":    schema.Int64Attribute{Optional: true},
			"session_policy_id": schema.Int64Attribute{Optional: true},
			"filter_rules": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_addresses": schema.ListAttribute{ElementType: types.StringType, Required: true},
						"ports": schema.SingleNestedAttribute{
							Optional: true,
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"list": schema.ListAttribute{ElementType: types.Int64Type, Optional: true, Computed: true},
								"range": schema.SingleNestedAttribute{
									Optional: true,
									Attributes: map[string]schema.Attribute{
										"start": schema.Int64Attribute{Optional: true},
										"end":   schema.Int64Attribute{Optional: true},
									},
								},
							},
						},
						"protocol": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("ANY")},
					},
				},
			},
		},
	}
}
