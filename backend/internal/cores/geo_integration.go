package cores

import (
	"fmt"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// GeoRulesData holds geo routing rules for a core
type GeoRulesData struct {
	Rules []models.GeoRule
}

// LoadGeoRules loads enabled GeoIP/GeoSite rules for a given core.
// Returns nil if no rules are configured (graceful skip).
func LoadGeoRules(ctx *ConfigContext, coreID uint) (*GeoRulesData, error) {
	var rules []models.GeoRule
	if err := ctx.DB.Where("core_id = ? AND is_enabled = ?", coreID, true).
		Order("priority DESC").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to load geo rules: %w", err)
	}

	if len(rules) == 0 {
		return nil, nil
	}

	return &GeoRulesData{Rules: rules}, nil
}

// mapGeoAction maps a GeoRule action to an outbound tag
func mapGeoAction(action string) string {
	switch action {
	case "direct":
		return "direct"
	case "block":
		return "block"
	case "warp":
		return warpTag
	case "proxy":
		return "proxy" // default proxy outbound
	default:
		return "direct"
	}
}

// ============================================================
// Sing-box Geo helpers
// ============================================================

// SingboxGeoRouteRules converts GeoRules to Sing-box route rules (v1.12+ format)
func SingboxGeoRouteRules(rules []models.GeoRule, geoDir string) []map[string]interface{} {
	routeRules := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		rr := map[string]interface{}{
			"action":   "route",
			"outbound": mapGeoAction(rule.Action),
		}
		switch rule.Type {
		case "geoip":
			rr["rule_set"] = []string{"geoip-" + rule.Code}
		case "geosite":
			rr["rule_set"] = []string{"geosite-" + rule.Code}
		}
		routeRules = append(routeRules, rr)
	}
	return routeRules
}

// SingboxGeoRuleSets returns rule-set definitions for Sing-box route config (v1.12+ format)
// Uses remote rule-sets from GitHub since we can't parse .db files at runtime
func SingboxGeoRuleSets(rules []models.GeoRule) []map[string]interface{} {
	seen := make(map[string]bool)
	ruleSets := make([]map[string]interface{}, 0)

	for _, rule := range rules {
		var tag string
		var url string
		switch rule.Type {
		case "geoip":
			tag = "geoip-" + rule.Code
			url = fmt.Sprintf("https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/%s.json", tag)
		case "geosite":
			tag = "geosite-" + rule.Code
			url = fmt.Sprintf("https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/%s.json", tag)
		default:
			continue
		}

		if !seen[tag] {
			seen[tag] = true
			ruleSets = append(ruleSets, map[string]interface{}{
				"tag":             tag,
				"type":            "remote",
				"url":             url,
				"download_detour": "direct",
				"update_interval": "7d",
			})
		}
	}

	return ruleSets
}

// ============================================================
// Xray Geo helpers
// ============================================================

// XrayGeoRoutingRules converts GeoRules to Xray routing rules
func XrayGeoRoutingRules(rules []models.GeoRule) []map[string]interface{} {
	routingRules := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		rr := map[string]interface{}{
			"type":        "field",
			"outboundTag": mapGeoAction(rule.Action),
		}
		switch rule.Type {
		case "geoip":
			rr["ip"] = []string{fmt.Sprintf("geoip:%s", rule.Code)}
		case "geosite":
			rr["domain"] = []string{fmt.Sprintf("geosite:%s", rule.Code)}
		}
		routingRules = append(routingRules, rr)
	}
	return routingRules
}

// ============================================================
// Mihomo Geo helpers
// ============================================================

// MihomoGeoRules converts GeoRules to Mihomo rule strings
func MihomoGeoRules(rules []models.GeoRule) []string {
	mihomoRules := make([]string, 0, len(rules))
	for _, rule := range rules {
		outbound := mapGeoAction(rule.Action)
		// Mihomo uses uppercase outbound tag for built-in: DIRECT, REJECT
		outboundTag := outbound
		switch outbound {
		case "direct":
			outboundTag = "DIRECT"
		case "block":
			outboundTag = "REJECT"
		}

		switch rule.Type {
		case "geoip":
			mihomoRules = append(mihomoRules, fmt.Sprintf("GEOIP,%s,%s", rule.Code, outboundTag))
		case "geosite":
			mihomoRules = append(mihomoRules, fmt.Sprintf("GEOSITE,%s,%s", rule.Code, outboundTag))
		}
	}
	return mihomoRules
}

// InjectGeo loads geo rules and indicates if injection is needed
func InjectGeo(ctx *ConfigContext, coreID uint) (*GeoRulesData, bool) {
	data, err := LoadGeoRules(ctx, coreID)
	if err != nil || data == nil {
		return nil, false
	}
	return data, true
}
