package subscription

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// IsolateSubscription represents the Isolate custom JSON subscription format
type IsolateSubscription struct {
	Version int                    `json:"version"`
	Profile IsolateProfile         `json:"profile"`
	Cores   map[string]IsolateCore `json:"cores"`
}

// IsolateProfile represents user profile information in Isolate format
type IsolateProfile struct {
	Username            string `json:"username"`
	UUID                string `json:"uuid"`
	TrafficUsed         int64  `json:"traffic_used"`
	TrafficLimit        *int64 `json:"traffic_limit,omitempty"`
	Expire              *int64 `json:"expire,omitempty"`
	UpdateIntervalHours int    `json:"update_interval_hours"`
	SubscriptionURL     string `json:"subscription_url"`
}

// IsolateCore represents a core with its inbounds in Isolate format
type IsolateCore struct {
	Inbounds []IsolateInbound `json:"inbounds"`
}

// IsolateInbound represents an inbound configuration in Isolate format
type IsolateInbound struct {
	ID        uint                   `json:"id"`
	Name      string                 `json:"name"`
	Protocol  string                 `json:"protocol"`
	Server    string                 `json:"server"`
	Port      int                    `json:"port"`
	UUID      string                 `json:"uuid,omitempty"`
	Password  string                 `json:"password,omitempty"`
	Method    string                 `json:"method,omitempty"`
	TLS       map[string]interface{} `json:"tls,omitempty"`
	Transport map[string]interface{} `json:"transport,omitempty"`
	RawLink   string                 `json:"raw_link"`
}

// IsolateGenerator generates Isolate custom JSON format subscriptions
type IsolateGenerator struct {
	panelURL string
	db       *gorm.DB
}

// NewIsolateGenerator creates a new Isolate generator
func NewIsolateGenerator(panelURL string, db *gorm.DB) *IsolateGenerator {
	return &IsolateGenerator{panelURL: panelURL, db: db}
}

// Name returns the generator name
func (g *IsolateGenerator) Name() string {
	return "isolate"
}

// Generate generates an Isolate custom JSON format subscription
func (g *IsolateGenerator) Generate(data *UserSubscriptionData) (string, error) {
	cores := make(map[string]IsolateCore)
	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	certs := LoadCertsByIDs(g.db, inbounds)

	for _, inbound := range inbounds {
		coreName := CoreNameForInbound(inbound)
		if coreName == "" {
			continue
		}

		var config map[string]interface{}
		if inbound.ConfigJSON != "" {
			_ = json.Unmarshal([]byte(inbound.ConfigJSON), &config)
		}
		if config == nil {
			config = make(map[string]interface{})
		}

		server := ResolveServerAddr(inbound, g.panelURL, certs)
		tlsInfo := GetInboundTLSInfo(inbound, certs)
		realityInfo := GetInboundRealityInfo(inbound)
		transport := GetStringOrDefault(config, "transport", "tcp")

		tlsMap := buildIsolateTLS(inbound, tlsInfo, realityInfo)
		transportMap := buildIsolateTransport(transport, config, inbound)

		rawLink := g.generateProxyLink(data.User, inbound, certs)

		ib := IsolateInbound{
			ID:        inbound.ID,
			Name:      inbound.Name,
			Protocol:  inbound.Protocol,
			Server:    server,
			Port:      inbound.Port,
			TLS:       tlsMap,
			Transport: transportMap,
			RawLink:   rawLink,
		}

		switch inbound.Protocol {
		case "vless", "vmess":
			ib.UUID = data.User.UUID
			if flow := GetStringOrDefault(config, "flow", ""); flow != "" {
				if ib.Transport == nil {
					ib.Transport = make(map[string]interface{})
				}
				ib.Transport["flow"] = flow
			}
		case "trojan":
			ib.Password = data.User.UUID
		case "shadowsocks":
			ib.Method = GetStringOrDefault(config, "method", "aes-256-gcm")
			ib.Password = GetStringOrDefault(config, "password", data.User.UUID)
		case "hysteria2":
			ib.Password = GetStringOrDefault(config, "password", data.User.UUID)
		case "tuic_v4":
			token := data.User.UUID
			if data.User.Token != nil && *data.User.Token != "" {
				token = *data.User.Token
			}
			ib.UUID = data.User.UUID
			ib.Password = token
		case "tuic_v5", "tuic":
			password := data.User.UUID
			if data.User.Token != nil && *data.User.Token != "" {
				password = *data.User.Token
			}
			ib.UUID = data.User.UUID
			ib.Password = password
		case "http", "socks5", "mixed":
			if data.User.Username != "" {
				ib.UUID = data.User.Username
				ib.Password = data.User.UUID
			}
		case "naive", "naiveproxy":
			ib.UUID = data.User.Username
			ib.Password = data.User.UUID
		case "xhttp":
			ib.UUID = data.User.UUID
			rawLink := g.generateProxyLink(data.User, inbound, certs)
			ib.RawLink = rawLink
		case "ssr", "shadowsocksr":
			ib.Password = GetStringOrDefault(config, "password", data.User.UUID)
			ib.Method = GetStringOrDefault(config, "cipher", GetStringOrDefault(config, "method", "chacha20-poly1305"))
		case "snell":
			psk := data.User.UUID
			if data.User.Token != nil && *data.User.Token != "" {
				psk = *data.User.Token
			}
			ib.Password = psk
		case "mieru", "sudoku", "trusttunnel":
			ib.Password = GetStringOrDefault(config, "password", data.User.UUID)
		case "anytls":
			ib.Password = GetStringOrDefault(config, "password", data.User.UUID)
		case "hysteria":
			ib.Password = GetStringOrDefault(config, "auth_str", data.User.UUID)
		}

		core, exists := cores[coreName]
		if !exists {
			core = IsolateCore{Inbounds: []IsolateInbound{}}
		}
		core.Inbounds = append(core.Inbounds, ib)
		cores[coreName] = core
	}

	var expireUnix *int64
	if data.User.ExpiryDate != nil {
		v := data.User.ExpiryDate.Unix()
		expireUnix = &v
	}

	subURL := fmt.Sprintf("%s/sub/%s/isolate", g.panelURL, data.User.SubscriptionToken)

	sub := IsolateSubscription{
		Version: 1,
		Profile: IsolateProfile{
			Username:            data.User.Username,
			UUID:                data.User.UUID,
			TrafficUsed:         data.User.TrafficUsedBytes,
			TrafficLimit:        data.User.TrafficLimitBytes,
			Expire:              expireUnix,
			UpdateIntervalHours: 24,
			SubscriptionURL:     subURL,
		},
		Cores: cores,
	}

	jsonData, err := json.MarshalIndent(sub, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Isolate subscription: %w", err)
	}

	return string(jsonData), nil
}

func (g *IsolateGenerator) generateProxyLink(user models.User, inbound models.Inbound, certs map[uint]*models.Certificate) string {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &config); err != nil {
			return ""
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

func buildIsolateTLS(inbound models.Inbound, tlsInfo InboundTLSInfo, realityInfo *InboundRealityInfo) map[string]interface{} {
	if !tlsInfo.IsTLS && !inbound.RealityEnabled {
		return nil
	}

	tlsMap := map[string]interface{}{
		"enabled": true,
	}

	if tlsInfo.SNI != "" {
		tlsMap["sni"] = tlsInfo.SNI
	}

	if inbound.RealityEnabled && realityInfo != nil {
		tlsMap["reality"] = true
		if realityInfo.PublicKey != "" {
			tlsMap["public_key"] = realityInfo.PublicKey
		}
		if realityInfo.ShortID != "" {
			tlsMap["short_id"] = realityInfo.ShortID
		}
		if realityInfo.Fingerprint != "" {
			tlsMap["fingerprint"] = realityInfo.Fingerprint
		}
	}

	return tlsMap
}

func buildIsolateTransport(transport string, config map[string]interface{}, inbound models.Inbound) map[string]interface{} {
	if transport == "tcp" {
		flow := GetStringOrDefault(config, "flow", "")
		if flow == "" {
			return nil
		}
		return map[string]interface{}{"type": "tcp", "flow": flow}
	}

	t := map[string]interface{}{"type": transport}

	switch transport {
	case "websocket":
		if path, ok := config["ws_path"].(string); ok && path != "" {
			t["path"] = path
		}
		if host, ok := config["ws_host"].(string); ok && host != "" {
			t["host"] = host
		}
	case "grpc":
		if sn, ok := config["grpc_service_name"].(string); ok && sn != "" {
			t["service_name"] = sn
		}
	case "http":
		if path, ok := config["h2_path"].(string); ok && path != "" {
			t["path"] = path
		}
		if host, ok := config["h2_host"].(string); ok && host != "" {
			t["host"] = host
		}
	case "httpupgrade":
		if path, ok := config["ws_path"].(string); ok && path != "" {
			t["path"] = path
		}
		if host, ok := config["ws_host"].(string); ok && host != "" {
			t["host"] = host
		}
	}

	return t
}
