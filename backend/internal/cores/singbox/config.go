package singbox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

const MaxConfigJSONSize = 32 * 1024 // 32KB — limit inbound configuration JSON size

func validateConfigJSONSize(configJSON string) error {
	if len(configJSON) > MaxConfigJSONSize {
		return fmt.Errorf("config JSON exceeds maximum size (%d bytes)", MaxConfigJSONSize)
	}
	return nil
}

// Config represents the Sing-box configuration
type Config struct {
	Log          *LogConfig          `json:"log"`
	DNS          *DNSConfig          `json:"dns,omitempty"`
	NTP          *NTPConfig          `json:"ntp,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
	Endpoints    []EndpointConfig    `json:"endpoints,omitempty"`
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

// DNSServer represents a DNS server entry (v1.12+ format)
// Note: The old string-based "address" format (e.g., "https://dns.google/dns-query") is deprecated in v1.14.0.
// The new typed format uses "type" + "server" (e.g., {"type": "https", "server": "dns.google"}).
// For v1.13.x compatibility, we support both formats via MarshalJSON conversion.
type DNSServer struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`              // "https", "tls", "tcp", "udp", "local", "fakeip", "predefined"
	Server  string `json:"server,omitempty"`  // hostname/IP (no scheme prefix)
	Address string `json:"address,omitempty"` // DEPRECATED: for backward compatibility with v1.11

	// For type="fakeip": inet4_range and inet6_range at server level
	Inet4Range string `json:"inet4_range,omitempty"`
	Inet6Range string `json:"inet6_range,omitempty"`

	// Extra fields for future compatibility
	Extra map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to flatten Extra fields
// Handles backward compatibility: if Address is set but Type/Server are not, convert Address to Type/Server
func (d DNSServer) MarshalJSON() ([]byte, error) {
	// Handle backward compatibility: convert old Address format to new Type/Server format
	if d.Address != "" && d.Type == "" && d.Server == "" {
		// Parse old address format like "https://dns.google/dns-query" or "local"
		if d.Address == "local" {
			d.Type = "local"
		} else if len(d.Address) > 8 && d.Address[:8] == "https://" {
			d.Type = "https"
			// Extract hostname from URL (remove scheme and path)
			host := d.Address[8:]
			if idx := len(host); idx > 0 {
				// Find first slash to remove path
				if slashIdx := 0; slashIdx < len(host) && host[slashIdx] != '/' {
					for i := 0; i < len(host); i++ {
						if host[i] == '/' {
							host = host[:i]
							break
						}
					}
				}
			}
			d.Server = host
		} else if len(d.Address) > 4 && d.Address[:4] == "tls://" {
			d.Type = "tls"
			d.Server = d.Address[6:]
		} else if len(d.Address) > 6 && d.Address[:6] == "tcp://" {
			d.Type = "tcp"
			d.Server = d.Address[6:]
		} else if len(d.Address) > 6 && d.Address[:6] == "udp://" {
			d.Type = "udp"
			d.Server = d.Address[6:]
		} else {
			// Fallback: treat as plain address
			d.Type = "udp"
			d.Server = d.Address
		}
	}

	// Build a map with all standard fields
	m := make(map[string]interface{})
	m["tag"] = d.Tag
	if d.Type != "" {
		m["type"] = d.Type
	}
	if d.Server != "" {
		m["server"] = d.Server
	}
	if d.Inet4Range != "" {
		m["inet4_range"] = d.Inet4Range
	}
	if d.Inet6Range != "" {
		m["inet6_range"] = d.Inet6Range
	}

	// Merge extra fields
	for k, v := range d.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}

	return json.Marshal(m)
}

// DNSRule represents a DNS routing rule
type DNSRule struct {
	Outbound string `json:"outbound,omitempty"`
	Server   string `json:"server,omitempty"`
}

// NTPConfig represents NTP configuration (optional, for future use)
type NTPConfig struct {
	Enabled    bool   `json:"enabled"`
	Server     string `json:"server,omitempty"`
	ServerPort int    `json:"server_port,omitempty"`
	Interval   string `json:"interval,omitempty"`
}

// EndpointConfig represents a single endpoint configuration (v1.12+)
type EndpointConfig struct {
	Type  string                 `json:"type"`
	Tag   string                 `json:"tag"`
	Extra map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to flatten Extra fields
func (e EndpointConfig) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type": e.Type,
		"tag":  e.Tag,
	}
	for k, v := range e.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// InboundConfig represents a single inbound configuration
type InboundConfig struct {
	Type       string           `json:"type"`
	Tag        string           `json:"tag"`
	Listen     string           `json:"listen,omitempty"`
	ListenPort int              `json:"listen_port,omitempty"`
	Users      json.RawMessage  `json:"users,omitempty"`
	TLS        *TLSConfig       `json:"tls,omitempty"`
	Transport  *TransportConfig `json:"transport,omitempty"`
	Sniff      bool             `json:"sniff,omitempty"`

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
	Enabled         bool                 `json:"enabled"`
	ServerName      string               `json:"server_name,omitempty"`
	CertificatePath string               `json:"certificate_path,omitempty"`
	KeyPath         string               `json:"key_path,omitempty"`
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
	Type           string                 `json:"type"`
	Tag            string                 `json:"tag"`
	DomainResolver string                 `json:"domain_resolver,omitempty"`
	Extra          map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to flatten Extra fields
func (o OutboundConfig) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type": o.Type,
		"tag":  o.Tag,
	}
	if o.DomainResolver != "" {
		m["domain_resolver"] = o.DomainResolver
	}
	for k, v := range o.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// RouteConfig represents routing configuration
type RouteConfig struct {
	Rules               []RouteRule  `json:"rules,omitempty"`
	RuleSet             []RuleSetDef `json:"rule_set,omitempty"`
	Final               string       `json:"final,omitempty"`
	AutoDetectInterface bool         `json:"auto_detect_interface,omitempty"`
}

// RouteRule represents a single routing rule
type RouteRule struct {
	Action       string   `json:"action,omitempty"` // "route", "block", "dns", "hijack"
	Inbound      []string `json:"inbound,omitempty"`
	Outbound     string   `json:"outbound,omitempty"` // Only used when action is "route" or absent
	Protocol     []string `json:"protocol,omitempty"`
	DomainSuffix []string `json:"domain_suffix,omitempty"`
	IPCIDR       []string `json:"ip_cidr,omitempty"`
	RuleSet      []string `json:"rule_set,omitempty"` // for rule-set references
}

// RuleSetDef represents a rule-set definition (v1.12+ format)
type RuleSetDef struct {
	Tag            string `json:"tag"`
	Type           string `json:"type"`           // "remote" or "local"
	URL            string `json:"url,omitempty"`  // for remote
	Path           string `json:"path,omitempty"` // for local
	DownloadDetour string `json:"download_detour,omitempty"`
	UpdateInterval string `json:"update_interval,omitempty"` // e.g., "7d"
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

// AnyTLSUser represents an AnyTLS user
type AnyTLSUser struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// ============================================================
// Config Generation
// ============================================================

// GenerateConfig generates Sing-box configuration from database
func GenerateConfig(ctx *cores.ConfigContext, coreID uint) (*Config, error) {
	if ctx.CoreConfig == nil {
		ctx.CoreConfig = &cores.CoreConfig{}
		ctx.CoreConfig.ApplyDefaults()
	}
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

	// Get per-core API secret
	var apiSecret string
	if ctx.GetCoreAPISecret != nil {
		secret, err := ctx.GetCoreAPISecret(core.ID)
		if err != nil {
			// Log warning but continue with empty secret
			logger.Log.Warn().Err(err).Uint("core_id", core.ID).Msg("Failed to get API secret for singbox config")
		} else {
			apiSecret = secret
		}
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
				Secret:             apiSecret,
			},
		},
		DNS: &DNSConfig{
			Servers: []DNSServer{
				{Tag: "remote", Type: "https", Server: ctx.CoreConfig.DNSServer},
				{Tag: "local", Type: "local"},
			},
		},
		Inbounds:  make([]InboundConfig, 0),
		Outbounds: make([]OutboundConfig, 0),
		Route: &RouteConfig{
			Final:               "direct",
			AutoDetectInterface: true,
		},
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
		// For block/dns protocols, add route rule action instead of outbound
		if outbound.Protocol == "block" || outbound.Protocol == "dns" {
			rule := RouteRule{Action: outbound.Protocol}
			// Parse domain/IP filters from ConfigJSON if present
			if outbound.ConfigJSON != "" {
				if err := validateConfigJSONSize(outbound.ConfigJSON); err != nil {
					logger.Log.Warn().Err(err).Uint("outbound_id", outbound.ID).Msg("ConfigJSON size validation failed, skipping")
				} else {
					var cfg map[string]interface{}
					if json.Unmarshal([]byte(outbound.ConfigJSON), &cfg) == nil {
						if ds, ok := cfg["domain_suffix"].([]interface{}); ok {
							for _, d := range ds {
								rule.DomainSuffix = append(rule.DomainSuffix, fmt.Sprint(d))
							}
						}
						if ic, ok := cfg["ip_cidr"].([]interface{}); ok {
							for _, i := range ic {
								rule.IPCIDR = append(rule.IPCIDR, fmt.Sprint(i))
							}
						}
					}
				}
			}
			config.Route.Rules = append(config.Route.Rules, rule)
			continue
		}

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

	// Inject WARP WireGuard endpoint + route rules (v1.13+ format)
	if warpData, ok := cores.InjectWARP(ctx, coreID); ok {
		wgEndpoint := cores.SingboxWARPOutbound(warpData.Account)
		config.Endpoints = append(config.Endpoints, EndpointConfig{
			Type:  "wireguard",
			Tag:   "warp-out",
			Extra: wgEndpoint,
		})
		// Add WARP routing rules
		warpRules := cores.SingboxWARPRouteRules(warpData.Routes)
		for _, wr := range warpRules {
			rule := RouteRule{Action: "route", Outbound: "warp-out"}
			if ds, ok := wr["domain_suffix"].([]string); ok {
				rule.DomainSuffix = ds
			}
			if ic, ok := wr["ip_cidr"].([]string); ok {
				rule.IPCIDR = ic
			}
			config.Route.Rules = append(config.Route.Rules, rule)
		}
	}

	// Inject GeoIP/GeoSite routing rules (v1.12+ format)
	if geoData, ok := cores.InjectGeo(ctx, coreID); ok {
		geoRules := cores.SingboxGeoRouteRules(geoData.Rules, ctx.GeoDir)
		for _, gr := range geoRules {
			rule := RouteRule{}
			if action, ok := gr["action"].(string); ok {
				rule.Action = action
			}
			if outbound, ok := gr["outbound"].(string); ok {
				rule.Outbound = outbound
			}
			if ruleSet, ok := gr["rule_set"].([]string); ok {
				rule.RuleSet = ruleSet
			}
			config.Route.Rules = append(config.Route.Rules, rule)
		}
		// Add rule-set definitions
		geoRuleSets := cores.SingboxGeoRuleSets(geoData.Rules)
		for _, rs := range geoRuleSets {
			if tag, ok := rs["tag"].(string); ok {
				ruleType, _ := rs["type"].(string)
				url, _ := rs["url"].(string)
				downloadDetour, _ := rs["download_detour"].(string)
				updateInterval, _ := rs["update_interval"].(string)
				config.Route.RuleSet = append(config.Route.RuleSet, RuleSetDef{
					Tag:            tag,
					Type:           ruleType,
					URL:            url,
					DownloadDetour: downloadDetour,
					UpdateInterval: updateInterval,
				})
			}
		}
	}

	return config, nil
}

// convertInbound converts database inbound model to Sing-box inbound config
func convertInbound(db *gorm.DB, inbound models.Inbound, users []models.User) (*InboundConfig, error) {
	// Map protocol name to Sing-box inbound type
	singboxType := mapSingboxProtocol(inbound.Protocol)

	// Generate tag from inbound ID and protocol
	tag := fmt.Sprintf("%s_%d", inbound.Protocol, inbound.ID)

	config := &InboundConfig{
		Type:       singboxType,
		Tag:        tag,
		Listen:     inbound.ListenAddress,
		ListenPort: inbound.Port,
		Sniff:      true,
		Extra:      make(map[string]interface{}),
	}

	// Build users based on protocol
	usersJSON, err := buildUsers(inbound.Protocol, inbound.ConfigJSON, users, inbound.RealityEnabled)
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
		if err := validateConfigJSONSize(inbound.RealityConfigJSON); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("RealityConfigJSON size validation failed, skipping")
		} else {
			var realitySettings map[string]interface{}
			if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &realitySettings); err != nil {
				logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse RealityConfigJSON")
			} else {
				realityConfig := &RealityInlineConfig{
					Enabled: true,
				}

				// Parse handshake server (dest in Xray terms)
				serverPort := 443
				if sp, ok := realitySettings["serverPort"].(float64); ok && sp > 0 {
					serverPort = int(sp)
				}
				if dest, ok := realitySettings["dest"].(string); ok {
					realityConfig.Handshake = &RealityHandshake{
						Server:     dest,
						ServerPort: serverPort,
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
	}

	// Apply transport settings from ConfigJSON
	if inbound.ConfigJSON != "" {
		if err := validateConfigJSONSize(inbound.ConfigJSON); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("ConfigJSON size validation failed, skipping transport settings")
		} else {
			var cfgSettings map[string]interface{}
			if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfgSettings); err != nil {
				logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse ConfigJSON")
				cfgSettings = make(map[string]interface{})
			}
			if len(cfgSettings) > 0 {
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

					default:
						logger.Log.Warn().Str("transport", transport).Msg("Transport not supported by sing-box, skipping transport config")
					}
				}
			}
		}
	}

	return config, nil
}

// buildUsers builds user list for the protocol
func buildUsers(protocol string, configJSON string, users []models.User, realityEnabled bool) (json.RawMessage, error) {
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
			flow := ""
			if realityEnabled {
				flow = "xtls-rprx-vision"
			}
			vlessUsers = append(vlessUsers, VLESSUser{
				UUID: user.UUID,
				Flow: flow,
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
			password := user.UUID
			if user.Token != nil && *user.Token != "" {
				password = *user.Token
			}
			tuicUsers = append(tuicUsers, TUICUser{
				UUID:     user.UUID,
				Password: password,
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

	case "anytls":
		anytlsUsers := make([]AnyTLSUser, 0, len(users))
		for _, user := range users {
			anytlsUsers = append(anytlsUsers, AnyTLSUser{
				Name:     fmt.Sprintf("user_%d", user.ID),
				Password: user.UUID,
			})
		}
		return json.Marshal(anytlsUsers)

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

	if err := validateConfigJSONSize(configJSON); err != nil {
		logger.Log.Warn().Err(err).Str("protocol", protocol).Msg("ConfigJSON size validation failed, skipping protocol settings")
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
		if cc, ok := settings["congestion_control"].(string); ok && cc != "" {
			config.Extra["congestion_control"] = cc
		}
		if brutalMode, ok := settings["brutal_mode"].(bool); ok && brutalMode {
			brutalConfig := map[string]interface{}{
				"enabled": true,
			}
			if upMbps, ok := settings["up_mbps"]; ok {
				brutalConfig["send_mbps"] = upMbps
			}
			if downMbps, ok := settings["down_mbps"]; ok {
				brutalConfig["receive_mbps"] = downMbps
			}
			config.Extra["brutal"] = brutalConfig
		}

	case "tuic_v4", "tuic_v5":
		if cc, ok := settings["congestion_control"].(string); ok {
			config.Extra["congestion_control"] = cc
		}

	case "http", "socks5", "mixed":
		// Apply username/password if provided
		if username, ok := settings["username"].(string); ok && username != "" {
			password, _ := settings["password"].(string)
			usersList := []map[string]string{
				{"username": username, "password": password},
			}
			if data, err := json.Marshal(usersList); err == nil {
				config.Users = data
			}
		}
	}
}

// convertOutbound converts database outbound model to Sing-box outbound config
func convertOutbound(outbound models.Outbound) (*OutboundConfig, error) {
	singboxType, err := MapSingboxOutboundProtocol(outbound.Protocol)
	if err != nil {
		return nil, err
	}
	tag := fmt.Sprintf("%s_%d", outbound.Protocol, outbound.ID)

	extra := make(map[string]interface{})
	if outbound.ConfigJSON != "" {
		if err := validateConfigJSONSize(outbound.ConfigJSON); err != nil {
			logger.Log.Warn().Err(err).Uint("outbound_id", outbound.ID).Msg("ConfigJSON size validation failed, using empty extra")
		} else {
			if err := json.Unmarshal([]byte(outbound.ConfigJSON), &extra); err != nil {
				logger.Log.Warn().Err(err).Uint("outbound_id", outbound.ID).Msg("Failed to parse outbound ConfigJSON")
			}
		}
	}

	domainResolver := ""
	if dr, ok := extra["domain_resolver"].(string); ok {
		domainResolver = dr
		delete(extra, "domain_resolver")
	}

	return &OutboundConfig{
		Type:           singboxType,
		Tag:            tag,
		DomainResolver: domainResolver,
		Extra:          extra,
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
	case "anytls":
		return "anytls"
	case "redirect":
		return "redirect"
	default:
		return protocol
	}
}

// MapSingboxOutboundProtocol maps our protocol names to Sing-box outbound types
// Note: "block" and "dns" are rejected in v1.13+ - use route rule actions instead
func MapSingboxOutboundProtocol(protocol string) (string, error) {
	switch protocol {
	case "block":
		return "", fmt.Errorf("protocol 'block' must use route rule action instead of outbound type in sing-box v1.13+")
	case "dns":
		return "", fmt.Errorf("protocol 'dns' must use route rule action instead of outbound type in sing-box v1.13+")
	case "direct":
		return "direct", nil
	case "tor":
		return "tor", nil
	default:
		return protocol, nil
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

	// Check for duplicate tags and validate TLS certificate files
	tags := make(map[string]bool)
	for _, inbound := range config.Inbounds {
		if tags[inbound.Tag] {
			return fmt.Errorf("duplicate inbound tag: %s", inbound.Tag)
		}
		tags[inbound.Tag] = true

		// Validate TLS certificate files if TLS is enabled and not using Reality
		if inbound.TLS != nil && inbound.TLS.Enabled && inbound.TLS.Reality == nil {
			if inbound.TLS.CertificatePath != "" {
				certInfo, err := os.Stat(inbound.TLS.CertificatePath)
				if os.IsNotExist(err) {
					return fmt.Errorf("TLS certificate file not found: %s", inbound.TLS.CertificatePath)
				}
				if err != nil {
					return fmt.Errorf("TLS certificate file check failed: %w", err)
				}
				if certInfo.Size() == 0 {
					return fmt.Errorf("TLS certificate file is empty: %s", inbound.TLS.CertificatePath)
				}
			}
			if inbound.TLS.KeyPath != "" {
				keyInfo, err := os.Stat(inbound.TLS.KeyPath)
				if os.IsNotExist(err) {
					return fmt.Errorf("TLS key file not found: %s", inbound.TLS.KeyPath)
				}
				if err != nil {
					return fmt.Errorf("TLS key file check failed: %w", err)
				}
				if keyInfo.Size() == 0 {
					return fmt.Errorf("TLS key file is empty: %s", inbound.TLS.KeyPath)
				}
			}
			if inbound.TLS.CertificatePath != "" && inbound.TLS.KeyPath != "" {
				if _, err := tls.LoadX509KeyPair(inbound.TLS.CertificatePath, inbound.TLS.KeyPath); err != nil {
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
