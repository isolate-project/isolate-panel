package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// Config represents the Xray configuration
type Config struct {
	Log       *LogConfig       `json:"log"`
	API       *APIConfig       `json:"api"`
	Stats     *StatsConfig     `json:"stats"`
	Policy    *PolicyConfig    `json:"policy"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   *RoutingConfig   `json:"routing"`
}

// LogConfig represents Xray log configuration
type LogConfig struct {
	Access   string `json:"access"`
	Error    string `json:"error"`
	LogLevel string `json:"loglevel"`
}

// APIConfig represents Xray gRPC API configuration
type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

// StatsConfig represents Xray stats configuration
type StatsConfig struct{}

// PolicyConfig represents Xray policy configuration
type PolicyConfig struct {
	Levels map[string]LevelPolicy `json:"levels"`
	System *SystemPolicy          `json:"system"`
}

// LevelPolicy represents policy for a specific level
type LevelPolicy struct {
	StatsUserUplink   bool `json:"statsUserUplink"`
	StatsUserDownlink bool `json:"statsUserDownlink"`
}

// SystemPolicy represents system-wide policy
type SystemPolicy struct {
	StatsInboundUplink   bool `json:"statsInboundUplink"`
	StatsInboundDownlink bool `json:"statsInboundDownlink"`
}

// InboundConfig represents a single inbound configuration
type InboundConfig struct {
	Tag            string          `json:"tag"`
	Listen         string          `json:"listen"`
	Port           int             `json:"port"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings *StreamConfig   `json:"streamSettings,omitempty"`
	Sniffing       *SniffingConfig `json:"sniffing,omitempty"`
}

// OutboundConfig represents a single outbound configuration
type OutboundConfig struct {
	Tag      string          `json:"tag"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings"`
}

// StreamConfig represents stream (transport) configuration
type StreamConfig struct {
	Network       string      `json:"network"`
	Security      string      `json:"security,omitempty"`
	TLSConfig     *TLSConfig  `json:"tlsSettings,omitempty"`
	RealityConfig *RealityConfig `json:"realitySettings,omitempty"`
	WSConfig      *WSConfig   `json:"wsSettings,omitempty"`
	HTTPConfig    *HTTPConfig `json:"httpSettings,omitempty"`
	GRPCConfig    *GRPCConfig `json:"grpcSettings,omitempty"`
}

// TLSConfig represents TLS settings
type TLSConfig struct {
	ServerName string `json:"serverName"`
	CertFile   string `json:"certificateFile,omitempty"`
	KeyFile    string `json:"keyFile,omitempty"`
}

// RealityConfig represents Xray Reality settings
type RealityConfig struct {
	Show        bool     `json:"show,omitempty"`
	Dest        string   `json:"dest"`
	Xver        int      `json:"xver,omitempty"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
}

// WSConfig represents WebSocket settings
type WSConfig struct {
	Path string `json:"path"`
	Host string `json:"host,omitempty"`
}

// HTTPConfig represents HTTP/2 settings
type HTTPConfig struct {
	Host []string `json:"host"`
	Path string   `json:"path"`
}

// GRPCConfig represents gRPC settings
type GRPCConfig struct {
	ServiceName string `json:"serviceName"`
}

// SniffingConfig represents sniffing configuration
type SniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

// RoutingConfig represents routing configuration
type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy"`
	Rules          []RoutingRule `json:"rules"`
	Balancers      []interface{} `json:"balancers,omitempty"`
}

// RoutingRule represents a single routing rule
type RoutingRule struct {
	Type        string   `json:"type"`
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
}

// GenerateConfig generates Xray configuration from database
func GenerateConfig(ctx *cores.ConfigContext, coreID uint) (*Config, error) {
	db := ctx.DB

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

	// Build base config with API and stats enabled
	config := &Config{
		Log: &LogConfig{
			Access:   "/var/log/supervisor/xray_access.log",
			Error:    "/var/log/supervisor/xray_error.log",
			LogLevel: "warning",
		},
		API: &APIConfig{
			Tag:      "api",
			Services: []string{"HandlerService", "StatsService"},
		},
		Stats: &StatsConfig{},
		Policy: &PolicyConfig{
			Levels: map[string]LevelPolicy{
				"0": {
					StatsUserUplink:   true,
					StatsUserDownlink: true,
				},
			},
			System: &SystemPolicy{
				StatsInboundUplink:   true,
				StatsInboundDownlink: true,
			},
		},
		Inbounds:  make([]InboundConfig, 0),
		Outbounds: make([]OutboundConfig, 0),
		Routing: &RoutingConfig{
			DomainStrategy: "AsIs",
			Rules: []RoutingRule{
				{
					Type:        "field",
					InboundTag:  []string{"api"},
					OutboundTag: "api",
				},
			},
		},
	}

	// Add API inbound (for gRPC)
	config.Inbounds = append(config.Inbounds, InboundConfig{
		Tag:      "api",
		Listen:   "127.0.0.1",
		Port:     10085,
		Protocol: "dokodemo-door",
		Settings: safeMarshal(map[string]interface{}{
			"address": "127.0.0.1",
		}),
	})

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

	// Add user inbounds
	for _, inbound := range inbounds {
		inboundConfig, err := convertInbound(db, inbound, usersByInbound[inbound.ID])
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
			Tag:      "direct",
			Protocol: "freedom",
			Settings: safeMarshal(map[string]interface{}{
				"domainStrategy": "UseIP",
			}),
		})
	}

	// Inject WARP WireGuard outbound + routing rules
	if warpData, ok := cores.InjectWARP(ctx, coreID); ok {
		tag, protocol, settings := cores.XrayWARPOutbound(warpData.Account)
		config.Outbounds = append(config.Outbounds, OutboundConfig{
			Tag:      tag,
			Protocol: protocol,
			Settings: settings,
		})
		// Add WARP routing rules
		warpRules := cores.XrayWARPRoutingRules(warpData.Routes)
		for _, wr := range warpRules {
			rule := RoutingRule{
				Type:        "field",
				OutboundTag: "warp-out",
			}
			if domain, ok := wr["domain"].([]string); ok {
				rule.Domain = domain
			}
			if ip, ok := wr["ip"].([]string); ok {
				rule.IP = ip
			}
			config.Routing.Rules = append(config.Routing.Rules, rule)
		}
	}

	// Inject GeoIP/GeoSite routing rules
	if geoData, ok := cores.InjectGeo(ctx, coreID); ok {
		geoRules := cores.XrayGeoRoutingRules(geoData.Rules)
		for _, gr := range geoRules {
			rule := RoutingRule{Type: "field"}
			if outboundTag, ok := gr["outboundTag"].(string); ok {
				rule.OutboundTag = outboundTag
			}
			if domain, ok := gr["domain"].([]string); ok {
				rule.Domain = domain
			}
			if ip, ok := gr["ip"].([]string); ok {
				rule.IP = ip
			}
			config.Routing.Rules = append(config.Routing.Rules, rule)
		}
	}

	return config, nil
}

// convertInbound converts database inbound model to Xray inbound config
func convertInbound(db *gorm.DB, inbound models.Inbound, users []models.User) (*InboundConfig, error) {
	// Build settings based on protocol
	settings, err := buildInboundSettings(inbound.Protocol, []byte(inbound.ConfigJSON), users)
	if err != nil {
		return nil, err
	}

	// Generate tag from inbound ID and protocol
	tag := fmt.Sprintf("%s_%d", inbound.Protocol, inbound.ID)

	config := &InboundConfig{
		Tag:      tag,
		Listen:   inbound.ListenAddress,
		Port:     inbound.Port,
		Protocol: inbound.Protocol,
		Settings: settings,
	}

	// Add stream settings if TLS is enabled
	if inbound.TLSEnabled {
		streamConfig := &StreamConfig{
			Network:  "tcp",
			Security: "tls",
			TLSConfig: &TLSConfig{},
		}

		// Load certificate paths from DB if bound
		if inbound.TLSCertID != nil {
			var cert models.Certificate
			if err := db.First(&cert, *inbound.TLSCertID).Error; err == nil {
				streamConfig.TLSConfig.CertFile = cert.CertPath
				streamConfig.TLSConfig.KeyFile = cert.KeyPath
				streamConfig.TLSConfig.ServerName = cert.Domain
			}
		}

		config.StreamSettings = streamConfig
	}

	// Add Reality settings if enabled (overrides TLS)
	if inbound.RealityEnabled && inbound.RealityConfigJSON != "" {
		var realitySettings map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &realitySettings); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse RealityConfigJSON")
		} else {
			realityConfig := &RealityConfig{}

			if dest, ok := realitySettings["dest"].(string); ok {
				realityConfig.Dest = dest
			}
			if serverNames, ok := realitySettings["serverNames"].([]interface{}); ok {
				for _, sn := range serverNames {
					if s, ok := sn.(string); ok {
						realityConfig.ServerNames = append(realityConfig.ServerNames, s)
					}
				}
			}
			if pk, ok := realitySettings["privateKey"].(string); ok {
				realityConfig.PrivateKey = pk
			}
			if shortIds, ok := realitySettings["shortIds"].([]interface{}); ok {
				for _, si := range shortIds {
					if s, ok := si.(string); ok {
						realityConfig.ShortIds = append(realityConfig.ShortIds, s)
					}
				}
			}

			if config.StreamSettings == nil {
				config.StreamSettings = &StreamConfig{Network: "tcp"}
			}
		config.StreamSettings.Security = "reality"
			config.StreamSettings.TLSConfig = nil // Reality replaces TLS
			config.StreamSettings.RealityConfig = realityConfig
		}
	}

	// Apply transport settings from ConfigJSON
	if inbound.ConfigJSON != "" {
		var cfgSettings map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse ConfigJSON")
		} else {
			if transport, ok := cfgSettings["transport"].(string); ok && transport != "" && transport != "tcp" {
				if config.StreamSettings == nil {
					config.StreamSettings = &StreamConfig{Network: "tcp"}
				}

				switch transport {
				case "ws":
					config.StreamSettings.Network = "ws"
					wsPath := "/ws"
					if p, ok := cfgSettings["ws_path"].(string); ok && p != "" {
						wsPath = p
					}
					wsConfig := &WSConfig{Path: wsPath}
					if host, ok := cfgSettings["ws_host"].(string); ok && host != "" {
						wsConfig.Host = host
					}
					config.StreamSettings.WSConfig = wsConfig

				case "grpc":
					config.StreamSettings.Network = "grpc"
					serviceName := "grpc"
					if sn, ok := cfgSettings["grpc_service_name"].(string); ok && sn != "" {
						serviceName = sn
					}
					config.StreamSettings.GRPCConfig = &GRPCConfig{ServiceName: serviceName}

				case "h2":
					config.StreamSettings.Network = "h2"
					h2Path := "/"
					if p, ok := cfgSettings["h2_path"].(string); ok && p != "" {
						h2Path = p
					}
					h2Config := &HTTPConfig{Path: h2Path}
					if host, ok := cfgSettings["h2_host"].(string); ok && host != "" {
						h2Config.Host = []string{host}
					}
					config.StreamSettings.HTTPConfig = h2Config

				case "xhttp":
					// XHTTP is Xray-exclusive, uses splithttp network
					config.StreamSettings.Network = "splithttp"
				}
			}
		}
	}

	// Add sniffing for most protocols
	if inbound.Protocol != "http" && inbound.Protocol != "socks" {
		config.Sniffing = &SniffingConfig{
			Enabled:      true,
			DestOverride: []string{"http", "tls"},
		}
	}

	return config, nil
}

// buildInboundSettings builds protocol-specific settings
func buildInboundSettings(protocol string, baseSettings json.RawMessage, users []models.User) (json.RawMessage, error) {
	var settings map[string]interface{}
	if len(baseSettings) > 0 {
		if err := json.Unmarshal(baseSettings, &settings); err != nil {
			return nil, err
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Handle socks/http which use accounts instead of clients
	switch protocol {
	case "socks":
		if len(users) > 0 {
			settings["auth"] = "password"
			accounts := make([]map[string]string, 0, len(users))
			for _, user := range users {
				accounts = append(accounts, map[string]string{
					"user": fmt.Sprintf("user_%d", user.ID),
					"pass": user.UUID,
				})
			}
			settings["accounts"] = accounts
		} else {
			settings["auth"] = "noauth"
		}
		settings["udp"] = true
		return json.Marshal(settings)

	case "http":
		if len(users) > 0 {
			accounts := make([]map[string]string, 0, len(users))
			for _, user := range users {
				accounts = append(accounts, map[string]string{
					"user": fmt.Sprintf("user_%d", user.ID),
					"pass": user.UUID,
				})
			}
			settings["accounts"] = accounts
		}
		return json.Marshal(settings)
	}

	// Add clients based on protocol (vmess, vless, trojan, etc.)
	clients := buildClients(protocol, users)
	if len(clients) > 0 {
		settings["clients"] = clients
	}

	return json.Marshal(settings)
}

// buildClients builds client list for the protocol
func buildClients(protocol string, users []models.User) []map[string]interface{} {
	clients := make([]map[string]interface{}, 0, len(users))

	for _, user := range users {
		switch protocol {
		case "vmess":
			clients = append(clients, map[string]interface{}{
				"id":      user.UUID,
				"level":   0,
				"email":   fmt.Sprintf("user_%d", user.ID),
				"alterId": 0,
			})
		case "vless":
			clients = append(clients, map[string]interface{}{
				"id":         user.UUID,
				"level":      0,
				"email":      fmt.Sprintf("user_%d", user.ID),
				"flow":       "",
				"encryption": "none",
			})
		case "trojan":
			clients = append(clients, map[string]interface{}{
				"password": user.UUID,
				"level":    0,
				"email":    fmt.Sprintf("user_%d", user.ID),
			})
		case "shadowsocks":
			clients = append(clients, map[string]interface{}{
				"method":   "chacha20-poly1305",
				"password": user.UUID,
				"email":    fmt.Sprintf("user_%d", user.ID),
				"level":    0,
			})
		case "hysteria2":
			clients = append(clients, map[string]interface{}{
				"password": user.UUID,
				"email":    fmt.Sprintf("user_%d", user.ID),
			})
		case "xhttp":
			// XHTTP (Xray exclusive) - uses path-based routing
			clients = append(clients, map[string]interface{}{
				"id":    user.UUID,
				"email": fmt.Sprintf("user_%d", user.ID),
			})
		}
	}

	return clients
}

// buildStreamSettings builds stream/transport settings
func buildStreamSettings(transport string, transportSettings json.RawMessage) *StreamConfig {
	config := &StreamConfig{
		Network: "tcp",
	}

	switch transport {
	case "ws":
		config.Network = "ws"
		config.WSConfig = &WSConfig{
			Path: "/ws",
		}
	case "http":
		config.Network = "h2"
		config.HTTPConfig = &HTTPConfig{
			Host: []string{},
			Path: "/",
		}
	case "grpc":
		config.Network = "grpc"
		config.GRPCConfig = &GRPCConfig{
			ServiceName: "grpc",
		}
	}

	if len(transportSettings) > 0 {
		// Override with custom settings
		var custom map[string]interface{}
		if err := json.Unmarshal(transportSettings, &custom); err == nil {
			// Apply custom settings as needed
		}
	}

	return config
}

// convertOutbound converts database outbound model to Xray outbound config
func convertOutbound(outbound models.Outbound) (*OutboundConfig, error) {
	settings, err := buildOutboundSettings(outbound.Protocol, []byte(outbound.ConfigJSON))
	if err != nil {
		return nil, err
	}

	// Generate tag from outbound ID and protocol
	tag := fmt.Sprintf("%s_%d", outbound.Protocol, outbound.ID)

	return &OutboundConfig{
		Tag:      tag,
		Protocol: outbound.Protocol,
		Settings: settings,
	}, nil
}

// buildOutboundSettings builds outbound protocol settings
func buildOutboundSettings(protocol string, baseSettings json.RawMessage) (json.RawMessage, error) {
	if len(baseSettings) > 0 {
		return baseSettings, nil
	}

	// Default settings for common protocols
	settings := make(map[string]interface{})

	switch protocol {
	case "freedom":
		settings["domainStrategy"] = "UseIP"
	case "blackhole":
		// No settings needed
	case "dns":
		// No settings needed
	}

	return json.Marshal(settings)
}

// safeMarshal marshals data to JSON, returns empty object on error
func safeMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

// ValidateConfig validates Xray configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Check API configuration
	if config.API == nil {
		return fmt.Errorf("API configuration is required")
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
