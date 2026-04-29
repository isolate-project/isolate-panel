package subscription

import (
	"encoding/json"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// InboundTLSInfo holds resolved TLS/SNI information for subscription link generation
type InboundTLSInfo struct {
	SNI   string // domain for SNI (from certificate or Reality)
	IsTLS bool   // whether TLS is enabled
}

// InboundRealityInfo holds Reality client parameters.
// Server stores privateKey; clients need publicKey + shortId + fingerprint.
type InboundRealityInfo struct {
	PublicKey   string // pbk in V2Ray links, public_key/publicKey in JSON
	ShortID     string // sid in V2Ray links, first shortId in JSON
	Fingerprint string // fp in V2Ray links, defaults to "chrome"
	SNI         string // masquerade domain from serverNames
}

// LoadCertsByIDs loads certificates by their IDs from the database
func LoadCertsByIDs(db *gorm.DB, inbounds []models.Inbound) map[uint]*models.Certificate {
	certIDs := make(map[uint]bool)
	for _, ib := range inbounds {
		if ib.TLSCertID != nil {
			certIDs[*ib.TLSCertID] = true
		}
	}
	if len(certIDs) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(certIDs))
	for id := range certIDs {
		ids = append(ids, id)
	}

	var certs []models.Certificate
	db.Where("id IN ?", ids).Find(&certs)

	result := make(map[uint]*models.Certificate, len(certs))
	for i := range certs {
		result[certs[i].ID] = &certs[i]
	}
	return result
}

// ResolveServerAddr determines the public server address for subscription links.
// Priority: 1) cert.Domain  2) panelURL hostname  3) "SERVER_IP" fallback
func ResolveServerAddr(inbound models.Inbound, panelURL string, certs map[uint]*models.Certificate) string {
	addr := inbound.ListenAddress
	if addr == "0.0.0.0" || addr == "" || addr == "::" {
		// 1. Domain from bound TLS certificate
		if inbound.TLSCertID != nil {
			if cert, ok := certs[*inbound.TLSCertID]; ok && cert.Domain != "" {
				return cert.Domain
			}
		}
		// 2. Hostname from panelURL
		if u, err := url.Parse(panelURL); err == nil && u.Hostname() != "" &&
			u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
			return u.Hostname()
		}
		// 3. Last resort fallback
		return "SERVER_IP"
	}
	return addr
}

// GetInboundTLSInfo resolves SNI for a given inbound:
//   - Reality: SNI from user-defined serverNames (masquerade domains)
//   - TLS: SNI from bound certificate's domain
func GetInboundTLSInfo(inbound models.Inbound, certsByIDs map[uint]*models.Certificate) InboundTLSInfo {
	info := InboundTLSInfo{IsTLS: inbound.TLSEnabled}

	// Reality — SNI from serverNames (user-configured masquerade domain)
	if inbound.RealityEnabled && inbound.RealityConfigJSON != "" {
		var realityCfg map[string]interface{}
		if json.Unmarshal([]byte(inbound.RealityConfigJSON), &realityCfg) == nil {
			if sns, ok := realityCfg["serverNames"].([]interface{}); ok && len(sns) > 0 {
				if sn, ok := sns[0].(string); ok {
					info.SNI = sn
				}
			}
		}
		info.IsTLS = true // Reality always implies TLS
		return info
	}

	// Standard TLS — SNI from bound certificate's domain
	if inbound.TLSEnabled && inbound.TLSCertID != nil {
		if cert, ok := certsByIDs[*inbound.TLSCertID]; ok && cert.Domain != "" {
			info.SNI = cert.Domain
		}
	}

	return info
}

// GetInboundRealityInfo extracts Reality client parameters from RealityConfigJSON.
// Returns nil if Reality is not enabled or config is missing/invalid.
func GetInboundRealityInfo(inbound models.Inbound) *InboundRealityInfo {
	if !inbound.RealityEnabled || inbound.RealityConfigJSON == "" {
		return nil
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &cfg); err != nil {
		return nil
	}

	info := &InboundRealityInfo{
		Fingerprint: "chrome",
	}

	if pbk, ok := cfg["public_key"].(string); ok && pbk != "" {
		info.PublicKey = pbk
	} else if pbk, ok := cfg["publicKey"].(string); ok && pbk != "" {
		info.PublicKey = pbk
	}

	if sids, ok := cfg["shortIds"].([]interface{}); ok && len(sids) > 0 {
		if sid, ok := sids[0].(string); ok {
			info.ShortID = sid
		}
	}

	if sns, ok := cfg["serverNames"].([]interface{}); ok && len(sns) > 0 {
		if sn, ok := sns[0].(string); ok {
			info.SNI = sn
		}
	}

	if fp, ok := cfg["fingerprint"].(string); ok && fp != "" {
		info.Fingerprint = fp
	}

	return info
}

// FormatCorePrefix returns a formatted prefix for the core name
func FormatCorePrefix(inbound models.Inbound) string {
	name := CoreNameForInbound(inbound)
	if name == "" {
		return ""
	}
	return "[" + name + "]"
}

// CoreNameForInbound returns the display name for an inbound's core.
func CoreNameForInbound(inbound models.Inbound) string {
	if inbound.Core == nil {
		return ""
	}
	switch inbound.Core.Name {
	case "xray":
		return "Xray"
	case "singbox":
		return "Sing-box"
	case "mihomo":
		return "Mihomo"
	default:
		name := inbound.Core.Name
		if len(name) > 0 {
			return strings.ToUpper(name[:1]) + name[1:]
		}
		return name
	}
}

// AddTransportParams adds transport parameters to URL values
func AddTransportParams(params *url.Values, config map[string]interface{}) {
	transport := GetStringOrDefault(config, "transport", "tcp")
	params.Set("type", transport)

	switch transport {
	case "websocket":
		if path, ok := config["ws_path"].(string); ok && path != "" {
			params.Set("path", path)
		}
		if host, ok := config["ws_host"].(string); ok && host != "" {
			params.Set("host", host)
		}
	case "grpc":
		if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
			params.Set("serviceName", sn)
		}
	case "http":
		if path, ok := config["h2_path"].(string); ok && path != "" {
			params.Set("path", path)
		}
		if host, ok := config["h2_host"].(string); ok && host != "" {
			params.Set("host", host)
		}
	case "httpupgrade":
		if path, ok := config["ws_path"].(string); ok && path != "" {
			params.Set("path", path)
		}
		if host, ok := config["ws_host"].(string); ok && host != "" {
			params.Set("host", host)
		}
	}
}

// AddTLSAndRealityParams adds TLS and Reality parameters to URL values
func AddTLSAndRealityParams(params *url.Values, inbound models.Inbound, certsByIDs map[uint]*models.Certificate) {
	tlsInfo := GetInboundTLSInfo(inbound, certsByIDs)

	if inbound.RealityEnabled {
		params.Set("security", "reality")
		realityInfo := GetInboundRealityInfo(inbound)
		if realityInfo != nil {
			if realityInfo.PublicKey != "" {
				params.Set("pbk", realityInfo.PublicKey)
			}
			if realityInfo.ShortID != "" {
				params.Set("sid", realityInfo.ShortID)
			}
			if realityInfo.Fingerprint != "" {
				params.Set("fp", realityInfo.Fingerprint)
			}
			if realityInfo.SNI != "" {
				params.Set("sni", realityInfo.SNI)
			}
		}
		return
	}

	if tlsInfo.IsTLS {
		params.Set("security", "tls")
		if tlsInfo.SNI != "" {
			params.Set("sni", tlsInfo.SNI)
		}
	}
}

// GetStringOrDefault retrieves a string value from config or returns default
func GetStringOrDefault(config map[string]interface{}, key, defaultVal string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

// GetIntOrDefault retrieves an int value from config or returns default
func GetIntOrDefault(config map[string]interface{}, key string, defaultVal int) int {
	if v, ok := config[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case string:
			var n int
			if _, err := url.Parse(val); err == nil {
				// Try to parse as int from string
				if n > 0 {
					return n
				}
			}
		}
	}
	return defaultVal
}

// ProxyRemark generates a proxy remark with core prefix
func ProxyRemark(inbound models.Inbound) string {
	return FormatCorePrefix(inbound) + inbound.Name
}
