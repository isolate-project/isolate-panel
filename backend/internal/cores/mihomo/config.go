package mihomo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// Config represents the Mihomo (Clash.Meta) configuration
type Config struct {
	Port               int                    `yaml:"port"`
	SocksPort          int                    `yaml:"socks-port"`
	MixedPort          int                    `yaml:"mixed-port,omitempty"`
	AllowLan           bool                   `yaml:"allow-lan"`
	Mode               string                 `yaml:"mode"`
	LogLevel           string                 `yaml:"log-level"`
	ExternalController string                 `yaml:"external-controller,omitempty"`
	Secret             string                 `yaml:"secret,omitempty"`
	IPv6               bool                   `yaml:"ipv6"`
	Interface          string                 `yaml:"interface,omitempty"`
	FallbackDNS        string                 `yaml:"fallback-dns,omitempty"`
	DNS                map[string]interface{} `yaml:"dns,omitempty"`
	Tun                map[string]interface{} `yaml:"tun,omitempty"`
	Proxies            []Proxy                `yaml:"proxies"`
	ProxyGroups        []ProxyGroup           `yaml:"proxy-groups,omitempty"`
	Rules              []string               `yaml:"rules"`
}

// Proxy represents a Mihomo proxy (inbound or outbound)
type Proxy struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Port     int                    `yaml:"port,omitempty"`
	Listen   string                 `yaml:"listen,omitempty"`
	Password string                 `yaml:"password,omitempty"`
	Cipher   string                 `yaml:"cipher,omitempty"`
	Protocol string                 `yaml:"protocol,omitempty"`
	Obfs     string                 `yaml:"obfs,omitempty"`
	Users    []ProxyUser            `yaml:"users,omitempty"`
	Extra    map[string]interface{} `yaml:",inline"`
}

// ProxyUser represents a user for a proxy
type ProxyUser struct {
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
}

// ProxyGroup represents a proxy group for load balancing
type ProxyGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies"`
	Strategy string   `yaml:"strategy,omitempty"`
}

// GenerateConfig generates Mihomo configuration from database
func GenerateConfig(ctx *cores.ConfigContext, coreID uint) (*Config, error) {
	db := ctx.DB

	// Get inbounds for this core
	var inbounds []models.Inbound
	if err := db.Where("core_id = ?", coreID).Find(&inbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get inbounds: %w", err)
	}

	// Get outbounds for this core
	var outbounds []models.Outbound
	if err := db.Where("core_id = ?", coreID).Find(&outbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get outbounds: %w", err)
	}

	// Build base config
	config := &Config{
		Port:               0, // Disabled - we use mixed-port
		SocksPort:          0, // Disabled
		MixedPort:          0, // Will be set if we have mixed inbounds
		AllowLan:           true,
		Mode:               "rule",
		LogLevel:           "warning",
		ExternalController: "127.0.0.1:9091",
		Secret:             ctx.CoreAPISecret,
		IPv6:               true,
		Proxies:            make([]Proxy, 0),
		Rules:              make([]string, 0),
	}

	// Batch-load all user mappings for this core's inbounds (eliminates N+1)
	inboundIDs := make([]uint, len(inbounds))
	for i, ib := range inbounds {
		inboundIDs[i] = ib.ID
	}

	usersByInbound := make(map[uint][]models.User)
	if len(inboundIDs) > 0 {
		var mappings []models.UserInboundMapping
		if err := db.Where("inbound_id IN ?", inboundIDs).Find(&mappings).Error; err != nil {
			return nil, fmt.Errorf("failed to batch-load user mappings: %w", err)
		}

		allUserIDs := make(map[uint]bool)
		for _, m := range mappings {
			allUserIDs[m.UserID] = true
		}

		var allUsers []models.User
		if len(allUserIDs) > 0 {
			uids := make([]uint, 0, len(allUserIDs))
			for uid := range allUserIDs {
				uids = append(uids, uid)
			}
			if err := db.Where("id IN ?", uids).Find(&allUsers).Error; err != nil {
				return nil, fmt.Errorf("failed to batch-load users: %w", err)
			}
		}

		userMap := make(map[uint]models.User)
		for _, u := range allUsers {
			userMap[u.ID] = u
		}

		for _, m := range mappings {
			if user, ok := userMap[m.UserID]; ok {
				usersByInbound[m.InboundID] = append(usersByInbound[m.InboundID], user)
			}
		}
	}

	// Add inbounds as proxies
	for _, inbound := range inbounds {
		proxy, err := convertInboundToProxy(db, inbound, usersByInbound[inbound.ID])
		if err != nil {
			return nil, fmt.Errorf("failed to convert inbound %d: %w", inbound.ID, err)
		}
		config.Proxies = append(config.Proxies, *proxy)
	}

	// Add outbounds as proxies
	for _, outbound := range outbounds {
		proxy, err := convertOutboundToProxy(outbound)
		if err != nil {
			return nil, fmt.Errorf("failed to convert outbound %d: %w", outbound.ID, err)
		}
		config.Proxies = append(config.Proxies, *proxy)
	}

	// Inject WARP WireGuard proxy + routing rules
	if warpData, ok := cores.InjectWARP(ctx, coreID); ok {
		wgProxy := cores.MihomoWARPProxy(warpData.Account)
		proxy := Proxy{
			Name:  "warp-out",
			Type:  "wireguard",
			Extra: wgProxy,
		}
		config.Proxies = append(config.Proxies, proxy)
		// Add WARP routing rules (before default MATCH)
		warpRules := cores.MihomoWARPRules(warpData.Routes)
		config.Rules = append(config.Rules, warpRules...)
	}

	// Inject GeoIP/GeoSite routing rules
	if geoData, ok := cores.InjectGeo(ctx, coreID); ok {
		geoRules := cores.MihomoGeoRules(geoData.Rules)
		config.Rules = append(config.Rules, geoRules...)
	}

	// Add default rule if no rules
	if len(config.Rules) == 0 {
		config.Rules = append(config.Rules, "MATCH,DIRECT")
	} else {
		// Always add MATCH,DIRECT as final fallback
		config.Rules = append(config.Rules, "MATCH,DIRECT")
	}

	return config, nil
}

// convertInboundToProxy converts database inbound to Mihomo proxy
func convertInboundToProxy(db *gorm.DB, inbound models.Inbound, users []models.User) (*Proxy, error) {
	// Parse inbound ConfigJSON once for all protocol cases
	var cfgSettings map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err != nil {
			logger.Log.Warn().Msgf(" Failed to parse ConfigJSON for inbound %d: %v", inbound.ID, err)
		}
	}
	if cfgSettings == nil {
		cfgSettings = make(map[string]interface{})
	}

	// Build proxy based on protocol
	proxy := &Proxy{
		Name: fmt.Sprintf("%s_%d", inbound.Protocol, inbound.ID),
		Type: mapMihomoProtocol(inbound.Protocol),
		Port: inbound.Port,
	}

	// Set listen address
	if inbound.ListenAddress != "" && inbound.ListenAddress != "0.0.0.0" {
		proxy.Listen = inbound.ListenAddress
	}

	// Add protocol-specific settings
	switch inbound.Protocol {
	case "shadowsocks":
		proxy.Cipher = getStringOrDefault(cfgSettings, "method", "2022-blake3-aes-128-gcm")
		// For 2022 ciphers: use server-level password + multi-user list
		if strings.HasPrefix(proxy.Cipher, "2022-") {
			if serverPass, ok := cfgSettings["password"].(string); ok {
				proxy.Password = serverPass
			}
			if len(users) > 0 {
				proxy.Users = make([]ProxyUser, len(users))
				for i, user := range users {
					proxy.Users[i] = ProxyUser{
						Name:     fmt.Sprintf("user_%d", user.ID),
						Password: user.UUID,
					}
				}
			}
		} else {
			// For AEAD ciphers: multi-user not supported, use first user
			if len(users) > 1 {
				logger.Log.Warn().Msgf(" Mihomo SS with AEAD cipher %q supports only 1 user on inbound %q; using first user", proxy.Cipher, inbound.Name)
			}
			if len(users) > 0 {
				proxy.Password = users[0].UUID
			}
		}
	case "trojan":
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "vmess":
		// VMess in Mihomo uses different structure
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "vless":
		// VLESS support in Mihomo
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "hysteria2":
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "tuic":
		// TUIC v5 support
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	}

	// Mihomo-exclusive protocols
	switch inbound.Protocol {
	case "mieru":
		// Mieru protocol (Mihomo exclusive)
		proxy.Type = "mieru"
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "sudoku":
		// Sudoku protocol (Mihomo exclusive)
		proxy.Type = "sudoku"
		if pass, ok := cfgSettings["password"].(string); ok {
			proxy.Password = pass
		}
	case "ssr":
		// ShadowsocksR (Mihomo exclusive)
		proxy.Type = "ssr"
		proxy.Protocol = getStringOrDefault(cfgSettings, "protocol", "origin")
		proxy.Obfs = getStringOrDefault(cfgSettings, "obfs", "plain")
		proxy.Cipher = getStringOrDefault(cfgSettings, "method", "chacha20-poly1305")
		if len(users) > 0 {
			proxy.Password = users[0].UUID
		}
	case "snell":
		// Snell protocol (Mihomo exclusive)
		proxy.Type = "snell"
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "trusttunnel":
		// TrustTunnel protocol (Mihomo exclusive)
		proxy.Type = "trusttunnel"
		if len(users) > 0 {
			proxy.Users = make([]ProxyUser, len(users))
			for i, user := range users {
				proxy.Users[i] = ProxyUser{
					Name:     fmt.Sprintf("user_%d", user.ID),
					Password: user.UUID,
				}
			}
		}
	case "masque":
		// MASQUE HTTP/3 proxy (Mihomo exclusive, outbound only)
		proxy.Type = "masque"
		// MASQUE typically uses URL-based configuration
		// Extra settings from ConfigJSON will be applied
	}

	// Add TLS if enabled
	if inbound.TLSEnabled {
		if proxy.Extra == nil {
			proxy.Extra = make(map[string]interface{})
		}
		proxy.Extra["tls"] = true
		proxy.Extra["skip-cert-verify"] = false

		// Load certificate info from DB if bound
		if inbound.TLSCertID != nil {
			var cert models.Certificate
			if err := db.First(&cert, *inbound.TLSCertID).Error; err == nil {
				proxy.Extra["servername"] = cert.Domain
			}
		}
	}

	// Add Reality settings if enabled
	if inbound.RealityEnabled && inbound.RealityConfigJSON != "" {
		if proxy.Extra == nil {
			proxy.Extra = make(map[string]interface{})
		}

		var realitySettings map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &realitySettings); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse RealityConfigJSON")
		} else {
			realityOpts := make(map[string]interface{})
			if pk, ok := realitySettings["publicKey"].(string); ok {
				realityOpts["public-key"] = pk
			}
			if shortId, ok := realitySettings["shortIds"].([]interface{}); ok && len(shortId) > 0 {
				if s, ok := shortId[0].(string); ok {
					realityOpts["short-id"] = s
				}
			}
			proxy.Extra["reality-opts"] = realityOpts

			// Set server names for SNI
			if serverNames, ok := realitySettings["serverNames"].([]interface{}); ok && len(serverNames) > 0 {
				if sn, ok := serverNames[0].(string); ok {
					proxy.Extra["servername"] = sn
				}
			}
		}
	}

	// Apply transport settings from ConfigJSON
	if inbound.ConfigJSON != "" {
		var cfgSettings map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse ConfigJSON")
		} else {
			if transport, ok := cfgSettings["transport"].(string); ok && transport != "" && transport != "tcp" {
				if proxy.Extra == nil {
					proxy.Extra = make(map[string]interface{})
				}
				proxy.Extra["network"] = transport

				switch transport {
				case "ws":
					wsOpts := make(map[string]interface{})
					if p, ok := cfgSettings["ws_path"].(string); ok && p != "" {
						wsOpts["path"] = p
					} else {
						wsOpts["path"] = "/ws"
					}
					if host, ok := cfgSettings["ws_host"].(string); ok && host != "" {
						wsOpts["headers"] = map[string]string{"Host": host}
					}
					proxy.Extra["ws-opts"] = wsOpts

				case "grpc":
					grpcOpts := make(map[string]interface{})
					if sn, ok := cfgSettings["grpc_service_name"].(string); ok && sn != "" {
						grpcOpts["grpc-service-name"] = sn
					} else {
						grpcOpts["grpc-service-name"] = "grpc"
					}
					proxy.Extra["grpc-opts"] = grpcOpts

				case "h2":
					h2Opts := make(map[string]interface{})
					if p, ok := cfgSettings["h2_path"].(string); ok && p != "" {
						h2Opts["path"] = p
					} else {
						h2Opts["path"] = "/"
					}
					if host, ok := cfgSettings["h2_host"].(string); ok && host != "" {
						h2Opts["host"] = []string{host}
					}
					proxy.Extra["h2-opts"] = h2Opts
				}
			}
		}
	}

	return proxy, nil
}

// convertOutboundToProxy converts database outbound to Mihomo proxy
func convertOutboundToProxy(outbound models.Outbound) (*Proxy, error) {
	proxy := &Proxy{
		Name: fmt.Sprintf("%s_%d", outbound.Protocol, outbound.ID),
		Type: mapMihomoProtocol(outbound.Protocol),
	}

	// Add extra settings from ConfigJSON if present
	if outbound.ConfigJSON != "" {
		var extra map[string]interface{}
		if err := yaml.Unmarshal([]byte(outbound.ConfigJSON), &extra); err == nil {
			proxy.Extra = extra
		}
	}

	return proxy, nil
}

// mapMihomoProtocol maps our protocol names to Mihomo protocol names
func mapMihomoProtocol(protocol string) string {
	switch protocol {
	case "http":
		return "http"
	case "socks":
		return "socks"
	case "mixed":
		return "mixed"
	case "shadowsocks":
		return "ss"
	case "shadowsocksr":
		return "ssr"
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	case "trojan":
		return "trojan"
	case "hysteria":
		return "hysteria"
	case "hysteria2":
		return "hysteria2"
	case "tuic":
		return "tuic"
	case "mieru":
		return "mieru"
	case "sudoku":
		return "sudoku"
	case "snell":
		return "snell"
	case "trusttunnel":
		return "trusttunnel"
	case "masque":
		return "masque"
	case "direct":
		return "direct"
	case "block":
		return "block"
	case "dns":
		return "dns"
	default:
		return protocol
	}
}

// ValidateConfig validates Mihomo configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Check for required fields
	if config.Mode == "" {
		return fmt.Errorf("mode is required")
	}

	// Check for duplicate proxy names
	names := make(map[string]bool)
	for _, proxy := range config.Proxies {
		if names[proxy.Name] {
			return fmt.Errorf("duplicate proxy name: %s", proxy.Name)
		}
		names[proxy.Name] = true
	}

	return nil
}

// WriteConfig writes configuration to YAML file
func WriteConfig(config *Config, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ReadConfig reads configuration from YAML file
func ReadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// getStringOrDefault returns a string value from the config map or the default
func getStringOrDefault(config map[string]interface{}, key, defaultVal string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}
