package cores

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// WARPAccount represents stored Cloudflare WARP account data
type WARPAccount struct {
	AccountID   string `json:"account_id"`
	DeviceID    string `json:"device_id"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	Token       string `json:"token"`
	IPv4Address string `json:"ipv4_address"`
	IPv6Address string `json:"ipv6_address"`
	ClientID    string `json:"client_id"` // base64 reserved bytes
}

// WARPOutboundData holds everything needed to inject WARP into a core config
type WARPOutboundData struct {
	Account *WARPAccount
	Routes  []models.WarpRoute
}

const (
	warpEndpoint  = "engage.cloudflareclient.com"
	warpPort      = 2408
	warpPublicKey = "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="
	warpTag       = "warp-out"
)

// LoadWARPOutbound loads WARP account and enabled routes for a given core.
// Returns nil if WARP is not registered (graceful skip).
func LoadWARPOutbound(ctx *ConfigContext, coreID uint) (*WARPOutboundData, error) {
	if ctx.WarpDir == "" {
		return nil, nil
	}

	// Load WARP account
	account, err := loadWARPAccount(ctx.WarpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load WARP account: %w", err)
	}
	if account == nil {
		return nil, nil // Not registered — skip
	}

	// Load WARP routes from database
	var routes []models.WarpRoute
	if err := ctx.DB.Where("core_id = ? AND is_enabled = ?", coreID, true).
		Order("priority DESC").
		Find(&routes).Error; err != nil {
		return nil, fmt.Errorf("failed to load WARP routes: %w", err)
	}

	// If no routes configured, nothing to do
	if len(routes) == 0 {
		return nil, nil
	}

	return &WARPOutboundData{
		Account: account,
		Routes:  routes,
	}, nil
}

// loadWARPAccount reads the WARP account JSON from disk
func loadWARPAccount(warpDir string) (*WARPAccount, error) {
	path := filepath.Join(warpDir, "warp_account.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not registered
		}
		return nil, err
	}

	var account WARPAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	// Validate minimum required fields
	if account.PrivateKey == "" {
		return nil, fmt.Errorf("WARP account missing private key")
	}

	return &account, nil
}

// ============================================================
// Sing-box WARP helpers
// ============================================================

// SingboxWARPOutbound returns Sing-box WireGuard outbound JSON map
func SingboxWARPOutbound(account *WARPAccount) map[string]interface{} {
	localAddresses := []string{}
	if account.IPv4Address != "" {
		localAddresses = append(localAddresses, account.IPv4Address+"/32")
	}
	if account.IPv6Address != "" {
		localAddresses = append(localAddresses, account.IPv6Address+"/128")
	}
	// Fallback if API didn't return addresses
	if len(localAddresses) == 0 {
		localAddresses = []string{"172.16.0.2/32"}
	}

	return map[string]interface{}{
		"type":           "wireguard",
		"tag":            warpTag,
		"server":         warpEndpoint,
		"server_port":    warpPort,
		"local_address":  localAddresses,
		"private_key":    account.PrivateKey,
		"peer_public_key": warpPublicKey,
		"mtu":            1280,
	}
}

// SingboxWARPRouteRules converts WARP routes to Sing-box route rules
func SingboxWARPRouteRules(routes []models.WarpRoute) []map[string]interface{} {
	var rules []map[string]interface{}
	for _, route := range routes {
		rule := map[string]interface{}{
			"outbound": warpTag,
		}
		switch route.ResourceType {
		case "domain":
			rule["domain_suffix"] = []string{route.ResourceValue}
		case "ip":
			rule["ip_cidr"] = []string{route.ResourceValue + "/32"}
		case "cidr":
			rule["ip_cidr"] = []string{route.ResourceValue}
		}
		rules = append(rules, rule)
	}
	return rules
}

// ============================================================
// Xray WARP helpers
// ============================================================

// XrayWARPOutbound returns Xray WireGuard outbound config
func XrayWARPOutbound(account *WARPAccount) (tag string, protocol string, settings json.RawMessage) {
	addresses := []string{}
	if account.IPv4Address != "" {
		addresses = append(addresses, account.IPv4Address+"/32")
	}
	if account.IPv6Address != "" {
		addresses = append(addresses, account.IPv6Address+"/128")
	}
	if len(addresses) == 0 {
		addresses = []string{"172.16.0.2/32"}
	}

	settingsMap := map[string]interface{}{
		"secretKey": account.PrivateKey,
		"address":   addresses,
		"peers": []map[string]interface{}{
			{
				"publicKey": warpPublicKey,
				"endpoint":  fmt.Sprintf("%s:%d", warpEndpoint, warpPort),
			},
		},
		"mtu": 1280,
	}

	data, _ := json.Marshal(settingsMap)
	return warpTag, "wireguard", data
}

// XrayWARPRoutingRules converts WARP routes to Xray routing rules
func XrayWARPRoutingRules(routes []models.WarpRoute) []map[string]interface{} {
	var rules []map[string]interface{}
	for _, route := range routes {
		rule := map[string]interface{}{
			"type":        "field",
			"outboundTag": warpTag,
		}
		switch route.ResourceType {
		case "domain":
			rule["domain"] = []string{route.ResourceValue}
		case "ip":
			rule["ip"] = []string{route.ResourceValue}
		case "cidr":
			rule["ip"] = []string{route.ResourceValue}
		}
		rules = append(rules, rule)
	}
	return rules
}

// ============================================================
// Mihomo WARP helpers
// ============================================================

// MihomoWARPProxy returns Mihomo WireGuard proxy map
func MihomoWARPProxy(account *WARPAccount) map[string]interface{} {
	proxy := map[string]interface{}{
		"name":       warpTag,
		"type":       "wireguard",
		"server":     warpEndpoint,
		"port":       warpPort,
		"private-key": account.PrivateKey,
		"public-key":  warpPublicKey,
		"udp":        true,
		"mtu":        1280,
	}

	if account.IPv4Address != "" {
		proxy["ip"] = account.IPv4Address
	} else {
		proxy["ip"] = "172.16.0.2"
	}
	if account.IPv6Address != "" {
		proxy["ipv6"] = account.IPv6Address
	}

	return proxy
}

// MihomoWARPRules converts WARP routes to Mihomo rule strings
func MihomoWARPRules(routes []models.WarpRoute) []string {
	var rules []string
	for _, route := range routes {
		switch route.ResourceType {
		case "domain":
			rules = append(rules, fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", route.ResourceValue, warpTag))
		case "ip":
			rules = append(rules, fmt.Sprintf("IP-CIDR,%s/32,%s", route.ResourceValue, warpTag))
		case "cidr":
			rules = append(rules, fmt.Sprintf("IP-CIDR,%s,%s", route.ResourceValue, warpTag))
		}
	}
	return rules
}

// InjectWARP is a convenience function that loads WARP data and indicates if injection is needed
func InjectWARP(ctx *ConfigContext, coreID uint) (*WARPOutboundData, bool) {
	data, err := LoadWARPOutbound(ctx, coreID)
	if err != nil || data == nil {
		return nil, false
	}
	return data, true
}

// Helper for DB access in tests
func LoadWARPAccountFromDir(warpDir string) (*WARPAccount, error) {
	return loadWARPAccount(warpDir)
}
