package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cache"
	"github.com/isolate-project/isolate-panel/internal/logger"
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

// SubscriptionFilter defines filtering criteria for subscription inbounds
type SubscriptionFilter struct {
	ProtocolRegex string // regex to match inbound.Protocol (e.g., "vless|vmess|trojan")
	CoreName      string // exact match on inbound.Core.Name (e.g., "xray", "singbox", "mihomo")
	CoreNameRegex string // regex to match inbound.Core.Name
	TagRegex      string // regex to match inbound.Name (e.g., "US*", ".*-ws")
}

// FilterInbounds filters inbounds based on the filter criteria
func (f *SubscriptionFilter) FilterInbounds(inbounds []models.Inbound) []models.Inbound {
	if f == nil || (f.ProtocolRegex == "" && f.CoreName == "" && f.CoreNameRegex == "" && f.TagRegex == "") {
		return inbounds
	}

	var result []models.Inbound
	for _, inbound := range inbounds {
		// Apply ProtocolRegex filter
		if f.ProtocolRegex != "" {
			re, err := regexp.Compile(f.ProtocolRegex)
			if err != nil {
				// Invalid regex, log warning and skip this filter
				logger.Log.Warn().Err(err).Str("regex", f.ProtocolRegex).Msg("Invalid protocol regex, skipping filter")
			} else if !re.MatchString(inbound.Protocol) {
				continue
			}
		}

		// Apply CoreName exact match filter
		if f.CoreName != "" {
			if inbound.Core == nil || inbound.Core.Name != f.CoreName {
				continue
			}
		}

		// Apply CoreNameRegex filter
		if f.CoreNameRegex != "" {
			if inbound.Core == nil {
				continue
			}
			re, err := regexp.Compile(f.CoreNameRegex)
			if err != nil {
				// Invalid regex, log warning and skip this filter
				logger.Log.Warn().Err(err).Str("regex", f.CoreNameRegex).Msg("Invalid core name regex, skipping filter")
			} else if !re.MatchString(inbound.Core.Name) {
				continue
			}
		}

		// Apply TagRegex filter
		if f.TagRegex != "" {
			re, err := regexp.Compile(f.TagRegex)
			if err != nil {
				// Invalid regex, log warning and skip this filter
				logger.Log.Warn().Err(err).Str("regex", f.TagRegex).Msg("Invalid tag regex, skipping filter")
			} else if !re.MatchString(inbound.Name) {
				continue
			}
		}

		result = append(result, inbound)
	}

	return result
}

func (f *SubscriptionFilter) Hash() string {
	if f == nil {
		return ""
	}
	h := fnv.New32a()
	fmt.Fprintf(h, "p:%s c:%s cr:%s t:%s", f.ProtocolRegex, f.CoreName, f.CoreNameRegex, f.TagRegex)
	return fmt.Sprintf("%08x", h.Sum32())
}

// UserSubscriptionData holds the data needed to generate a subscription
type UserSubscriptionData struct {
	User     models.User
	Inbounds []models.Inbound
	Filter   *SubscriptionFilter
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

func (s *SubscriptionService) GenerateV2Ray(data *UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "v2ray", filterHash); ok {
		return cached, nil
	}

	certsByIDs := s.loadCertsByIDs(data.Inbounds)
	var links []string

	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	for _, inbound := range inbounds {
		link := generateProxyLink(data.User, inbound, s.panelURL, certsByIDs)
		if link != "" {
			links = append(links, link)
		}
	}

	result := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(result))

	s.SetCachedSubscription(data.User.ID, "v2ray", filterHash, encoded)

	return encoded, nil
}

func (s *SubscriptionService) GenerateClash(data *UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "clash", filterHash); ok {
		return cached, nil
	}

	certsByIDs := s.loadCertsByIDs(data.Inbounds)
	var proxies []clashProxy
	var proxyNames []string

	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

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

		server := resolveServerAddr(inbound, s.panelURL, certsByIDs)
		tlsInfo := getInboundTLSInfo(inbound, certsByIDs)
		realityInfo := getInboundRealityInfo(inbound)
		corePrefix := formatCorePrefix(inbound)
		proxyName := corePrefix + inbound.Name

		proxy := buildClashProxy(inbound.Protocol, proxyName, server, inbound.Port, data.User, config, tlsInfo, certsByIDs, realityInfo)
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

	s.SetCachedSubscription(data.User.ID, "clash", filterHash, result)

	return result, nil
}

func (s *SubscriptionService) GenerateSingbox(data *UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "singbox", filterHash); ok {
		return cached, nil
	}

	certsByIDs := s.loadCertsByIDs(data.Inbounds)
	outbounds := []map[string]interface{}{}

	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	for _, inbound := range inbounds {
		ob := generateSingboxOutbound(data.User, inbound, s.panelURL, certsByIDs)
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

	result := string(jsonData)

	s.SetCachedSubscription(data.User.ID, "singbox", filterHash, result)

	return result, nil
}

type IsolateSubscription struct {
	Version int                    `json:"version"`
	Profile IsolateProfile         `json:"profile"`
	Cores   map[string]IsolateCore `json:"cores"`
}

type IsolateProfile struct {
	Username            string `json:"username"`
	UUID                string `json:"uuid"`
	TrafficUsed         int64  `json:"traffic_used"`
	TrafficLimit        *int64 `json:"traffic_limit,omitempty"`
	Expire              *int64 `json:"expire,omitempty"`
	UpdateIntervalHours int    `json:"update_interval_hours"`
	SubscriptionURL     string `json:"subscription_url"`
}

type IsolateCore struct {
	Inbounds []IsolateInbound `json:"inbounds"`
}

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

func (s *SubscriptionService) GenerateIsolate(data *UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "isolate", filterHash); ok {
		return cached, nil
	}

	certsByIDs := s.loadCertsByIDs(data.Inbounds)

	cores := make(map[string]IsolateCore)
	inbounds := data.Inbounds
	if data.Filter != nil {
		inbounds = data.Filter.FilterInbounds(data.Inbounds)
	}

	for _, inbound := range inbounds {
		coreName := coreNameForInbound(inbound)
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

		server := resolveServerAddr(inbound, s.panelURL, certsByIDs)
		tlsInfo := getInboundTLSInfo(inbound, certsByIDs)
		realityInfo := getInboundRealityInfo(inbound)
		transport := getStringOrDefault(config, "transport", "tcp")

		tlsMap := buildIsolateTLS(inbound, tlsInfo, realityInfo)
		transportMap := buildIsolateTransport(transport, config, inbound)

		rawLink := generateProxyLink(data.User, inbound, s.panelURL, certsByIDs)

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
			if flow := getStringOrDefault(config, "flow", ""); flow != "" {
				if ib.Transport == nil {
					ib.Transport = make(map[string]interface{})
				}
				ib.Transport["flow"] = flow
			}
		case "trojan":
			ib.Password = data.User.UUID
		case "shadowsocks":
			ib.Method = getStringOrDefault(config, "method", "aes-256-gcm")
			ib.Password = getStringOrDefault(config, "password", data.User.UUID)
		case "hysteria2":
			ib.Password = getStringOrDefault(config, "password", data.User.UUID)
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
			rawLink := generateProxyLink(data.User, inbound, s.panelURL, certsByIDs)
			ib.RawLink = rawLink
		case "ssr", "shadowsocksr":
			ib.Password = getStringOrDefault(config, "password", data.User.UUID)
			ib.Method = getStringOrDefault(config, "cipher", getStringOrDefault(config, "method", "chacha20-poly1305"))
		case "snell":
			psk := data.User.UUID
			if data.User.Token != nil && *data.User.Token != "" {
				psk = *data.User.Token
			}
			ib.Password = psk
		case "mieru", "sudoku", "trusttunnel":
			ib.Password = getStringOrDefault(config, "password", data.User.UUID)
		case "anytls":
			ib.Password = getStringOrDefault(config, "password", data.User.UUID)
		case "hysteria":
			ib.Password = getStringOrDefault(config, "auth_str", data.User.UUID)
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

	subURL := fmt.Sprintf("%s/sub/%s/isolate", s.panelURL, data.User.SubscriptionToken)

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

	result := string(jsonData)
	s.SetCachedSubscription(data.User.ID, "isolate", filterHash, result)

	return result, nil
}

func buildIsolateTLS(inbound models.Inbound, tlsInfo inboundTLSInfo, realityInfo *inboundRealityInfo) map[string]interface{} {
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
		flow := getStringOrDefault(config, "flow", "")
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

func (s *SubscriptionService) GetCachedSubscription(userID uint, format string, filterHash string) (string, bool) {
	if s.cache == nil {
		return "", false
	}
	key := fmt.Sprintf("sub:%d:%s:%s", userID, format, filterHash)
	if cached, found := s.cache.GetString(key); found {
		return cached, true
	}
	return "", false
}

// SetCachedSubscription sets cached subscription content
func (s *SubscriptionService) SetCachedSubscription(userID uint, format string, filterHash string, content string) {
	if s.cache != nil {
		key := fmt.Sprintf("sub:%d:%s:%s", userID, format, filterHash)
		s.cache.Set(key, content)
	}
}

// InvalidateUserCache invalidates all cached subscriptions for a user.
// Ristretto does not support prefix-based deletion, so filtered keys like
// sub:{uid}:{format}:{hash} cannot be individually targeted. Clearing the
// entire subscription cache is safe for a panel with limited concurrent users.
func (s *SubscriptionService) InvalidateUserCache(userID uint) {
	if s.cache != nil {
		s.cache.Clear()
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

// inboundTLSInfo holds resolved TLS/SNI information for subscription link generation
type inboundTLSInfo struct {
	SNI   string // domain for SNI (from certificate or Reality)
	IsTLS bool   // whether TLS is enabled
}

// inboundRealityInfo holds Reality client parameters.
// Server stores privateKey; clients need publicKey + shortId + fingerprint.
type inboundRealityInfo struct {
	PublicKey   string // pbk in V2Ray links, public_key/publicKey in JSON
	ShortID     string // sid in V2Ray links, first shortId in JSON
	Fingerprint string // fp in V2Ray links, defaults to "chrome"
	SNI         string // masquerade domain from serverNames
}

// getInboundTLSInfo resolves SNI for a given inbound:
//   - Reality: SNI from user-defined serverNames (masquerade domains)
//   - TLS: SNI from bound certificate's domain
func getInboundTLSInfo(inbound models.Inbound, certsByIDs map[uint]*models.Certificate) inboundTLSInfo {
	info := inboundTLSInfo{IsTLS: inbound.TLSEnabled}

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

func (s *SubscriptionService) getInboundTLSInfoMethod(inbound models.Inbound) inboundTLSInfo {
	certsByIDs := s.loadCertsByIDs([]models.Inbound{inbound})
	return getInboundTLSInfo(inbound, certsByIDs)
}

// getInboundRealityInfo extracts Reality client parameters from RealityConfigJSON.
// Returns nil if Reality is not enabled or config is missing/invalid.
func getInboundRealityInfo(inbound models.Inbound) *inboundRealityInfo {
	if !inbound.RealityEnabled || inbound.RealityConfigJSON == "" {
		return nil
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(inbound.RealityConfigJSON), &cfg); err != nil {
		return nil
	}

	info := &inboundRealityInfo{
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

// coreNameForInbound returns the display name for an inbound's core.
func coreNameForInbound(inbound models.Inbound) string {
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

func formatCorePrefix(inbound models.Inbound) string {
	name := coreNameForInbound(inbound)
	if name == "" {
		return ""
	}
	return "[" + name + "]"
}

// resolveServerAddr determines the public server address for subscription links.
// Priority: 1) cert.Domain  2) panelURL hostname  3) "SERVER_IP" fallback
func resolveServerAddr(inbound models.Inbound, panelURL string, certsByIDs map[uint]*models.Certificate) string {
	addr := inbound.ListenAddress
	if addr == "0.0.0.0" || addr == "" || addr == "::" {
		// 1. Domain from bound TLS certificate
		if inbound.TLSCertID != nil {
			if cert, ok := certsByIDs[*inbound.TLSCertID]; ok && cert.Domain != "" {
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

func (s *SubscriptionService) resolveServerAddrMethod(inbound models.Inbound) string {
	certsByIDs := s.loadCertsByIDs([]models.Inbound{inbound})
	return resolveServerAddr(inbound, s.panelURL, certsByIDs)
}

func (s *SubscriptionService) loadCertsByIDs(inbounds []models.Inbound) map[uint]*models.Certificate {
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
	s.db.Where("id IN ?", ids).Find(&certs)

	result := make(map[uint]*models.Certificate, len(certs))
	for i := range certs {
		result[certs[i].ID] = &certs[i]
	}
	return result
}

func generateProxyLink(user models.User, inbound models.Inbound, panelURL string, certsByIDs map[uint]*models.Certificate) string {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &config); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse inbound ConfigJSON for proxy link")
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	serverAddr := resolveServerAddr(inbound, panelURL, certsByIDs)

	switch inbound.Protocol {
	case "vless":
		return generateVLESSLink(user, inbound, config, serverAddr, certsByIDs)
	case "vmess":
		return generateVMessLink(user, inbound, config, serverAddr, certsByIDs)
	case "trojan":
		return generateTrojanLink(user, inbound, config, serverAddr, certsByIDs)
	case "anytls":
		return generateAnyTLSLink(user, inbound, config, serverAddr, certsByIDs)
	case "shadowsocks":
		return generateSSLink(user, inbound, config, serverAddr)
	case "hysteria2":
		return generateHysteria2Link(user, inbound, config, serverAddr, certsByIDs)
	case "tuic_v4":
		return generateTUICv4Link(user, inbound, config, serverAddr, certsByIDs)
	case "tuic_v5", "tuic":
		return generateTUICv5Link(user, inbound, config, serverAddr, certsByIDs)
	case "naive", "naiveproxy":
		return generateNaiveLink(user, inbound, config, serverAddr)
	case "xhttp":
		return generateXHTTPLink(user, inbound, config, serverAddr, certsByIDs)
	case "ssr":
		return generateSSRLink(user, inbound, config, serverAddr)
	case "http":
		return generateHTTPLink(user, inbound, config, serverAddr)
	case "socks5":
		return generateSOCKS5Link(user, inbound, config, serverAddr)
	case "mixed":
		return generateMixedLink(user, inbound, config, serverAddr)
	case "snell":
		return generateSnellLink(user, inbound, config, serverAddr)
	case "mieru", "sudoku", "trusttunnel":
		return ""
	default:
		return ""
	}
}

func addTransportParams(params *url.Values, config map[string]interface{}) {
	transport := getStringOrDefault(config, "transport", "tcp")
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

func addTLSAndRealityParams(params *url.Values, inbound models.Inbound, certsByIDs map[uint]*models.Certificate) {
	tlsInfo := getInboundTLSInfo(inbound, certsByIDs)

	if inbound.RealityEnabled {
		params.Set("security", "reality")
		realityInfo := getInboundRealityInfo(inbound)
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

func proxyRemark(inbound models.Inbound) string {
	return formatCorePrefix(inbound) + inbound.Name
}

func generateVLESSLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}

	addTLSAndRealityParams(&params, inbound, certsByIDs)
	addTransportParams(&params, config)

	if flow := getStringOrDefault(config, "flow", ""); flow != "" {
		params.Set("flow", flow)
	}

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateVMessLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	transport := getStringOrDefault(config, "transport", "tcp")
	tlsInfo := getInboundTLSInfo(inbound, certsByIDs)

	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   proxyRemark(inbound),
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

func generateTrojanLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}

	addTLSAndRealityParams(&params, inbound, certsByIDs)
	addTransportParams(&params, config)

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateAnyTLSLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}
	password := getStringOrDefault(config, "password", user.UUID)
	addTLSAndRealityParams(&params, inbound, certsByIDs)
	return fmt.Sprintf("anytls://%s@%s:%d?%s#%s",
		password, server, inbound.Port,
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateSSLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	method := getStringOrDefault(config, "method", "aes-256-gcm")
	password := getStringOrDefault(config, "password", user.UUID)
	userInfo := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", method, password)))
	return fmt.Sprintf("ss://%s@%s:%d#%s",
		userInfo, server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
}

func generateHysteria2Link(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("insecure", "1")

	tlsInfo := getInboundTLSInfo(inbound, certsByIDs)
	if tlsInfo.SNI != "" {
		params.Set("sni", tlsInfo.SNI)
	}

	if obfsType := getStringOrDefault(config, "obfs_type", ""); obfsType != "" {
		params.Set("obfs", obfsType)
		if obfsPass := getStringOrDefault(config, "obfs_password", ""); obfsPass != "" {
			params.Set("obfs-password", obfsPass)
		}
	}

	return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateTUICv4Link(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("congestion_control", getStringOrDefault(config, "congestion_control", "bbr"))
	params.Set("alpn", getStringOrDefault(config, "alpn", "h3"))

	tlsInfo := getInboundTLSInfo(inbound, certsByIDs)
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
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateTUICv5Link(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("congestion_control", getStringOrDefault(config, "congestion_control", "bbr"))
	params.Set("alpn", getStringOrDefault(config, "alpn", "h3"))

	tlsInfo := getInboundTLSInfo(inbound, certsByIDs)
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
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateNaiveLink(user models.User, inbound models.Inbound, _ map[string]interface{}, server string) string {
	return fmt.Sprintf("naive+https://%s:%s@%s:%d#%s",
		url.PathEscape(user.Username), url.PathEscape(user.UUID),
		server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
}

func generateXHTTPLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string, certsByIDs map[uint]*models.Certificate) string {
	params := url.Values{}
	params.Set("type", "xhttp")

	addTLSAndRealityParams(&params, inbound, certsByIDs)

	if path, ok := config["path"].(string); ok && path != "" {
		params.Set("path", path)
	}
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		user.UUID, server, inbound.Port,
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateSSRLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	method := getStringOrDefault(config, "method", "chacha20-poly1305")
	protocol := getStringOrDefault(config, "protocol", "origin")
	obfs := getStringOrDefault(config, "obfs", "plain")
	passwordB64 := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(user.UUID))
	raw := fmt.Sprintf("%s:%d:%s:%s:%s:%s",
		server, inbound.Port, protocol, method, obfs, passwordB64)
	return "ssr://" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(raw))
}

func generateHTTPLink(user models.User, inbound models.Inbound, _ map[string]interface{}, server string) string {
	if user.Username != "" {
		return fmt.Sprintf("http://%s:%s@%s:%d#%s",
			url.PathEscape(user.Username), url.PathEscape(user.UUID),
			server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
	}
	return fmt.Sprintf("http://%s:%d#%s", server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
}

func generateSOCKS5Link(user models.User, inbound models.Inbound, _ map[string]interface{}, server string) string {
	if user.Username != "" {
		return fmt.Sprintf("socks5://%s:%s@%s:%d#%s",
			url.PathEscape(user.Username), url.PathEscape(user.UUID),
			server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
	}
	return fmt.Sprintf("socks5://%s:%d#%s", server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
}

func generateMixedLink(user models.User, inbound models.Inbound, _ map[string]interface{}, server string) string {
	if user.Username != "" {
		return fmt.Sprintf("mixed://%s:%s@%s:%d#%s",
			url.PathEscape(user.Username), url.PathEscape(user.UUID),
			server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
	}
	return fmt.Sprintf("mixed://%s:%d#%s", server, inbound.Port, url.PathEscape(proxyRemark(inbound)))
}

func generateSnellLink(user models.User, inbound models.Inbound, config map[string]interface{}, server string) string {
	params := url.Values{}
	psk := user.UUID
	if user.Token != nil && *user.Token != "" {
		psk = *user.Token
	}
	version := getIntOrDefault(config, "version", 3)
	params.Set("version", strconv.Itoa(version))
	if obfs := getStringOrDefault(config, "obfs", ""); obfs != "" {
		params.Set("obfs", obfs)
	}
	return fmt.Sprintf("snell://%s@%s:%d?%s#%s",
		psk, server, inbound.Port,
		params.Encode(), url.PathEscape(proxyRemark(inbound)))
}

func generateSingboxOutbound(user models.User, inbound models.Inbound, panelURL string, certsByIDs map[uint]*models.Certificate) map[string]interface{} {
	var config map[string]interface{}
	if inbound.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(inbound.ConfigJSON), &config); err != nil {
			logger.Log.Warn().Err(err).Uint("inbound_id", inbound.ID).Msg("Failed to parse inbound ConfigJSON for Sing-box outbound")
		}
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	server := resolveServerAddr(inbound, panelURL, certsByIDs)
	tlsInfo := getInboundTLSInfo(inbound, certsByIDs)
	transport := getStringOrDefault(config, "transport", "tcp")
	tag := formatCorePrefix(inbound) + inbound.Name

	buildTLSConfig := func() map[string]interface{} {
		tlsConfig := map[string]interface{}{
			"enabled":  true,
			"insecure": true,
		}
		if tlsInfo.SNI != "" {
			tlsConfig["server_name"] = tlsInfo.SNI
		}
		if inbound.RealityEnabled {
			realityInfo := getInboundRealityInfo(inbound)
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
		if flow := getStringOrDefault(config, "flow", ""); flow != "" {
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
			"method":      getStringOrDefault(config, "method", "aes-256-gcm"),
			"password":    getStringOrDefault(config, "password", user.UUID),
		}
		return ob
	case "hysteria2":
		ob := map[string]interface{}{
			"type":        "hysteria2",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"password":    getStringOrDefault(config, "password", user.UUID),
			"tls":         buildTLSConfig(),
		}
		if obfsType := getStringOrDefault(config, "obfs_type", ""); obfsType != "" {
			ob["obfs"] = map[string]interface{}{
				"type":     obfsType,
				"password": getStringOrDefault(config, "obfs_password", ""),
			}
		}
		return ob
	case "hysteria":
		ob := map[string]interface{}{
			"type":        "hysteria",
			"tag":         tag,
			"server":      server,
			"server_port": inbound.Port,
			"password":    getStringOrDefault(config, "auth_str", user.UUID),
			"tls":         buildTLSConfig(),
		}
		if upMbps := getIntOrDefault(config, "up_mbps", 0); upMbps > 0 {
			ob["up_mbps"] = upMbps
		}
		if downMbps := getIntOrDefault(config, "down_mbps", 0); downMbps > 0 {
			ob["down_mbps"] = downMbps
		}
		if obfs := getStringOrDefault(config, "obfs", ""); obfs != "" {
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
			"congestion_control": getStringOrDefault(config, "congestion_control", "bbr"),
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
			"password":    getStringOrDefault(config, "password", user.UUID),
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
