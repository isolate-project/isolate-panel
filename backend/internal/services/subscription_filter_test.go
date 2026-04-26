package services

import (
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
)

func TestFilterInbounds_EmptyFilter(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
	}

	filter := &SubscriptionFilter{}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 3 {
		t.Errorf("Expected 3 inbounds, got %d", len(result))
	}
}

func TestFilterInbounds_NilFilter(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
	}

	var filter *SubscriptionFilter
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}
}

func TestFilterInbounds_ProtocolRegex(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
		{Protocol: "shadowsocks", Name: "EU-02", Core: &models.Core{Name: "xray"}},
	}

	filter := &SubscriptionFilter{ProtocolRegex: "vless|vmess"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}

	if result[0].Protocol != "vless" {
		t.Errorf("Expected first inbound to be vless, got %s", result[0].Protocol)
	}

	if result[1].Protocol != "vmess" {
		t.Errorf("Expected second inbound to be vmess, got %s", result[1].Protocol)
	}
}

func TestFilterInbounds_CoreNameExactMatch(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
		{Protocol: "vless", Name: "EU-02", Core: &models.Core{Name: "xray"}},
	}

	filter := &SubscriptionFilter{CoreName: "xray"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}

	for _, inbound := range result {
		if inbound.Core == nil || inbound.Core.Name != "xray" {
			t.Errorf("Expected all inbounds to have xray core, got %s", inbound.Core.Name)
		}
	}
}

func TestFilterInbounds_CoreNameRegex(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
		{Protocol: "vless", Name: "EU-02", Core: &models.Core{Name: "xray"}},
	}

	filter := &SubscriptionFilter{CoreNameRegex: "xray|singbox"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 3 {
		t.Errorf("Expected 3 inbounds, got %d", len(result))
	}

	for _, inbound := range result {
		if inbound.Core == nil {
			t.Errorf("Expected all inbounds to have core, got nil")
		}
		if inbound.Core.Name != "xray" && inbound.Core.Name != "singbox" {
			t.Errorf("Expected core name to be xray or singbox, got %s", inbound.Core.Name)
		}
	}
}

func TestFilterInbounds_TagRegex(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
		{Protocol: "vless", Name: "EU-02", Core: &models.Core{Name: "xray"}},
	}

	filter := &SubscriptionFilter{TagRegex: "US.*"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}

	for _, inbound := range result {
		if inbound.Name != "US-01" && inbound.Name != "US-02" {
			t.Errorf("Expected all inbounds to have US prefix, got %s", inbound.Name)
		}
	}
}

func TestFilterInbounds_CombinedFilters(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
		{Protocol: "vless", Name: "EU-02", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-03", Core: &models.Core{Name: "xray"}},
	}

	filter := &SubscriptionFilter{
		ProtocolRegex: "vless|vmess",
		CoreName:      "xray",
		TagRegex:      "US.*",
	}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}

	for _, inbound := range result {
		if inbound.Protocol != "vless" && inbound.Protocol != "vmess" {
			t.Errorf("Expected protocol to be vless or vmess, got %s", inbound.Protocol)
		}
		if inbound.Core == nil || inbound.Core.Name != "xray" {
			t.Errorf("Expected core to be xray, got %v", inbound.Core)
		}
		if inbound.Name != "US-01" && inbound.Name != "US-03" {
			t.Errorf("Expected name to be US-01 or US-03, got %s", inbound.Name)
		}
	}
}

func TestFilterInbounds_NilCoreWithCoreNameFilter(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: nil},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
	}

	filter := &SubscriptionFilter{CoreName: "xray"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 1 {
		t.Errorf("Expected 1 inbound, got %d", len(result))
	}

	if result[0].Core == nil || result[0].Core.Name != "xray" {
		t.Errorf("Expected inbound to have xray core, got %v", result[0].Core)
	}
}

func TestFilterInbounds_NilCoreWithCoreNameRegexFilter(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: nil},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
	}

	filter := &SubscriptionFilter{CoreNameRegex: "xray|mihomo"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}

	for _, inbound := range result {
		if inbound.Core == nil {
			t.Errorf("Expected all inbounds to have core, got nil")
		}
	}
}

func TestFilterInbounds_InvalidRegex(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
	}

	filter := &SubscriptionFilter{ProtocolRegex: "[invalid"}
	result := filter.FilterInbounds(inbounds)

	// Invalid regex should be skipped, so all inbounds should be returned
	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds (invalid regex should be skipped), got %d", len(result))
	}
}

func TestFilterInbounds_NoMatches(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
		{Protocol: "trojan", Name: "EU-01", Core: &models.Core{Name: "mihomo"}},
	}

	filter := &SubscriptionFilter{ProtocolRegex: "shadowsocks"}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 0 {
		t.Errorf("Expected 0 inbounds, got %d", len(result))
	}
}

func TestFilterInbounds_AllFiltersEmpty(t *testing.T) {
	inbounds := []models.Inbound{
		{Protocol: "vless", Name: "US-01", Core: &models.Core{Name: "xray"}},
		{Protocol: "vmess", Name: "US-02", Core: &models.Core{Name: "singbox"}},
	}

	filter := &SubscriptionFilter{
		ProtocolRegex: "",
		CoreName:      "",
		CoreNameRegex: "",
		TagRegex:      "",
	}
	result := filter.FilterInbounds(inbounds)

	if len(result) != 2 {
		t.Errorf("Expected 2 inbounds, got %d", len(result))
	}
}