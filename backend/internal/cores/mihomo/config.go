package mihomo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
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
		Secret:             "isolate-panel-secret", // Should be from config
		IPv6:               true,
		Proxies:            make([]Proxy, 0),
		Rules:              make([]string, 0),
	}

	// Add inbounds as proxies
	for _, inbound := range inbounds {
		proxy, err := convertInboundToProxy(db, inbound)
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
func convertInboundToProxy(db *gorm.DB, inbound models.Inbound) (*Proxy, error) {
	// Get users assigned to this inbound
	var userMappings []models.UserInboundMapping
	if err := db.Where("inbound_id = ?", inbound.ID).Find(&userMappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get user mappings: %w", err)
	}

	// Get users
	userIDs := make([]uint, len(userMappings))
	for i, m := range userMappings {
		userIDs[i] = m.UserID
	}

	var users []models.User
	if len(userIDs) > 0 {
		if err := db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}
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
		proxy.Cipher = "chacha20-poly1305"
		// For Mihomo, we need to handle multiple users differently
		// Using first user's password for simplicity
		if len(users) > 0 {
			proxy.Password = users[0].UUID
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
		proxy.Password = "sudoku-password" // Should be from inbound settings
	case "ssr":
		// ShadowsocksR (Mihomo exclusive)
		proxy.Type = "ssr"
		proxy.Protocol = "origin"
		proxy.Obfs = "plain"
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
		if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &realitySettings); err == nil {
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
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err == nil {
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
