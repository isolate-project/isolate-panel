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

	"github.com/isolate-project/isolate-panel/internal/cache"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// SubscriptionService generates subscription configs in 3 formats
type SubscriptionService struct {
	db       *gorm.DB
	panelURL string
	cache    *cache.Cache
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(db *gorm.DB, panelURL string, cacheManager ...*cache.CacheManager) *SubscriptionService {
	var subCache *cache.Cache
	if len(cacheManager) > 0 && cacheManager[0] != nil {
		subCache = cacheManager[0].GetSubscriptionCache()
	}
	if panelURL == "" {
		panelURL = "http://localhost:8080"
	}
	return &SubscriptionService{
		db:       db,
		panelURL: panelURL,
		cache:    subCache,
	}
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
	// Try cache first
	if cached, ok := s.GetCachedSubscription(data.User.ID, "v2ray"); ok {
		return cached, nil
	}

	var links []string

	for _, inbound := range data.Inbounds {
		link := s.generateProxyLink(data.User, inbound)
		if link != "" {
			links = append(links, link)
		}
	}

	result := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(result))

	// Cache the result
	s.SetCachedSubscription(data.User.ID, "v2ray", encoded)

	return encoded, nil
}

// GenerateClash generates Clash subscription format (YAML)
func (s *SubscriptionService) GenerateClash(data *UserSubscriptionData) (string, error) {
	// Try cache first
	if cached, ok := s.GetCachedSubscription(data.User.ID, "clash"); ok {
		return cached, nil
	}

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

	result := sb.String()

	// Cache the result
	s.SetCachedSubscription(data.User.ID, "clash", result)

	return result, nil
}

// GenerateSingbox generates Sing-box subscription format (JSON)
func (s *SubscriptionService) GenerateSingbox(data *UserSubscriptionData) (string, error) {
	// Try cache first
	if cached, ok := s.GetCachedSubscription(data.User.ID, "singbox"); ok {
		return cached, nil
	}

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

	result := string(jsonData)

	// Cache the result
	s.SetCachedSubscription(data.User.ID, "singbox", result)

	return result, nil
}

// GetCachedSubscription gets cached subscription content
func (s *SubscriptionService) GetCachedSubscription(userID uint, format string) (string, bool) {
	if s.cache == nil {
		return "", false
	}
	key := fmt.Sprintf("sub:%d:%s", userID, format)
	if cached, found := s.cache.GetString(key); found {
		return cached, true
	}
	return "", false
}

// SetCachedSubscription sets cached subscription content
func (s *SubscriptionService) SetCachedSubscription(userID uint, format string, content string) {
	if s.cache != nil {
		key := fmt.Sprintf("sub:%d:%s", userID, format)
		s.cache.Set(key, content)
	}
}

// InvalidateUserCache invalidates all cached subscriptions for a user
func (s *SubscriptionService) InvalidateUserCache(userID uint) {
	if s.cache != nil {
		formats := []string{"v2ray", "clash", "singbox"}
		for _, format := range formats {
			key := fmt.Sprintf("sub:%d:%s", userID, format)
			s.cache.Delete(key)
		}
	}
}

// LogAccess logs a subscription access
func (s *SubscriptionService) LogAccess(userID uint, ip, userAgent, format string, responseTimeMs int, isSuspicious bool) {
	access := &models.SubscriptionAccess{
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
func (s *SubscriptionService) GetOrCreateShortURL(userID uint, subscriptionToken string) (*models.SubscriptionShortURL, error) {
	var existing models.SubscriptionShortURL
	err := s.db.Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}

	// Generate short code
	code, err := generateShortCode(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate short code: %w", err)
	}

	shortURL := &models.SubscriptionShortURL{
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
func (s *SubscriptionService) ResolveShortURL(shortCode string) (*models.SubscriptionShortURL, error) {
	var shortURL models.SubscriptionShortURL
	if err := s.db.Where("short_code = ?", shortCode).First(&shortURL).Error; err != nil {
		return nil, fmt.Errorf("short URL not found")
	}
	return &shortURL, nil
}

// SubscriptionStats holds subscription access statistics
type SubscriptionStats struct {
	TotalAccesses int            `json:"total_accesses"`
	ByFormat      map[string]int `json:"by_format"`
	ByDay         map[string]int `json:"by_day"`
	UniqueIPs     int            `json:"unique_ips"`
	LastAccess    *time.Time     `json:"last_access"`
}

// GetAccessStats retrieves subscription access statistics for a user
func (s *SubscriptionService) GetAccessStats(userID uint, days int) (*SubscriptionStats, error) {
	var accesses []models.SubscriptionAccess
	since := time.Now().AddDate(0, 0, -days)

	err := s.db.Where("user_id = ? AND accessed_at > ?", userID, since).
		Order("accessed_at DESC").
		Find(&accesses).Error

	if err != nil {
		return nil, err
	}

	stats := &SubscriptionStats{
		TotalAccesses: len(accesses),
		ByFormat:      make(map[string]int),
		ByDay:         make(map[string]int),
	}

	uniqueIPs := make(map[string]bool)

	for _, access := range accesses {
		stats.ByFormat[access.Format]++
		day := access.AccessedAt.Format("2006-01-02")
		stats.ByDay[day]++
		uniqueIPs[access.IPAddress] = true

		if stats.LastAccess == nil || access.AccessedAt.After(*stats.LastAccess) {
			t := access.AccessedAt
			stats.LastAccess = &t
		}
	}

	stats.UniqueIPs = len(uniqueIPs)

	return stats, nil
}

// RegenerateToken generates a new subscription token for a user
func (s *SubscriptionService) RegenerateToken(userID uint) (string, error) {
	// Generate new token
	newToken := generateSubscriptionToken()

	// Update user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	user.SubscriptionToken = newToken

	if err := s.db.Save(&user).Error; err != nil {
		return "", fmt.Errorf("failed to update user token: %w", err)
	}

	// Delete old short URLs
	s.db.Where("user_id = ?", userID).Delete(&models.SubscriptionShortURL{})

	// Invalidate cache
	s.InvalidateUserCache(userID)

	return newToken, nil
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
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(inbound.Name))
}

func (s *SubscriptionService) generateSSLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	method := getStringOrDefault(config, "method", "aes-256-gcm")
	userInfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", method, user.UUID)))
	return fmt.Sprintf("ss://%s@%s:%d#%s",
		userInfo, server, inbound.Port, url.PathEscape(inbound.Name))
}

func (s *SubscriptionService) generateHysteria2Link(user models.User, inbound models.Inbound, _ map[string]interface{}, server string) string {
	return fmt.Sprintf("hysteria2://%s@%s:%d?insecure=1#%s",
		user.UUID, server, inbound.Port, url.PathEscape(inbound.Name))
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
			name, server, inbound.Port, user.UUID, server), name
	case "shadowsocks":
		return fmt.Sprintf("  - name: %s\n    type: ss\n    server: %s\n    port: %d\n    cipher: %s\n    password: %s\n",
			name, server, inbound.Port,
			getStringOrDefault(config, "method", "aes-256-gcm"),
			user.UUID), name
	case "hysteria2":
		return fmt.Sprintf("  - name: %s\n    type: hysteria2\n    server: %s\n    port: %d\n    password: %s\n    skip-cert-verify: true\n",
			name, server, inbound.Port, user.UUID), name
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
			"password":    user.UUID,
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
			"password":    user.UUID,
		}
	case "hysteria2":
		return map[string]interface{}{
			"type":        "hysteria2",
			"tag":         inbound.Name,
			"server":      server,
			"server_port": inbound.Port,
			"password":    user.UUID,
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
	// Each byte produces 2 hex chars, so we need ceil(length/2) bytes
	nBytes := (length + 1) / 2
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:length], nil
}

func generateSubscriptionToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}
