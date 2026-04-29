package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// LinkFunc is a function type for generating protocol-specific links
type LinkFunc func(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string

// V2RayGenerator generates V2Ray/auto-detect format subscriptions
type V2RayGenerator struct {
	panelURL string
	db       *gorm.DB
}

// NewV2RayGenerator creates a new V2Ray generator
func NewV2RayGenerator(panelURL string, db *gorm.DB) *V2RayGenerator {
	return &V2RayGenerator{panelURL: panelURL, db: db}
}

// Name returns the generator name
func (g *V2RayGenerator) Name() string {
	return "v2ray"
}

// Generate generates a V2Ray/auto-detect format subscription
func (g *V2RayGenerator) Generate(data *UserSubscriptionData) (string, error) {
	var links []string

	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	certs := LoadCertsByIDs(g.db, inbounds)

	for _, inbound := range inbounds {
		link := g.generateProxyLink(data.User, inbound, certs)
		if link != "" {
			links = append(links, link)
		}
	}

	result := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(result))

	return encoded, nil
}

// protocolLinks maps protocol names to their link generation functions
var protocolLinks = map[string]LinkFunc{
	"vless":       generateVLESSLink,
	"vmess":       generateVMessLink,
	"trojan":      generateTrojanLink,
	"anytls":      generateAnyTLSLink,
	"shadowsocks": generateSSLink,
	"hysteria2":   generateHysteria2Link,
	"tuic_v4":     generateTUICv4Link,
	"tuic_v5":     generateTUICv5Link,
	"tuic":        generateTUICv5Link,
	"naive":       generateNaiveLink,
	"naiveproxy":  generateNaiveLink,
	"xhttp":       generateXHTTPLink,
	"ssr":         generateSSRLink,
	"shadowsocksr": generateSSRLink,
	"http":        generateHTTPLink,
	"socks5":      generateSOCKS5Link,
	"mixed":       generateMixedLink,
	"snell":       generateSnellLink,
}

func (g *V2RayGenerator) generateProxyLink(user models.User, inbound models.Inbound, certs map[uint]*models.Certificate) string {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &config); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse inbound ConfigJSON for proxy link")
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	serverAddr := ResolveServerAddr(inbound, g.panelURL, certs)

	if linkFunc, ok := protocolLinks[inbound.Protocol]; ok {
		return linkFunc(user, inbound, serverAddr, config, certs)
	}

	return ""
}

func generateVLESSLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}

	AddTLSAndRealityParams(&params, inbound, certs)
	AddTransportParams(&params, config)

	if flow := GetStringOrDefault(config, "flow", ""); flow != "" {
		params.Set("flow", flow)
	}

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateVMessLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	transport := GetStringOrDefault(config, "transport", "tcp")
	tlsInfo := GetInboundTLSInfo(inbound, certs)

	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   ProxyRemark(inbound),
		"add":  server,
		"port": inbound.Port,
		"id":   user.UUID,
		"aid":  0,
		"net":  transport,
		"type": "none",
		"tls":  "",
	}

	if tlsInfo.IsTLS || inbound.RealityEnabled {
		vmessConfig["tls"] = "tls"
		if tlsInfo.SNI != "" {
			vmessConfig["sni"] = tlsInfo.SNI
		}
	}

	switch transport {
	case "websocket":
		if path, ok := config["ws_path"].(string); ok && path != "" {
			vmessConfig["path"] = path
		}
		if host, ok := config["ws_host"].(string); ok && host != "" {
			vmessConfig["host"] = host
		}
	case "grpc":
		if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
			vmessConfig["path"] = sn
		}
	case "http":
		if path, ok := config["h2_path"].(string); ok && path != "" {
			vmessConfig["path"] = path
		}
		if host, ok := config["h2_host"].(string); ok && host != "" {
			vmessConfig["host"] = host
		}
	case "httpupgrade":
		if path, ok := config["ws_path"].(string); ok && path != "" {
			vmessConfig["path"] = path
		}
		if host, ok := config["ws_host"].(string); ok && host != "" {
			vmessConfig["host"] = host
		}
	}

	jsonData, _ := json.Marshal(vmessConfig)
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData)
}

func generateTrojanLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}

	AddTLSAndRealityParams(&params, inbound, certs)
	AddTransportParams(&params, config)

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateAnyTLSLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}
	password := GetStringOrDefault(config, "password", user.UUID)
	AddTLSAndRealityParams(&params, inbound, certs)
	return fmt.Sprintf("anytls://%s@%s:%d?%s#%s",
		password, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateSSLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	method := GetStringOrDefault(config, "method", "aes-256-gcm")
	password := GetStringOrDefault(config, "password", user.UUID)
	userInfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", method, password)))
	return fmt.Sprintf("ss://%s@%s:%d#%s",
		userInfo, server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
}

func generateHysteria2Link(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("insecure", "1")

	tlsInfo := GetInboundTLSInfo(inbound, certs)
	if tlsInfo.SNI != "" {
		params.Set("sni", tlsInfo.SNI)
	}

	if obfsType := GetStringOrDefault(config, "obfs_type", ""); obfsType != "" {
		params.Set("obfs", obfsType)
		if obfsPass := GetStringOrDefault(config, "obfs_password", ""); obfsPass != "" {
			params.Set("obfs-password", obfsPass)
		}
	}

	return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateTUICv4Link(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("congestion_control", GetStringOrDefault(config, "congestion_control", "bbr"))
	params.Set("alpn", GetStringOrDefault(config, "alpn", "h3"))

	tlsInfo := GetInboundTLSInfo(inbound, certs)
	if tlsInfo.IsTLS {
		params.Set("allow_insecure", "1")
	}
	if tlsInfo.SNI != "" {
		params.Set("sni", tlsInfo.SNI)
	}

	token := user.UUID
	if user.Token != nil && *user.Token != "" {
		token = *user.Token
	}
	return fmt.Sprintf("tuic://%s@%s:%d?%s#%s",
		token, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateTUICv5Link(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("congestion_control", GetStringOrDefault(config, "congestion_control", "bbr"))
	params.Set("alpn", GetStringOrDefault(config, "alpn", "h3"))

	tlsInfo := GetInboundTLSInfo(inbound, certs)
	if tlsInfo.IsTLS {
		params.Set("allow_insecure", "1")
	}
	if tlsInfo.SNI != "" {
		params.Set("sni", tlsInfo.SNI)
	}

	password := user.UUID
	if user.Token != nil && *user.Token != "" {
		password = *user.Token
	}
	return fmt.Sprintf("tuic://%s:%s@%s:%d?%s#%s",
		user.UUID, password, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateNaiveLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	return fmt.Sprintf("naive+https://%s:%s@%s:%d#%s",
		url.PathEscape(user.Username), url.PathEscape(user.UUID),
		server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
}

func generateXHTTPLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("type", "xhttp")

	AddTLSAndRealityParams(&params, inbound, certs)

	if path, ok := config["path"].(string); ok && path != "" {
		params.Set("path", path)
	}
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}

func generateSSRLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	method := GetStringOrDefault(config, "method", "chacha20-poly1305")
	protocol := GetStringOrDefault(config, "protocol", "origin")
	obfs := GetStringOrDefault(config, "obfs", "plain")
	passwordB64 := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(user.UUID))
	raw := fmt.Sprintf("%s:%d:%s:%s:%s:%s",
		server, inbound.Port, protocol, method, obfs, passwordB64)
	return "ssr://" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(raw))
}

func generateHTTPLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	if user.Username != "" {
		return fmt.Sprintf("http://%s:%s@%s:%d#%s",
			url.PathEscape(user.Username), url.PathEscape(user.UUID),
			server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
	}
	return fmt.Sprintf("http://%s:%d#%s", server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
}

func generateSOCKS5Link(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	if user.Username != "" {
		return fmt.Sprintf("socks5://%s:%s@%s:%d#%s",
			url.PathEscape(user.Username), url.PathEscape(user.UUID),
			server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
	}
	return fmt.Sprintf("socks5://%s:%d#%s", server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
}

func generateMixedLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	if user.Username != "" {
		return fmt.Sprintf("mixed://%s:%s@%s:%d#%s",
			url.PathEscape(user.Username), url.PathEscape(user.UUID),
			server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
	}
	return fmt.Sprintf("mixed://%s:%d#%s", server, inbound.Port, url.PathEscape(ProxyRemark(inbound)))
}

func generateSnellLink(user models.User, inbound models.Inbound, server string, config map[string]interface{}, certs map[uint]*models.Certificate) string {
	params := url.Values{}
	psk := user.UUID
	if user.Token != nil && *user.Token != "" {
		psk = *user.Token
	}
	version := GetIntOrDefault(config, "version", 3)
	params.Set("version", fmt.Sprintf("%d", version))
	if obfs := GetStringOrDefault(config, "obfs", ""); obfs != "" {
		params.Set("obfs", obfs)
	}
	return fmt.Sprintf("snell://%s@%s:%d?%s#%s",
		psk, server, inbound.Port,
		params.Encode(), url.PathEscape(ProxyRemark(inbound)))
}
