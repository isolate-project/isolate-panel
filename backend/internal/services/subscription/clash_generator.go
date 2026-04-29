package subscription

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gopkg.in/yaml.v3"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// clashConfig represents a Clash configuration
type clashConfig struct {
	Port        int               `yaml:"port"`
	SocksPort   int               `yaml:"socks-port"`
	AllowLan    bool              `yaml:"allow-lan"`
	Mode        string            `yaml:"mode"`
	LogLevel    string            `yaml:"log-level"`
	Proxies     []clashProxy      `yaml:"proxies"`
	ProxyGroups []clashProxyGroup `yaml:"proxy-groups"`
	Rules       []string          `yaml:"rules"`
}

// clashProxyGroup represents a proxy group in Clash config
type clashProxyGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

// clashProxy represents a proxy in Clash config
type clashProxy struct {
	Type string `yaml:"type"`

	Name   string `yaml:"name"`
	Server string `yaml:"server"`
	Port   int    `yaml:"port"`

	// VLESS/VMess
	UUID       string `yaml:"uuid,omitempty"`
	AlterId    int    `yaml:"alterId,omitempty"`
	Cipher     string `yaml:"cipher,omitempty"`
	TLS        *bool  `yaml:"tls,omitempty"`
	Network    string `yaml:"network,omitempty"`
	ServerName string `yaml:"servername,omitempty"`

	// Trojan/Hysteria2/Sudoku/TrustTunnel/Mieru
	Password string `yaml:"password,omitempty"`

	// Trojan/Hysteria2
	SNI string `yaml:"sni,omitempty"`

	// Shadowsocks/SSR
	Protocol string `yaml:"protocol,omitempty"`
	Obfs     string `yaml:"obfs,omitempty"`

	// TUIC v4
	Token                string `yaml:"token,omitempty"`
	Version              int    `yaml:"version,omitempty"`
	CongestionController string `yaml:"congestion-controller,omitempty"`

	// TUIC v5
	// UUID is shared with VLESS/VMess
	// Password is shared with Trojan etc

	// Snell
	PSK      string              `yaml:"psk,omitempty"`
	ObfsOpts *clashSnellObfsOpts `yaml:"obfs-opts,omitempty"`

	// HTTP/Socks5/Mixed auth
	Username string `yaml:"username,omitempty"`

	// Common
	SkipCertVerify *bool `yaml:"skip-cert-verify,omitempty"`

	// Transport options
	Extra map[string]interface{} `yaml:",inline"`
}

// clashSnellObfsOpts represents obfs options for Snell
type clashSnellObfsOpts struct {
	Mode string `yaml:"mode"`
}

// ClashGenerator generates Clash YAML format subscriptions
type ClashGenerator struct {
	panelURL string
	db       *gorm.DB
}

// NewClashGenerator creates a new Clash generator
func NewClashGenerator(panelURL string, db *gorm.DB) *ClashGenerator {
	return &ClashGenerator{panelURL: panelURL, db: db}
}

// Name returns the generator name
func (g *ClashGenerator) Name() string {
	return "clash"
}

// Generate generates a Clash YAML format subscription
func (g *ClashGenerator) Generate(data *UserSubscriptionData) (string, error) {
	var proxies []clashProxy
	var proxyNames []string

	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	certs := LoadCertsByIDs(g.db, inbounds)

	for _, inbound := range inbounds {
		var config map[string]interface{}
		if inbound.ConfigJSON != "" {
			if err := json.Unmarshal([]byte(inbound.ConfigJSON), &config); err != nil {
				logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse inbound ConfigJSON for Clash proxy")
			}
		}
		if config == nil {
			config = make(map[string]interface{})
		}

		server := ResolveServerAddr(inbound, g.panelURL, certs)
		tlsInfo := GetInboundTLSInfo(inbound, certs)
		realityInfo := GetInboundRealityInfo(inbound)
		corePrefix := FormatCorePrefix(inbound)
		proxyName := corePrefix + inbound.Name

		proxy := buildClashProxy(inbound.Protocol, proxyName, server, inbound.Port, data.User, config, tlsInfo, certs, realityInfo)
		if proxy != nil {
			proxies = append(proxies, *proxy)
			proxyNames = append(proxyNames, proxyName)
		}
	}

	cfg := clashConfig{
		Port:        7890,
		SocksPort:   7891,
		AllowLan:    false,
		Mode:        "rule",
		LogLevel:    "info",
		Proxies:     proxies,
		ProxyGroups: []clashProxyGroup{{Name: "PROXY", Type: "select", Proxies: proxyNames}},
		Rules:       []string{"MATCH,PROXY"},
	}

	result, err := marshalClashConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Clash config: %w", err)
	}

	return result, nil
}

func boolPtr(b bool) *bool { return &b }

func buildClashProxy(protocol, name, server string, port int, user models.User, config map[string]interface{}, tlsInfo InboundTLSInfo, certsByIDs map[uint]*models.Certificate, realityInfo *InboundRealityInfo) *clashProxy {
	p := &clashProxy{
		Name:   name,
		Server: server,
		Port:   port,
	}
	sni := server
	if tlsInfo.SNI != "" {
		sni = tlsInfo.SNI
	}

	switch protocol {
	case "vless":
		p.Type = "vless"
		p.UUID = user.UUID
		p.TLS = boolPtr(inbound2TLS(config, tlsInfo))
		p.SkipCertVerify = boolPtr(false)
		p.Network = GetStringOrDefault(config, "transport", "tcp")
		if *p.TLS && sni != server {
			p.ServerName = sni
		}
		extra := map[string]interface{}{}
		if flow, ok := config["flow"].(string); ok && flow != "" {
			extra["flow"] = flow
		}
		if encryption, ok := config["encryption"].(string); ok && encryption != "" && encryption != "none" {
			extra["encryption"] = encryption
		}
		if realityInfo != nil {
			realityOpts := map[string]interface{}{}
			if realityInfo.PublicKey != "" {
				realityOpts["public-key"] = realityInfo.PublicKey
			}
			if realityInfo.ShortID != "" {
				realityOpts["short-id"] = realityInfo.ShortID
			}
			if len(realityOpts) > 0 {
				extra["reality-opts"] = realityOpts
			}
			extra["client-fingerprint"] = realityInfo.Fingerprint
		}
		switch p.Network {
		case "ws":
			wsOpts := map[string]interface{}{}
			if path, ok := config["ws_path"].(string); ok && path != "" {
				wsOpts["path"] = path
			}
			if host, ok := config["ws_host"].(string); ok && host != "" {
				wsOpts["headers"] = map[string]string{"Host": host}
			}
			if len(wsOpts) > 0 {
				extra["ws-opts"] = wsOpts
			}
		case "grpc":
			grpcOpts := map[string]interface{}{}
			if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
				grpcOpts["grpc-service-name"] = sn
			}
			if len(grpcOpts) > 0 {
				extra["grpc-opts"] = grpcOpts
			}
		case "httpupgrade":
			httpupgradeOpts := map[string]interface{}{}
			if host, ok := config["ws_host"].(string); ok && host != "" {
				httpupgradeOpts["headers"] = map[string]string{"Host": host}
			}
			if path, ok := config["ws_path"].(string); ok && path != "" {
				httpupgradeOpts["path"] = path
			}
			if len(httpupgradeOpts) > 0 {
				extra["http-opts"] = httpupgradeOpts
			}
		}
		if len(extra) > 0 {
			p.Extra = extra
		}
	case "vmess":
		p.Type = "vmess"
		p.UUID = user.UUID
		p.AlterId = 0
		p.Cipher = "auto"
		p.TLS = boolPtr(inbound2TLS(config, tlsInfo))
		p.SkipCertVerify = boolPtr(false)
		p.Network = GetStringOrDefault(config, "transport", "tcp")
		if *p.TLS && sni != server {
			p.ServerName = sni
		}
		extra := map[string]interface{}{}
		switch p.Network {
		case "ws":
			wsOpts := map[string]interface{}{}
			if path, ok := config["ws_path"].(string); ok && path != "" {
				wsOpts["path"] = path
			}
			if host, ok := config["ws_host"].(string); ok && host != "" {
				wsOpts["headers"] = map[string]string{"Host": host}
			}
			if len(wsOpts) > 0 {
				extra["ws-opts"] = wsOpts
			}
		case "grpc":
			grpcOpts := map[string]interface{}{}
			if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
				grpcOpts["grpc-service-name"] = sn
			}
			if len(grpcOpts) > 0 {
				extra["grpc-opts"] = grpcOpts
			}
		case "h2":
			h2Opts := map[string]interface{}{}
			if path, ok := config["h2_path"].(string); ok && path != "" {
				h2Opts["path"] = path
			}
			if host, ok := config["h2_host"].(string); ok && host != "" {
				h2Opts["host"] = []string{host}
			}
			if len(h2Opts) > 0 {
				extra["h2-opts"] = h2Opts
			}
		case "httpupgrade":
			httpupgradeOpts := map[string]interface{}{}
			if host, ok := config["ws_host"].(string); ok && host != "" {
				httpupgradeOpts["headers"] = map[string]string{"Host": host}
			}
			if path, ok := config["ws_path"].(string); ok && path != "" {
				httpupgradeOpts["path"] = path
			}
			if len(httpupgradeOpts) > 0 {
				extra["http-opts"] = httpupgradeOpts
			}
		}
		if len(extra) > 0 {
			p.Extra = extra
		}
	case "trojan":
		p.Type = "trojan"
		p.Password = user.UUID
		p.SNI = sni
		p.SkipCertVerify = boolPtr(false)
	case "shadowsocks":
		p.Type = "ss"
		p.Cipher = GetStringOrDefault(config, "method", "aes-256-gcm")
		p.Password = GetStringOrDefault(config, "password", user.UUID)
	case "hysteria2":
		p.Type = "hysteria2"
		p.Password = GetStringOrDefault(config, "password", user.UUID)
		p.SkipCertVerify = boolPtr(false)
		if sni != server {
			p.SNI = sni
		}
		extra := map[string]interface{}{}
		if obfsType := GetStringOrDefault(config, "obfs_type", ""); obfsType != "" {
			obfsOpts := map[string]interface{}{"type": obfsType}
			if obfsPass := GetStringOrDefault(config, "obfs_password", ""); obfsPass != "" {
				obfsOpts["password"] = obfsPass
			}
			extra["obfs"] = obfsOpts
		}
		if len(extra) > 0 {
			p.Extra = extra
		}
	case "tuic_v4":
		p.Type = "tuic"
		token := user.UUID
		if user.Token != nil && *user.Token != "" {
			token = *user.Token
		}
		p.Token = token
		p.Version = 4
		p.CongestionController = GetStringOrDefault(config, "congestion_control", "bbr")
		p.SkipCertVerify = boolPtr(false)
	case "tuic_v5", "tuic":
		p.Type = "tuic"
		password := user.UUID
		if user.Token != nil && *user.Token != "" {
			password = *user.Token
		}
		p.UUID = user.UUID
		p.Password = password
		p.Version = 5
		p.CongestionController = GetStringOrDefault(config, "congestion_control", "bbr")
		p.SkipCertVerify = boolPtr(false)
	case "ssr", "shadowsocksr":
		p.Type = "ssr"
		p.Cipher = GetStringOrDefault(config, "cipher", GetStringOrDefault(config, "method", "chacha20-poly1305"))
		p.Password = user.UUID
		p.Protocol = GetStringOrDefault(config, "protocol", "origin")
		p.Obfs = GetStringOrDefault(config, "obfs", "plain")
	case "snell":
		p.Type = "snell"
		psk := user.UUID
		if user.Token != nil && *user.Token != "" {
			psk = *user.Token
		}
		p.PSK = psk
		p.Version = GetIntOrDefault(config, "version", 3)
		p.ObfsOpts = &clashSnellObfsOpts{
			Mode: GetStringOrDefault(config, "obfs", "tls"),
		}
	case "mieru":
		p.Type = "mieru"
		p.Password = user.UUID
	case "sudoku":
		p.Type = "sudoku"
		p.Password = GetStringOrDefault(config, "password", user.UUID)
	case "trusttunnel":
		p.Type = "trusttunnel"
		p.Password = user.UUID
	case "http":
		p.Type = "http"
		if user.Username != "" {
			p.Username = user.Username
			p.Password = user.UUID
		}
		if inbound2TLS(config, tlsInfo) {
			p.TLS = boolPtr(true)
			p.SkipCertVerify = boolPtr(false)
		}
	case "socks5":
		p.Type = "socks5"
		if user.Username != "" {
			p.Username = user.Username
			p.Password = user.UUID
		}
		if inbound2TLS(config, tlsInfo) {
			p.TLS = boolPtr(true)
			p.SkipCertVerify = boolPtr(false)
		}
	case "mixed":
		p.Type = "mixed"
		if user.Username != "" {
			p.Username = user.Username
			p.Password = user.UUID
		}
	case "anytls":
		p.Type = "anytls"
		p.Password = GetStringOrDefault(config, "password", user.UUID)
		p.SkipCertVerify = boolPtr(false)
		if sni != server {
			p.SNI = sni
		}
	case "hysteria":
		p.Type = "hysteria"
		p.Password = GetStringOrDefault(config, "auth_str", user.UUID)
		p.SkipCertVerify = boolPtr(false)
		if sni != server {
			p.SNI = sni
		}
	default:
		return nil
	}

	return p
}

func inbound2TLS(config map[string]interface{}, tlsInfo InboundTLSInfo) bool {
	return tlsInfo.SNI != "" || tlsInfo.IsTLS
}

func marshalClashConfig(cfg clashConfig) (string, error) {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
