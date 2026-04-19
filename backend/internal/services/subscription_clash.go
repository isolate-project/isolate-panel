package services

import (
	"fmt"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gopkg.in/yaml.v3"
)

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

type clashProxyGroup struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

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

type clashSnellObfsOpts struct {
	Mode string `yaml:"mode"`
}

func boolPtr(b bool) *bool { return &b }

func buildClashProxy(protocol, name, server string, port int, user models.User, config map[string]interface{}, tlsInfo inboundTLSInfo, certsByIDs map[uint]*models.Certificate) *clashProxy {
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
		p.Network = getStringOrDefault(config, "transport", "tcp")
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
		if p.Network == "ws" {
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
		} else if p.Network == "grpc" {
			grpcOpts := map[string]interface{}{}
			if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
				grpcOpts["grpc-service-name"] = sn
			}
			if len(grpcOpts) > 0 {
				extra["grpc-opts"] = grpcOpts
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
		p.Network = getStringOrDefault(config, "transport", "tcp")
		if *p.TLS && sni != server {
			p.ServerName = sni
		}
		extra := map[string]interface{}{}
		if p.Network == "ws" {
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
		} else if p.Network == "grpc" {
			grpcOpts := map[string]interface{}{}
			if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
				grpcOpts["grpc-service-name"] = sn
			}
			if len(grpcOpts) > 0 {
				extra["grpc-opts"] = grpcOpts
			}
		} else if p.Network == "h2" {
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
		p.Cipher = getStringOrDefault(config, "method", "aes-256-gcm")
		p.Password = user.UUID
	case "hysteria2":
		p.Type = "hysteria2"
		p.Password = user.UUID
		p.SkipCertVerify = boolPtr(false)
		if sni != server {
			p.SNI = sni
		}
	case "tuic_v4":
		p.Type = "tuic"
		token := user.UUID
		if user.Token != nil && *user.Token != "" {
			token = *user.Token
		}
		p.Token = token
		p.Version = 4
		p.CongestionController = getStringOrDefault(config, "congestion_control", "bbr")
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
		p.CongestionController = getStringOrDefault(config, "congestion_control", "bbr")
		p.SkipCertVerify = boolPtr(false)
	case "ssr", "shadowsocksr":
		p.Type = "ssr"
		p.Cipher = getStringOrDefault(config, "cipher", getStringOrDefault(config, "method", "chacha20-poly1305"))
		p.Password = user.UUID
		p.Protocol = getStringOrDefault(config, "protocol", "origin")
		p.Obfs = getStringOrDefault(config, "obfs", "plain")
	case "snell":
		p.Type = "snell"
		psk := user.UUID
		if user.Token != nil && *user.Token != "" {
			psk = *user.Token
		}
		p.PSK = psk
		p.Version = getIntOrDefault(config, "version", 3)
		p.ObfsOpts = &clashSnellObfsOpts{
			Mode: getStringOrDefault(config, "obfs", "tls"),
		}
	case "mieru":
		p.Type = "mieru"
		p.Password = user.UUID
	case "sudoku":
		p.Type = "sudoku"
		p.Password = getStringOrDefault(config, "password", user.UUID)
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
		p.Password = getStringOrDefault(config, "password", user.UUID)
		p.SkipCertVerify = boolPtr(false)
		if sni != server {
			p.SNI = sni
		}
	case "hysteria":
		p.Type = "hysteria"
		p.Password = getStringOrDefault(config, "auth_str", user.UUID)
		p.SkipCertVerify = boolPtr(false)
		if sni != server {
			p.SNI = sni
		}
	default:
		return nil
	}

	return p
}

func inbound2TLS(config map[string]interface{}, tlsInfo inboundTLSInfo) bool {
	return tlsInfo.SNI != "" || tlsInfo.IsTLS
}

func getIntOrDefault(config map[string]interface{}, key string, defaultVal int) int {
	// Snell version can be string or int
	if v, ok := config[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case string:
			var n int
			fmt.Sscanf(val, "%d", &n)
			if n > 0 {
				return n
			}
		}
	}
	return defaultVal
}

func marshalClashConfig(cfg clashConfig) (string, error) {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
