package xray

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

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
	Network       string           `json:"network"`
	Security      string           `json:"security,omitempty"`
	TLSConfig     *TLSConfig       `json:"tlsSettings,omitempty"`
	RealityConfig *RealityConfig   `json:"realitySettings,omitempty"`
	WSConfig      *WSConfig        `json:"wsSettings,omitempty"`
	HTTPConfig    *HTTPConfig      `json:"httpSettings,omitempty"`
	GRPCConfig    *GRPCConfig      `json:"grpcSettings,omitempty"`
	XHTTPConfig   *XHTTPConfig     `json:"splithttpSettings,omitempty"`
	Finalmask     *FinalmaskConfig `json:"finalmask,omitempty"`
}

// TLSConfig represents TLS settings
type TLSConfig struct {
	ServerName    string `json:"serverName"`
	CertFile      string `json:"certificateFile,omitempty"`
	KeyFile       string `json:"keyFile,omitempty"`
	ECHForceQuery string `json:"echForceQuery,omitempty"` // "off", "strict", "full" (default in v26+ is "full")
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

// TUNConfig represents TUN inbound settings
type TUNConfig struct {
	Name         string                 `json:"name,omitempty"`
	MTU          int                    `json:"mtu,omitempty"`
	Stack        string                 `json:"stack,omitempty"`
	Inet4Address string                 `json:"inet4_address,omitempty"`
	Inet6Address string                 `json:"inet6_address,omitempty"`
	Extra        map[string]interface{} `json:"-"`
}

// MarshalJSON flattens Extra fields
func (t TUNConfig) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	if t.Name != "" {
		m["name"] = t.Name
	}
	if t.MTU > 0 {
		m["mtu"] = t.MTU
	}
	if t.Stack != "" {
		m["stack"] = t.Stack
	}
	if t.Inet4Address != "" {
		m["inet4_address"] = t.Inet4Address
	}
	if t.Inet6Address != "" {
		m["inet6_address"] = t.Inet6Address
	}
	for k, v := range t.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// XHTTPConfig represents Xray XHTTP/splithttp transport settings
type XHTTPConfig struct {
	Path  string                 `json:"path,omitempty"`
	Host  string                 `json:"host,omitempty"`
	Mode  string                 `json:"mode,omitempty"` // "auto", "packet-up", "stream-up"
	Extra map[string]interface{} `json:"-"`
}

// MarshalJSON flattens Extra fields
func (x XHTTPConfig) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if x.Path != "" {
		m["path"] = x.Path
	}
	if x.Host != "" {
		m["host"] = x.Host
	}
	if x.Mode != "" {
		m["mode"] = x.Mode
	}
	for k, v := range x.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// FinalmaskConfig represents Xray Finalmask obfuscation layer (v26+)
type FinalmaskConfig struct {
	QUICParams *QUICParamsConfig      `json:"quicParams,omitempty"`
	Extra      map[string]interface{} `json:"-"`
}

// MarshalJSON flattens Extra fields
func (f FinalmaskConfig) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if f.QUICParams != nil {
		m["quicParams"] = f.QUICParams
	}
	for k, v := range f.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// QUICParamsConfig represents QUIC congestion control parameters
type QUICParamsConfig struct {
	Congestion  string `json:"congestion,omitempty"`   // "bbr", "cubic", "new_reno"
	BrutalUp    string `json:"brutal_up,omitempty"`    // upload bandwidth e.g. "100 mbps"
	BrutalDown  string `json:"brutal_down,omitempty"`  // download bandwidth
	ForceBrutal bool   `json:"force_brutal,omitempty"` // force brutal congestion
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

	// Generate random API port if not set (for Xray cores only)
	apiPort := core.APIPort
	if core.Name == "xray" && apiPort == 0 || apiPort == 10085 {
		randomPort, err := generateRandomPort()
		if err != nil {
			logger.Log.Warn().Err(err).Msg("Failed to generate random API port, using default 10085")
			apiPort = 10085
		} else {
			apiPort = randomPort
			core.APIPort = apiPort
			if err := db.Save(&core).Error; err != nil {
				logger.Log.Warn().Err(err).Int("port", apiPort).Msg("Failed to save API port to database")
			}
		}
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
	logDir := "/var/log/supervisor"
	if ctx.CoreConfig != nil && ctx.CoreConfig.LogDirectory != "" {
		logDir = ctx.CoreConfig.LogDirectory
	}
	config := &Config{
		Log: &LogConfig{
			Access:   filepath.Join(logDir, "xray_access.log"),
			Error:    filepath.Join(logDir, "xray_error.log"),
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
		Port:     apiPort,
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
		tag, protocol, settings, err := cores.XrayWARPOutbound(warpData.Account)
		if err != nil {
			return nil, fmt.Errorf("failed to generate WARP outbound: %w", err)
		}
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
	settings, err := buildInboundSettings(inbound.Protocol, []byte(inbound.ConfigJSON), users, inbound.RealityEnabled)
	if err != nil {
		return nil, err
	}

	// Generate tag from inbound ID and protocol
	tag := fmt.Sprintf("%s_%d", inbound.Protocol, inbound.ID)

	// Parse ConfigJSON early so it can be used throughout the function
	var cfgSettings map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse ConfigJSON")
			cfgSettings = make(map[string]interface{})
		}
	}
	if cfgSettings == nil {
		cfgSettings = make(map[string]interface{})
	}

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
			Network:   "tcp",
			Security:  "tls",
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

		// Xray v26+ defaults echForceQuery to "full" which breaks non-ECH clients.
		// Explicitly set to "off" unless user specifies otherwise.
		if echFQ, ok := cfgSettings["ech_force_query"].(string); ok {
			streamConfig.TLSConfig.ECHForceQuery = echFQ
		} else {
			streamConfig.TLSConfig.ECHForceQuery = "off"
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
	if len(cfgSettings) > 0 {
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
				xhttpConfig := &XHTTPConfig{
					Path: "/xhttp",
				}
				if p, ok := cfgSettings["xhttp_path"].(string); ok && p != "" {
					xhttpConfig.Path = p
				}
				if h, ok := cfgSettings["xhttp_host"].(string); ok && h != "" {
					xhttpConfig.Host = h
				}
				if m, ok := cfgSettings["xhttp_mode"].(string); ok && m != "" {
					xhttpConfig.Mode = m
				}
				config.StreamSettings.XHTTPConfig = xhttpConfig
			}
		}

		// Apply Finalmask settings if enabled
		if finalmaskEnabled, ok := cfgSettings["finalmask_enabled"].(bool); ok && finalmaskEnabled {
			if config.StreamSettings == nil {
				config.StreamSettings = &StreamConfig{Network: "tcp"}
			}
			finalmaskConfig := &FinalmaskConfig{
				QUICParams: &QUICParamsConfig{},
			}
			if congestion, ok := cfgSettings["finalmask_congestion"].(string); ok && congestion != "" {
				finalmaskConfig.QUICParams.Congestion = congestion
			}
			if brutalUp, ok := cfgSettings["finalmask_brutal_up"].(string); ok && brutalUp != "" {
				finalmaskConfig.QUICParams.BrutalUp = brutalUp
			}
			if brutalDown, ok := cfgSettings["finalmask_brutal_down"].(string); ok && brutalDown != "" {
				finalmaskConfig.QUICParams.BrutalDown = brutalDown
			}
			config.StreamSettings.Finalmask = finalmaskConfig
		}
	}

	// Add sniffing for most protocols
	if inbound.Protocol != "http" && inbound.Protocol != "socks" {
		config.Sniffing = &SniffingConfig{
			Enabled:      true,
			DestOverride: []string{"http", "tls"},
		}
	}

	// TUN doesn't use stream settings (it's a Layer 3 tunnel)
	if inbound.Protocol == "tun" {
		config.StreamSettings = nil
	}

	return config, nil
}

// buildInboundSettings builds protocol-specific settings
func buildInboundSettings(protocol string, baseSettings json.RawMessage, users []models.User, realityEnabled bool) (json.RawMessage, error) {
	var settings map[string]interface{}
	if len(baseSettings) > 0 {
		if err := json.Unmarshal(baseSettings, &settings); err != nil {
			return nil, err
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Handle TUN which doesn't use clients
	if protocol == "tun" {
		return buildTUNSettings(settings), nil
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
	clients := buildClients(protocol, users, realityEnabled)
	if len(clients) > 0 {
		settings["clients"] = clients
	}

	// Handle Shadowsocks SS2022 ciphers - method at inbound level, not client level
	if protocol == "shadowsocks" {
		if method, ok := settings["method"].(string); ok && strings.HasPrefix(method, "2022-blake3") {
			settings["method"] = method
			// SS2022 key length must match cipher: 16 bytes for aes-128-gcm, 32 for aes-256-gcm/chacha20-poly1305
			keyLen := 32
			if strings.Contains(method, "aes-128") {
				keyLen = 16
			}
			key := make([]byte, keyLen)
			if _, err := rand.Read(key); err == nil {
				settings["password"] = base64.StdEncoding.EncodeToString(key)
			}
			// Remove method from individual clients (SS2022 forbids it)
			if clientList, ok := settings["clients"].([]map[string]interface{}); ok {
				for i := range clientList {
					delete(clientList[i], "method")
					delete(clientList[i], "level")
				}
			}
		}
	}

	// Handle hysteria2 QUIC params - move to finalmask.quicParams (Xray v26+)
	if protocol == "hysteria2" {
		if congestion, ok := settings["congestion"]; ok {
			// Create finalmask structure
			if _, fmOk := settings["finalmask"]; !fmOk {
				settings["finalmask"] = make(map[string]interface{})
			}
			fm := settings["finalmask"].(map[string]interface{})
			if _, qpOk := fm["quicParams"]; !qpOk {
				fm["quicParams"] = make(map[string]interface{})
			}
			qp := fm["quicParams"].(map[string]interface{})
			qp["congestion"] = congestion
			delete(settings, "congestion")

			if brutalUp, ok := settings["brutal_up"]; ok {
				qp["brutal_up"] = brutalUp
				delete(settings, "brutal_up")
			}
			if brutalDown, ok := settings["brutal_down"]; ok {
				qp["brutal_down"] = brutalDown
				delete(settings, "brutal_down")
			}
			if forceBrutal, ok := settings["force_brutal"]; ok {
				qp["force_brutal"] = forceBrutal
				delete(settings, "force_brutal")
			}
		}
	}

	// Handle Finalmask settings for other protocols (vmess, vless, trojan)
	if finalmaskEnabled, ok := settings["finalmask_enabled"].(bool); ok && finalmaskEnabled {
		// Create finalmask structure
		if _, fmOk := settings["finalmask"]; !fmOk {
			settings["finalmask"] = make(map[string]interface{})
		}
		fm := settings["finalmask"].(map[string]interface{})
		if _, qpOk := fm["quicParams"]; !qpOk {
			fm["quicParams"] = make(map[string]interface{})
		}
		qp := fm["quicParams"].(map[string]interface{})

		if congestion, ok := settings["finalmask_congestion"].(string); ok && congestion != "" {
			qp["congestion"] = congestion
		}
		if brutalUp, ok := settings["finalmask_brutal_up"].(string); ok && brutalUp != "" {
			qp["brutal_up"] = brutalUp
		}
		if brutalDown, ok := settings["finalmask_brutal_down"].(string); ok && brutalDown != "" {
			qp["brutal_down"] = brutalDown
		}

		// Remove the finalmask_* keys from settings
		delete(settings, "finalmask_enabled")
		delete(settings, "finalmask_congestion")
		delete(settings, "finalmask_brutal_up")
		delete(settings, "finalmask_brutal_down")
	}

	return json.Marshal(settings)
}

// buildTUNSettings builds TUN inbound settings
func buildTUNSettings(cfgSettings map[string]interface{}) json.RawMessage {
	settings := map[string]interface{}{
		"name":          "tun0",
		"mtu":           1500,
		"stack":         "system",
		"inet4_address": "10.0.0.1/24",
	}
	// Override with user settings
	for k, v := range cfgSettings {
		// Skip transport-related keys that don't apply to TUN
		if k == "transport" || k == "ws_path" || k == "ws_host" || k == "grpc_service_name" {
			continue
		}
		// Map interface_name to name for Xray config
		if k == "interface_name" {
			settings["name"] = v
		} else {
			settings[k] = v
		}
	}
	data, _ := json.Marshal(settings)
	return data
}

// buildClients builds client list for the protocol
func buildClients(protocol string, users []models.User, realityEnabled bool) []map[string]interface{} {
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
			// VLESS with Vision flow when Reality is enabled (required for Xray v26+)
			flow := ""
			if realityEnabled {
				flow = "xtls-rprx-vision"
			}
			clients = append(clients, map[string]interface{}{
				"id":         user.UUID,
				"level":      0,
				"email":      fmt.Sprintf("user_%d", user.ID),
				"flow":       flow,
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
				"method":   "aes-128-gcm",
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

	// Check for duplicate tags and validate TLS certificate files
	tags := make(map[string]bool)
	for _, inbound := range config.Inbounds {
		if tags[inbound.Tag] {
			return fmt.Errorf("duplicate inbound tag: %s", inbound.Tag)
		}
		tags[inbound.Tag] = true

		// Validate TLS certificate files if TLS is enabled
		if inbound.StreamSettings != nil && inbound.StreamSettings.Security == "tls" && inbound.StreamSettings.TLSConfig != nil {
			if inbound.StreamSettings.TLSConfig.CertFile != "" {
				certInfo, err := os.Stat(inbound.StreamSettings.TLSConfig.CertFile)
				if os.IsNotExist(err) {
					return fmt.Errorf("TLS certificate file not found: %s", inbound.StreamSettings.TLSConfig.CertFile)
				}
				if err != nil {
					return fmt.Errorf("TLS certificate file check failed: %w", err)
				}
				if certInfo.Size() == 0 {
					return fmt.Errorf("TLS certificate file is empty: %s", inbound.StreamSettings.TLSConfig.CertFile)
				}
			}
			if inbound.StreamSettings.TLSConfig.KeyFile != "" {
				keyInfo, err := os.Stat(inbound.StreamSettings.TLSConfig.KeyFile)
				if os.IsNotExist(err) {
					return fmt.Errorf("TLS key file not found: %s", inbound.StreamSettings.TLSConfig.KeyFile)
				}
				if err != nil {
					return fmt.Errorf("TLS key file check failed: %w", err)
				}
				if keyInfo.Size() == 0 {
					return fmt.Errorf("TLS key file is empty: %s", inbound.StreamSettings.TLSConfig.KeyFile)
				}
			}
			if inbound.StreamSettings.TLSConfig.CertFile != "" && inbound.StreamSettings.TLSConfig.KeyFile != "" {
				if _, err := tls.LoadX509KeyPair(inbound.StreamSettings.TLSConfig.CertFile, inbound.StreamSettings.TLSConfig.KeyFile); err != nil {
					return fmt.Errorf("invalid TLS certificate/key pair: %w", err)
				}
			}
		}
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
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
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

// generateRandomPort generates a random port in the range 10000-65535
// and checks if it's available (not in use)
func generateRandomPort() (int, error) {
	const maxAttempts = 100

	for i := 0; i < maxAttempts; i++ {
		port := 10000 + int(randUint32()%55536)

		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to find available port after %d attempts", maxAttempts)
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// randUint32 generates a random uint32
func randUint32() uint32 {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}
