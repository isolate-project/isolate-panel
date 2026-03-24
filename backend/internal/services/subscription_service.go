package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// SubscriptionService generates subscription configs in 3 formats
type SubscriptionService struct {
	db       *gorm.DB
	panelURL string
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(db *gorm.DB, panelURL string) *SubscriptionService {
	if panelURL == "" {
		panelURL = "http://localhost:8080"
	}
	return &SubscriptionService{
		db:       db,
		panelURL: panelURL,
	}
}

// SubscriptionShortURL model for short URLs
type SubscriptionShortURL struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	ShortCode string    `gorm:"uniqueIndex;not null" json:"short_code"`
	FullURL   string    `gorm:"not null" json:"full_url"`
	CreatedAt time.Time `json:"created_at"`
}

func (SubscriptionShortURL) TableName() string {
	return "subscription_short_urls"
}

// SubscriptionAccess model for access logging
type SubscriptionAccess struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null" json:"user_id"`
	IPAddress      string    `gorm:"not null" json:"ip_address"`
	UserAgent      string    `json:"user_agent"`
	Country        string    `json:"country"`
	Format         string    `json:"format"`
	IsSuspicious   bool      `gorm:"default:false" json:"is_suspicious"`
	ResponseTimeMs int       `gorm:"default:0" json:"response_time_ms"`
	AccessedAt     time.Time `json:"accessed_at"`
}

func (SubscriptionAccess) TableName() string {
	return "subscription_accesses"
}

// UserSubscriptionData holds the data needed to generate a subscription
type UserSubscriptionData struct {
	User     models.User
	Inbounds []models.Inbound
}

// GetUserBySubscriptionToken retrieves a user by their subscription token
func (s *SubscriptionService) GetUserBySubscriptionToken(token string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("subscription_token = ? AND is_active = ?", token, true).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check expiry
	if user.ExpiryDate != nil && user.ExpiryDate.Before(time.Now()) {
		return nil, fmt.Errorf("user subscription has expired")
	}

	return &user, nil
}

// GetUserSubscriptionData retrieves all data needed for subscription generation
func (s *SubscriptionService) GetUserSubscriptionData(token string) (*UserSubscriptionData, error) {
	user, err := s.GetUserBySubscriptionToken(token)
	if err != nil {
		return nil, err
	}

	// Get user's assigned inbounds
	var mappings []models.UserInboundMapping
	if err := s.db.Where("user_id = ?", user.ID).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get user inbound mappings: %w", err)
	}

	if len(mappings) == 0 {
		return &UserSubscriptionData{User: *user, Inbounds: []models.Inbound{}}, nil
	}

	inboundIDs := make([]uint, len(mappings))
	for i, m := range mappings {
		inboundIDs[i] = m.InboundID
	}

	var inbounds []models.Inbound
	if err := s.db.Where("id IN ? AND is_enabled = ?", inboundIDs, true).Preload("Core").Find(&inbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get inbounds: %w", err)
	}

	return &UserSubscriptionData{User: *user, Inbounds: inbounds}, nil
}

// GenerateV2Ray generates V2Ray subscription format (base64-encoded link list)
func (s *SubscriptionService) GenerateV2Ray(data *UserSubscriptionData) (string, error) {
	var links []string

	for _, inbound := range data.Inbounds {
		link := s.generateProxyLink(data.User, inbound)
		if link != "" {
			links = append(links, link)
		}
	}

	result := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(result))
	return encoded, nil
}

// GenerateClash generates Clash subscription format (YAML)
func (s *SubscriptionService) GenerateClash(data *UserSubscriptionData) (string, error) {
	var proxies []string
	var proxyNames []string

	for _, inbound := range data.Inbounds {
		proxy, name := s.generateClashProxy(data.User, inbound)
		if proxy != "" {
			proxies = append(proxies, proxy)
			proxyNames = append(proxyNames, name)
		}
	}

	// Build YAML
	var sb strings.Builder
	sb.WriteString("port: 7890\n")
	sb.WriteString("socks-port: 7891\n")
	sb.WriteString("allow-lan: false\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n\n")

	sb.WriteString("proxies:\n")
	for _, proxy := range proxies {
		sb.WriteString(proxy)
	}

	sb.WriteString("\nproxy-groups:\n")
	sb.WriteString("  - name: PROXY\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	for _, name := range proxyNames {
		sb.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	sb.WriteString("\nrules:\n")
	sb.WriteString("  - MATCH,PROXY\n")

	return sb.String(), nil
}

// GenerateSingbox generates Sing-box subscription format (JSON)
func (s *SubscriptionService) GenerateSingbox(data *UserSubscriptionData) (string, error) {
	outbounds := []map[string]interface{}{}

	for _, inbound := range data.Inbounds {
		ob := s.generateSingboxOutbound(data.User, inbound)
		if ob != nil {
			outbounds = append(outbounds, ob)
		}
	}

	// Add selector and direct outbounds
	selectorProxies := []string{}
	for _, ob := range outbounds {
		if tag, ok := ob["tag"].(string); ok {
			selectorProxies = append(selectorProxies, tag)
		}
	}
	selectorProxies = append(selectorProxies, "direct")

	allOutbounds := []map[string]interface{}{
		{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": selectorProxies,
		},
	}
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

// LogAccess logs a subscription access
func (s *SubscriptionService) LogAccess(userID uint, ip, userAgent, format string, responseTimeMs int, isSuspicious bool) {
	access := &SubscriptionAccess{
		UserID:         userID,
		IPAddress:      ip,
		UserAgent:      userAgent,
		Format:         format,
		ResponseTimeMs: responseTimeMs,
		IsSuspicious:   isSuspicious,
		AccessedAt:     time.Now(),
	}
	s.db.Create(access)
}

// GetOrCreateShortURL gets or creates a short URL for a user
func (s *SubscriptionService) GetOrCreateShortURL(userID uint, subscriptionToken string) (*SubscriptionShortURL, error) {
	var existing SubscriptionShortURL
	err := s.db.Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}

	// Generate short code
	code, err := generateShortCode(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate short code: %w", err)
	}

	shortURL := &SubscriptionShortURL{
		UserID:    userID,
		ShortCode: code,
		FullURL:   fmt.Sprintf("/sub/%s", subscriptionToken),
	}

	if err := s.db.Create(shortURL).Error; err != nil {
		return nil, fmt.Errorf("failed to create short URL: %w", err)
	}

	return shortURL, nil
}

// ResolveShortURL resolves a short code to the full subscription URL
func (s *SubscriptionService) ResolveShortURL(shortCode string) (*SubscriptionShortURL, error) {
	var shortURL SubscriptionShortURL
	if err := s.db.Where("short_code = ?", shortCode).First(&shortURL).Error; err != nil {
		return nil, fmt.Errorf("short URL not found")
	}
	return &shortURL, nil
}

// generateProxyLink generates a proxy link for V2Ray format
func (s *SubscriptionService) generateProxyLink(user models.User, inbound models.Inbound) string {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		json.Unmarshal([]byte(inbound.ConfigJSON), &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	serverAddr := inbound.ListenAddress
	if serverAddr == "0.0.0.0" || serverAddr == "" {
		serverAddr = "SERVER_IP"
	}

	switch inbound.Protocol {
	case "vless":
		return s.generateVLESSLink(user, inbound, config, serverAddr)
	case "vmess":
		return s.generateVMessLink(user, inbound, config, serverAddr)
	case "trojan":
		return s.generateTrojanLink(user, inbound, config, serverAddr)
	case "shadowsocks":
		return s.generateSSLink(user, inbound, config, serverAddr)
	case "hysteria2":
		return s.generateHysteria2Link(user, inbound, config, serverAddr)
	default:
		return ""
	}
}

func (s *SubscriptionService) generateVLESSLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	params := url.Values{}
	if inbound.TLSEnabled {
		params.Set("security", "tls")
	}
	if inbound.RealityEnabled {
		params.Set("security", "reality")
	}
	params.Set("type", getStringOrDefault(config, "transport", "tcp"))

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(inbound.Name))
}

func (s *SubscriptionService) generateVMessLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   inbound.Name,
		"add":  server,
		"port": inbound.Port,
		"id":   user.UUID,
		"aid":  0,
		"net":  getStringOrDefault(config, "transport", "tcp"),
		"type": "none",
		"tls":  "",
	}
	if inbound.TLSEnabled {
		vmessConfig["tls"] = "tls"
	}

	jsonData, _ := json.Marshal(vmessConfig)
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData)
}

func (s *SubscriptionService) generateTrojanLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	params := url.Values{}
	if inbound.TLSEnabled {
		params.Set("security", "tls")
	}
	params.Set("type", getStringOrDefault(config, "transport", "tcp"))

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		user.Password, server, inbound.Port,
		params.Encode(), url.PathEscape(inbound.Name))
}

func (s *SubscriptionService) generateSSLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	method := getStringOrDefault(config, "method", "aes-256-gcm")
	userInfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", method, user.Password)))
	return fmt.Sprintf("ss://%s@%s:%d#%s",
		userInfo, server, inbound.Port, url.PathEscape(inbound.Name))
}

func (s *SubscriptionService) generateHysteria2Link(user models.User, inbound models.Inbound, _ map[string]interface{}, server string) string {
	return fmt.Sprintf("hysteria2://%s@%s:%d?insecure=1#%s",
		user.Password, server, inbound.Port, url.PathEscape(inbound.Name))
}

// generateClashProxy generates a Clash proxy entry
func (s *SubscriptionService) generateClashProxy(user models.User, inbound models.Inbound) (string, string) {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		json.Unmarshal([]byte(inbound.ConfigJSON), &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	server := inbound.ListenAddress
	if server == "0.0.0.0" || server == "" {
		server = "SERVER_IP"
	}

	name := inbound.Name

	switch inbound.Protocol {
	case "vless":
		return fmt.Sprintf("  - name: %s\n    type: vless\n    server: %s\n    port: %d\n    uuid: %s\n    tls: %t\n    skip-cert-verify: true\n    network: %s\n",
			name, server, inbound.Port, user.UUID, inbound.TLSEnabled,
			getStringOrDefault(config, "transport", "tcp")), name
	case "vmess":
		return fmt.Sprintf("  - name: %s\n    type: vmess\n    server: %s\n    port: %d\n    uuid: %s\n    alterId: 0\n    cipher: auto\n    tls: %t\n    skip-cert-verify: true\n    network: %s\n",
			name, server, inbound.Port, user.UUID, inbound.TLSEnabled,
			getStringOrDefault(config, "transport", "tcp")), name
	case "trojan":
		return fmt.Sprintf("  - name: %s\n    type: trojan\n    server: %s\n    port: %d\n    password: %s\n    sni: %s\n    skip-cert-verify: true\n",
			name, server, inbound.Port, user.Password, server), name
	case "shadowsocks":
		return fmt.Sprintf("  - name: %s\n    type: ss\n    server: %s\n    port: %d\n    cipher: %s\n    password: %s\n",
			name, server, inbound.Port,
			getStringOrDefault(config, "method", "aes-256-gcm"),
			user.Password), name
	case "hysteria2":
		return fmt.Sprintf("  - name: %s\n    type: hysteria2\n    server: %s\n    port: %d\n    password: %s\n    skip-cert-verify: true\n",
			name, server, inbound.Port, user.Password), name
	default:
		return "", ""
	}
}

// generateSingboxOutbound generates a Sing-box outbound entry
func (s *SubscriptionService) generateSingboxOutbound(user models.User, inbound models.Inbound) map[string]interface{} {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		json.Unmarshal([]byte(inbound.ConfigJSON), &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	server := inbound.ListenAddress
	if server == "0.0.0.0" || server == "" {
		server = "SERVER_IP"
	}

	switch inbound.Protocol {
	case "vless":
		ob := map[string]interface{}{
			"type":        "vless",
			"tag":         inbound.Name,
			"server":      server,
			"server_port": inbound.Port,
			"uuid":        user.UUID,
		}
		if inbound.TLSEnabled {
			ob["tls"] = map[string]interface{}{
				"enabled":  true,
				"insecure": true,
			}
		}
		return ob
	case "vmess":
		ob := map[string]interface{}{
			"type":        "vmess",
			"tag":         inbound.Name,
			"server":      server,
			"server_port": inbound.Port,
			"uuid":        user.UUID,
			"alter_id":    0,
			"security":    "auto",
		}
		if inbound.TLSEnabled {
			ob["tls"] = map[string]interface{}{
				"enabled":  true,
				"insecure": true,
			}
		}
		return ob
	case "trojan":
		ob := map[string]interface{}{
			"type":        "trojan",
			"tag":         inbound.Name,
			"server":      server,
			"server_port": inbound.Port,
			"password":    user.Password,
		}
		if inbound.TLSEnabled {
			ob["tls"] = map[string]interface{}{
				"enabled":     true,
				"insecure":    true,
				"server_name": server,
			}
		}
		return ob
	case "shadowsocks":
		return map[string]interface{}{
			"type":        "shadowsocks",
			"tag":         inbound.Name,
			"server":      server,
			"server_port": inbound.Port,
			"method":      getStringOrDefault(config, "method", "aes-256-gcm"),
			"password":    user.Password,
		}
	case "hysteria2":
		return map[string]interface{}{
			"type":        "hysteria2",
			"tag":         inbound.Name,
			"server":      server,
			"server_port": inbound.Port,
			"password":    user.Password,
			"tls": map[string]interface{}{
				"enabled":  true,
				"insecure": true,
			},
		}
	default:
		return nil
	}
}

// Helper functions

func getStringOrDefault(config map[string]interface{}, key, defaultVal string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func generateShortCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}
