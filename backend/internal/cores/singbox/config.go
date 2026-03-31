package singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// Config represents the Sing-box configuration
type Config struct {
	Log          *LogConfig          `json:"log"`
	DNS          *DNSConfig          `json:"dns,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
	Inbounds     []InboundConfig     `json:"inbounds"`
	Outbounds    []OutboundConfig    `json:"outbounds"`
	Route        *RouteConfig        `json:"route,omitempty"`
}

// LogConfig represents Sing-box log configuration
type LogConfig struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
	Output    string `json:"output,omitempty"`
}

// ExperimentalConfig represents experimental features (Clash API for stats)
type ExperimentalConfig struct {
	ClashAPI *ClashAPIConfig `json:"clash_api,omitempty"`
}

// ClashAPIConfig represents Clash-compatible API configuration
type ClashAPIConfig struct {
	ExternalController string `json:"external_controller"`
	Secret             string `json:"secret,omitempty"`
}

// DNSConfig represents DNS configuration
type DNSConfig struct {
	Servers []DNSServer `json:"servers"`
	Rules   []DNSRule   `json:"rules,omitempty"`
}

// DNSServer represents a DNS server entry
type DNSServer struct {
	Tag     string `json:"tag"`
	Address string `json:"address"`
}

// DNSRule represents a DNS routing rule
type DNSRule struct {
	Outbound string `json:"outbound,omitempty"`
	Server   string `json:"server,omitempty"`
}

// InboundConfig represents a single inbound configuration
type InboundConfig struct {
	Type      string          `json:"type"`
	Tag       string          `json:"tag"`
	Listen    string          `json:"listen,omitempty"`
	ListenPort int            `json:"listen_port,omitempty"`
	Users     json.RawMessage `json:"users,omitempty"`
	TLS       *TLSConfig      `json:"tls,omitempty"`
	Transport *TransportConfig `json:"transport,omitempty"`
	Sniff     bool            `json:"sniff,omitempty"`

	// Protocol-specific fields stored as raw JSON
	Extra map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to flatten Extra fields
func (c InboundConfig) MarshalJSON() ([]byte, error) {
	// Build a map with all standard fields
	m := make(map[string]interface{})
	m["type"] = c.Type
	m["tag"] = c.Tag
	if c.Listen != "" {
		m["listen"] = c.Listen
	}
	if c.ListenPort > 0 {
		m["listen_port"] = c.ListenPort
	}
	if len(c.Users) > 0 {
		m["users"] = json.RawMessage(c.Users)
	}
	if c.TLS != nil {
		m["tls"] = c.TLS
	}
	if c.Transport != nil {
		m["transport"] = c.Transport
	}
	if c.Sniff {
		m["sniff"] = c.Sniff
	}

	// Merge extra fields
	for k, v := range c.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}

	return json.Marshal(m)
}

// TLSConfig represents TLS settings
type TLSConfig struct {
	Enabled         bool                `json:"enabled"`
	ServerName      string              `json:"server_name,omitempty"`
	CertificatePath string              `json:"certificate_path,omitempty"`
	KeyPath         string              `json:"key_path,omitempty"`
	Reality         *RealityInlineConfig `json:"reality,omitempty"`
}

// RealityInlineConfig represents Sing-box Reality settings (nested under TLS)
type RealityInlineConfig struct {
	Enabled    bool              `json:"enabled"`
	Handshake  *RealityHandshake `json:"handshake,omitempty"`
	PrivateKey string            `json:"private_key"`
	ShortID    []string          `json:"short_id,omitempty"`
}

// RealityHandshake represents the handshake server config for Reality
type RealityHandshake struct {
	Server     string `json:"server"`
	ServerPort int    `json:"server_port,omitempty"`
}

// TransportConfig represents transport settings
type TransportConfig struct {
	Type        string `json:"type"`
	Path        string `json:"path,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
	Host        string `json:"host,omitempty"`
}

// OutboundConfig represents a single outbound configuration
type OutboundConfig struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
}

// RouteConfig represents routing configuration
type RouteConfig struct {
	Rules         []RouteRule `json:"rules,omitempty"`
	Final         string      `json:"final,omitempty"`
	AutoDetectInterface bool  `json:"auto_detect_interface,omitempty"`
}

// RouteRule represents a single routing rule
type RouteRule struct {
	Inbound  []string `json:"inbound,omitempty"`
	Outbound string   `json:"outbound"`
	Protocol []string `json:"protocol,omitempty"`
}

// ============================================================
// User types for different protocols
// ============================================================

// VMessUser represents a VMess user
type VMessUser struct {
	UUID    string `json:"uuid"`
	AlterID int    `json:"alter_id,omitempty"`
}

// VLESSUser represents a VLESS user
type VLESSUser struct {
	UUID string `json:"uuid"`
	Flow string `json:"flow,omitempty"`
}

// TrojanUser represents a Trojan user
type TrojanUser struct {
	Password string `json:"password"`
}

// ShadowsocksUser represents a Shadowsocks user (multi-user mode)
type ShadowsocksUser struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Hysteria2User represents a Hysteria2 user
type Hysteria2User struct {
	Password string `json:"password"`
}

// TUICUser represents a TUIC user
type TUICUser struct {
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}

// NaiveUser represents a NaiveProxy user
type NaiveUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ============================================================
// Config Generation
// ============================================================

// GenerateConfig generates Sing-box configuration from database
func GenerateConfig(db *gorm.DB, coreID uint) (*Config, error) {
	// Get core
	var core models.Core
	if err := db.First(&core, coreID).Error; err != nil {
		return nil, fmt.Errorf("failed to get core: %w", err)
	}

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

	// Build base config with Clash API for stats
	config := &Config{
		Log: &LogConfig{
			Level:     "warning",
			Timestamp: true,
			Output:    "/var/log/supervisor/singbox.log",
		},
		Experimental: &ExperimentalConfig{
			ClashAPI: &ClashAPIConfig{
				ExternalController: "127.0.0.1:9090",
				Secret:             "isolate-panel-secret",
			},
		},
		DNS: &DNSConfig{
			Servers: []DNSServer{
				{Tag: "google", Address: "https://dns.google/dns-query"},
				{Tag: "cloudflare", Address: "https://1.1.1.1/dns-query"},
				{Tag: "local", Address: "local"},
			},
		},
		Inbounds:  make([]InboundConfig, 0),
		Outbounds: make([]OutboundConfig, 0),
		Route: &RouteConfig{
			Final:               "direct",
			AutoDetectInterface: true,
		},
	}

	// Add user inbounds
	for _, inbound := range inbounds {
		inboundConfig, err := convertInbound(db, inbound)
		if err != nil {
			return nil, fmt.Errorf("failed to convert inbound %d: %w", inbound.ID, err)
		}
		config.Inbounds = append(config.Inbounds, *inboundConfig)
	}

	// Add outbounds
	for _, outbound := range outbounds {
		outboundConfig, err := convertOutbound(outbound)
		if err != nil {
			return nil, fmt.Errorf("failed to convert outbound %d: %w", outbound.ID, err)
		}
		config.Outbounds = append(config.Outbounds, *outboundConfig)
	}

	// Add default outbound if none exists
	if len(config.Outbounds) == 0 {
		config.Outbounds = append(config.Outbounds, OutboundConfig{
			Type: "direct",
			Tag:  "direct",
		})
	}

	return config, nil
}

// convertInbound converts database inbound model to Sing-box inbound config
func convertInbound(db *gorm.DB, inbound models.Inbound) (*InboundConfig, error) {
	// Get users assigned to this inbound
	var userMappings []models.UserInboundMapping
	if err := db.Where("inbound_id = ?", inbound.ID).Find(&userMappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get user mappings: %w", err)
	}

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

	// Map protocol name to Sing-box inbound type
	singboxType := mapSingboxProtocol(inbound.Protocol)

	// Generate tag from inbound ID and protocol
	tag := fmt.Sprintf("%s_%d", inbound.Protocol, inbound.ID)

	config := &InboundConfig{
		Type:      singboxType,
		Tag:       tag,
		Listen:    inbound.ListenAddress,
		ListenPort: inbound.Port,
		Sniff:     true,
		Extra:     make(map[string]interface{}),
	}

	// Build users based on protocol
	usersJSON, err := buildUsers(inbound.Protocol, inbound.ConfigJSON, users)
	if err != nil {
		return nil, err
	}
	if usersJSON != nil {
		config.Users = usersJSON
	}

	// Apply protocol-specific settings from ConfigJSON
	applyProtocolSettings(config, inbound.Protocol, inbound.ConfigJSON)

	// Add TLS if enabled
	if inbound.TLSEnabled {
		tlsConfig := &TLSConfig{
			Enabled: true,
		}

		// Load certificate paths from DB if bound
		if inbound.TLSCertID != nil {
			var cert models.Certificate
			if err := db.First(&cert, *inbound.TLSCertID).Error; err == nil {
				tlsConfig.CertificatePath = cert.CertPath
				tlsConfig.KeyPath = cert.KeyPath
				tlsConfig.ServerName = cert.Domain
			}
		}

		config.TLS = tlsConfig
	}

	// Add Reality settings if enabled (extends TLS)
	if inbound.RealityEnabled && inbound.RealityConfigJSON != "" {
		var realitySettings map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &realitySettings); err == nil {
			realityConfig := &RealityInlineConfig{
				Enabled: true,
			}

			// Parse handshake server (dest in Xray terms)
			if dest, ok := realitySettings["dest"].(string); ok {
				realityConfig.Handshake = &RealityHandshake{
					Server:     dest,
					ServerPort: 443,
				}
			}
			if pk, ok := realitySettings["privateKey"].(string); ok {
				realityConfig.PrivateKey = pk
			}
			if shortIds, ok := realitySettings["shortIds"].([]interface{}); ok {
				for _, si := range shortIds {
					if s, ok := si.(string); ok {
						realityConfig.ShortID = append(realityConfig.ShortID, s)
					}
				}
			}

			// Determine server name for TLS
			serverName := ""
			if serverNames, ok := realitySettings["serverNames"].([]interface{}); ok && len(serverNames) > 0 {
				if sn, ok := serverNames[0].(string); ok {
					serverName = sn
				}
			}

			// Ensure TLS config exists
			if config.TLS == nil {
				config.TLS = &TLSConfig{Enabled: true}
			}
			config.TLS.Reality = realityConfig
			if serverName != "" {
				config.TLS.ServerName = serverName
			}
			// Clear certificate paths — Reality doesn't use them
			config.TLS.CertificatePath = ""
			config.TLS.KeyPath = ""
		}
	}

	// Apply transport settings from ConfigJSON
	if inbound.ConfigJSON != "" {
		var cfgSettings map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err == nil {
			if transport, ok := cfgSettings["transport"].(string); ok && transport != "" && transport != "tcp" {
				switch transport {
				case "ws":
					wsPath := "/ws"
					if p, ok := cfgSettings["ws_path"].(string); ok && p != "" {
						wsPath = p
					}
					transportConfig := &TransportConfig{
						Type: "ws",
						Path: wsPath,
					}
					if host, ok := cfgSettings["ws_host"].(string); ok && host != "" {
						transportConfig.Host = host
					}
					config.Transport = transportConfig

				case "grpc":
					serviceName := "grpc"
					if sn, ok := cfgSettings["grpc_service_name"].(string); ok && sn != "" {
						serviceName = sn
					}
					config.Transport = &TransportConfig{
						Type:        "grpc",
						ServiceName: serviceName,
					}

				case "h2":
					h2Path := "/"
					if p, ok := cfgSettings["h2_path"].(string); ok && p != "" {
						h2Path = p
					}
					transportConfig := &TransportConfig{
						Type: "http",
						Path: h2Path,
					}
					if host, ok := cfgSettings["h2_host"].(string); ok && host != "" {
						transportConfig.Host = host
					}
					config.Transport = transportConfig
				}
			}
		}
	}

	return config, nil
}

// buildUsers builds user list for the protocol
func buildUsers(protocol string, configJSON string, users []models.User) (json.RawMessage, error) {
	if len(users) == 0 {
		return nil, nil
	}

	switch protocol {
	case "vmess":
		vmessUsers := make([]VMessUser, 0, len(users))
		for _, user := range users {
			vmessUsers = append(vmessUsers, VMessUser{
				UUID:    user.UUID,
				AlterID: 0,
			})
		}
		return json.Marshal(vmessUsers)

	case "vless":
		vlessUsers := make([]VLESSUser, 0, len(users))
		for _, user := range users {
			vlessUsers = append(vlessUsers, VLESSUser{
				UUID: user.UUID,
				// Flow will be set by caller if needed (e.g., for Reality)
			})
		}
		return json.Marshal(vlessUsers)

	case "trojan":
		trojanUsers := make([]TrojanUser, 0, len(users))
		for _, user := range users {
			trojanUsers = append(trojanUsers, TrojanUser{
				Password: user.UUID,
			})
		}
		return json.Marshal(trojanUsers)

	case "shadowsocks":
		// Sing-box SS multi-user mode
		ssUsers := make([]ShadowsocksUser, 0, len(users))
		for _, user := range users {
			ssUsers = append(ssUsers, ShadowsocksUser{
				Name:     fmt.Sprintf("user_%d", user.ID),
				Password: user.UUID,
			})
		}
		return json.Marshal(ssUsers)

	case "hysteria2":
		hyUsers := make([]Hysteria2User, 0, len(users))
		for _, user := range users {
			hyUsers = append(hyUsers, Hysteria2User{
				Password: user.UUID,
			})
		}
		return json.Marshal(hyUsers)

	case "tuic_v4":
		// TUIC v4 uses token-based auth
		tuicUsers := make([]TUICUser, 0, len(users))
		for _, user := range users {
			tuicUsers = append(tuicUsers, TUICUser{
				Password: user.UUID, // Token stored as password
				Name:     fmt.Sprintf("user_%d", user.ID),
			})
		}
		return json.Marshal(tuicUsers)

	case "tuic_v5":
		// TUIC v5 uses UUID + password auth
		tuicUsers := make([]TUICUser, 0, len(users))
		for _, user := range users {
			tuicUsers = append(tuicUsers, TUICUser{
				UUID:     user.UUID,
				Password: user.UUID, // Can be different; for now same
				Name:     fmt.Sprintf("user_%d", user.ID),
			})
		}
		return json.Marshal(tuicUsers)

	case "naive":
		naiveUsers := make([]NaiveUser, 0, len(users))
		for _, user := range users {
			naiveUsers = append(naiveUsers, NaiveUser{
				Username: fmt.Sprintf("user_%d", user.ID),
				Password: user.UUID,
			})
		}
		return json.Marshal(naiveUsers)

	case "http", "socks5", "mixed":
		// These protocols use simple username/password from ConfigJSON, not per-user
		return nil, nil

	default:
		return nil, nil
	}
}

// applyProtocolSettings applies protocol-specific extra settings from ConfigJSON
func applyProtocolSettings(config *InboundConfig, protocol string, configJSON string) {
	if configJSON == "" {
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &settings); err != nil {
		return
	}

	switch protocol {
	case "shadowsocks":
		// Sing-box requires method at inbound level
		if method, ok := settings["method"].(string); ok {
			config.Extra["method"] = method
		} else {
			config.Extra["method"] = "2022-blake3-aes-128-gcm" // Default
		}
		if password, ok := settings["password"].(string); ok {
			config.Extra["password"] = password // Server password for 2022 ciphers
		}

	case "hysteria2":
		if upMbps, ok := settings["up_mbps"]; ok {
			config.Extra["up_mbps"] = upMbps
		}
		if downMbps, ok := settings["down_mbps"]; ok {
			config.Extra["down_mbps"] = downMbps
		}
		if obfsType, ok := settings["obfs_type"].(string); ok && obfsType != "" {
			config.Extra["obfs"] = map[string]interface{}{
				"type":     obfsType,
				"password": settings["obfs_password"],
			}
		}

	case "tuic_v4", "tuic_v5":
		if cc, ok := settings["congestion_control"].(string); ok {
			config.Extra["congestion_control"] = cc
		}

	case "http", "socks5", "mixed":
		// Apply username/password if provided
		if username, ok := settings["username"].(string); ok && username != "" {
			usersList := []map[string]string{
				{"username": username, "password": settings["password"].(string)},
			}
			if data, err := json.Marshal(usersList); err == nil {
				config.Users = data
			}
		}
	}
}

// convertOutbound converts database outbound model to Sing-box outbound config
func convertOutbound(outbound models.Outbound) (*OutboundConfig, error) {
	singboxType := mapSingboxOutboundProtocol(outbound.Protocol)
	tag := fmt.Sprintf("%s_%d", outbound.Protocol, outbound.ID)

	return &OutboundConfig{
		Type: singboxType,
		Tag:  tag,
	}, nil
}

// mapSingboxProtocol maps our protocol names to Sing-box inbound types
func mapSingboxProtocol(protocol string) string {
	switch protocol {
	case "http":
		return "http"
	case "socks5":
		return "socks"
	case "mixed":
		return "mixed"
	case "shadowsocks":
		return "shadowsocks"
	case "vmess":
		return "vmess"
	case "vless":
		return "vless"
	case "trojan":
		return "trojan"
	case "hysteria2":
		return "hysteria2"
	case "tuic_v4", "tuic_v5":
		return "tuic"
	case "naive":
		return "naive"
	case "redirect":
		return "redirect"
	default:
		return protocol
	}
}

// mapSingboxOutboundProtocol maps our protocol names to Sing-box outbound types
func mapSingboxOutboundProtocol(protocol string) string {
	switch protocol {
	case "direct":
		return "direct"
	case "block":
		return "block"
	case "dns":
		return "dns"
	case "tor":
		return "tor"
	default:
		return protocol
	}
}

// ============================================================
// Config I/O and Validation
// ============================================================

// ValidateConfig validates Sing-box configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Check inbounds
	if len(config.Inbounds) == 0 {
		return fmt.Errorf("at least one inbound is required")
	}

	// Check for duplicate tags
	tags := make(map[string]bool)
	for _, inbound := range config.Inbounds {
		if tags[inbound.Tag] {
			return fmt.Errorf("duplicate inbound tag: %s", inbound.Tag)
		}
		tags[inbound.Tag] = true
	}

	for _, outbound := range config.Outbounds {
		if tags[outbound.Tag] {
			return fmt.Errorf("duplicate outbound tag: %s", outbound.Tag)
		}
		tags[outbound.Tag] = true
	}

	return nil
}

// WriteConfig writes configuration to file
func WriteConfig(config *Config, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ReadConfig reads configuration from file
func ReadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
