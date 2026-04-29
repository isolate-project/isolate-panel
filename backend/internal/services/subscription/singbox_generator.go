package subscription

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// SingboxGenerator generates Sing-box JSON format subscriptions
type SingboxGenerator struct {
	panelURL string
	db       *gorm.DB
}

// NewSingboxGenerator creates a new Sing-box generator
func NewSingboxGenerator(panelURL string, db *gorm.DB) *SingboxGenerator {
	return &SingboxGenerator{panelURL: panelURL, db: db}
}

// Name returns the generator name
func (g *SingboxGenerator) Name() string {
	return "singbox"
}

// Generate generates a Sing-box JSON format subscription
func (g *SingboxGenerator) Generate(data *UserSubscriptionData) (string, error) {
	outbounds := []map[string]interface{}{}

	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	certs := LoadCertsByIDs(g.db, inbounds)

	for _, inbound := range inbounds {
		ob := g.generateSingboxOutbound(data.User, inbound, certs)
		if ob != nil {
			outbounds = append(outbounds, ob)
		}
	}

	selectorProxies := []string{}
	for _, ob := range outbounds {
		if tag, ok := ob["tag"].(string); ok {
			selectorProxies = append(selectorProxies, tag)
		}
	}
	selectorProxies = append(selectorProxies, "direct")

	allOutbounds := make([]map[string]interface{}, 0, 1+len(outbounds)+1)
	allOutbounds = append(allOutbounds, map[string]interface{}{
		"type":      "selector",
		"tag":       "proxy",
		"outbounds": selectorProxies,
	})
	allOutbounds = append(allOutbounds, outbounds...)
	allOutbounds = append(allOutbounds, map[string]interface{}{
		"type": "direct",
		"tag":  "direct",
	})

	config := map[string]interface{}{
		"outbounds": allOutbounds,
	}

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sing-box config: %w", err)
	}

	return string(jsonData), nil
}

func (g *SingboxGenerator) generateSingboxOutbound(user models.User, inbound models.Inbound, certsByIDs map[uint]*models.Certificate) map[string]interface{} {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &config); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse inbound ConfigJSON for Sing-box outbound")
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	server := ResolveServerAddr(inbound, g.panelURL, certsByIDs)
	tlsInfo := GetInboundTLSInfo(inbound, certsByIDs)
	transport := GetStringOrDefault(config, "transport", "tcp")
	tag := FormatCorePrefix(inbound) + inbound.Name

	buildTLSConfig := func() map[string]interface{} {
		tlsConfig := map[string]interface{}{
			"enabled":  true,
			"insecure": true,
		}
		if tlsInfo.SNI != "" {
			tlsConfig["server_name"] = tlsInfo.SNI
		}
		if inbound.RealityEnabled {
			realityInfo := GetInboundRealityInfo(inbound)
			if realityInfo != nil {
				reality := map[string]interface{}{
					"enabled": true,
				}
				if realityInfo.PublicKey != "" {
					reality["public_key"] = realityInfo.PublicKey
				}
				if realityInfo.ShortID != "" {
					reality["short_id"] = realityInfo.ShortID
				}
				tlsConfig["reality"] = reality
			}
		}
		return tlsConfig
	}

	buildTransport := func() map[string]interface{} {
		t := map[string]interface{}{"type": transport}
		switch transport {
		case "ws":
			if path, ok := config["ws_path"].(string); ok && path != "" {
				t["path"] = path
			}
			if host, ok := config["ws_host"].(string); ok && host != "" {
				t["headers"] = map[string]string{"Host": host}
			}
		case "grpc":
			if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
				t["service_name"] = sn
			}
		case "http":
			if host, ok := config["h2_host"].(string); ok && host != "" {
				t["host"] = host
			}
			if path, ok := config["h2_path"].(string); ok && path != "" {
				t["path"] = path
			}
		case "httpupgrade":
			if host, ok := config["ws_host"].(string); ok && host != "" {
				t["host"] = host
			}
			if path, ok := config["ws_path"].(string); ok && path != "" {
				t["path"] = path
			}
		}
		return t
	}

	switch inbound.Protocol {
	case "vless":
		ob := map[string]interface{}{
			"type":        "vless",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"uuid":        user.UUID,
		}
		if flow := GetStringOrDefault(config, "flow", ""); flow != "" {
			ob["flow"] = flow
		}
		if transport != "tcp" {
			ob["transport"] = buildTransport()
		}
		if tlsInfo.IsTLS || inbound.RealityEnabled {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "vmess":
		ob := map[string]interface{}{
			"type":        "vmess",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"uuid":        user.UUID,
			"alter_id":    0,
			"security":    "auto",
		}
		if transport != "tcp" {
			ob["transport"] = buildTransport()
		}
		if tlsInfo.IsTLS {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "trojan":
		ob := map[string]interface{}{
			"type":        "trojan",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"password":    user.UUID,
		}
		if transport != "tcp" {
			ob["transport"] = buildTransport()
		}
		if tlsInfo.IsTLS || inbound.RealityEnabled {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "shadowsocks":
		ob := map[string]interface{}{
			"type":        "shadowsocks",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"method":      GetStringOrDefault(config, "method", "aes-256-gcm"),
			"password":    GetStringOrDefault(config, "password", user.UUID),
		}
		return ob
	case "hysteria2":
		ob := map[string]interface{}{
			"type":        "hysteria2",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"password":    GetStringOrDefault(config, "password", user.UUID),
			"tls":         buildTLSConfig(),
		}
		if obfsType := GetStringOrDefault(config, "obfs_type", ""); obfsType != "" {
			ob["obfs"] = map[string]interface{}{
				"type":     obfsType,
				"password": GetStringOrDefault(config, "obfs_password", ""),
			}
		}
		return ob
	case "hysteria":
		ob := map[string]interface{}{
			"type":        "hysteria",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"password":    GetStringOrDefault(config, "auth_str", user.UUID),
			"tls":         buildTLSConfig(),
		}
		if upMbps := GetIntOrDefault(config, "up_mbps", 0); upMbps > 0 {
			ob["up_mbps"] = upMbps
		}
		if downMbps := GetIntOrDefault(config, "down_mbps", 0); downMbps > 0 {
			ob["down_mbps"] = downMbps
		}
		if obfs := GetStringOrDefault(config, "obfs", ""); obfs != "" {
			ob["obfs"] = obfs
		}
		return ob
	case "tuic_v4", "tuic_v5", "tuic":
		password := user.UUID
		if user.Token != nil && *user.Token != "" {
			password = *user.Token
		}
		ob := map[string]interface{}{
			"type":               "tuic",
			"tag":                tag,
			"server":             server,
			"server_port":        inbound.Port,
			"uuid":               user.UUID,
			"password":           password,
			"congestion_control": GetStringOrDefault(config, "congestion_control", "bbr"),
		}
		if tlsInfo.IsTLS {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "naive", "naiveproxy":
		ob := map[string]interface{}{
			"type":        "naive",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"username":    user.Username,
			"password":    user.UUID,
		}
		if tlsInfo.IsTLS {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "anytls":
		ob := map[string]interface{}{
			"type":        "anytls",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"password":    GetStringOrDefault(config, "password", user.UUID),
		}
		if tlsInfo.IsTLS || inbound.RealityEnabled {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "http":
		ob := map[string]interface{}{
			"type":        "http",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
		}
		if user.Username != "" {
			ob["username"] = user.Username
			ob["password"] = user.UUID
		}
		if tlsInfo.IsTLS {
			ob["tls"] = buildTLSConfig()
		}
		return ob
	case "socks5":
		ob := map[string]interface{}{
			"type":        "socks",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
		}
		if user.Username != "" {
			ob["username"] = user.Username
			ob["password"] = user.UUID
		}
		return ob
	case "mixed":
		ob := map[string]interface{}{
			"type":        "mixed",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
		}
		if user.Username != "" {
			ob["username"] = user.Username
			ob["password"] = user.UUID
		}
		return ob
	case "ssr", "snell", "mieru", "sudoku", "trusttunnel", "xhttp":
		return nil
	default:
		return nil
	}
}
