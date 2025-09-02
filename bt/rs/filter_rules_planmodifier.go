package rs

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// filterRulesPlanModifier normalizes the JSON string value for filter_rules in the plan.
// It implements planmodifier.String.
type filterRulesPlanModifier struct{}

func (m filterRulesPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the config value is null/unknown, leave it alone.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	// Extract the raw string value from config (user input) if present, otherwise fall back to plan.
	var in types.String
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		in = req.ConfigValue
	} else if !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() {
		in = req.PlanValue
	} else {
		return
	}

	s := in.ValueString()
	if s == "" {
		return
	}

	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("filter_rules invalid JSON", "filter_rules must be a JSON array of objects"))
		return
	}

	var outList []map[string]interface{}
	// helper to normalize protocol and ports for a rule item
	normalize := func(m map[string]interface{}) map[string]interface{} {
		// protocol: keep as-provided in plan to avoid plan vs config diffs; API-side will
		// uppercase when sending the request. Default to "ANY" only if not present.
		if _, ok := m["protocol"]; !ok {
			m["protocol"] = "ANY"
		}
		// ports
		if p, ok := m["ports"]; ok {
			switch pv := p.(type) {
			case []interface{}:
				m["ports"] = map[string]interface{}{"list": pv}
			case map[string]interface{}:
				// keep as-is
			default:
				m["ports"] = map[string]interface{}{"list": []interface{}{}}
			}
		} else {
			m["ports"] = map[string]interface{}{"list": []interface{}{}}
		}
		return m
	}

	for _, item := range list {
		if v, ok := item["ip_addresses"]; ok {
			switch vv := v.(type) {
			case []interface{}:
				// user provided bare array -> preserve as nested object with list
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
		outList = append(outList, normalize(item))
	}

	nb, _ := json.Marshal(outList)
	resp.PlanValue = types.StringValue(string(nb))
}

func (filterRulesPlanModifier) Description(ctx context.Context) string {
	return "Normalizes filter_rules JSON into canonical API shape"
}

func (filterRulesPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Normalizes filter_rules JSON into canonical API shape"
}
