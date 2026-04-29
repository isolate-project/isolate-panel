package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cache"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services/subscription"
)

// SubscriptionService generates subscription configs in multiple formats
type SubscriptionService struct {
	db       *gorm.DB
	panelURL string
	cache    *cache.Cache
	registry *subscription.Registry
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

	// Initialize registry with all generators
	registry := subscription.NewRegistry()
	registry.Register(subscription.NewV2RayGenerator(panelURL, db))
	registry.Register(subscription.NewClashGenerator(panelURL, db))
	registry.Register(subscription.NewSingboxGenerator(panelURL, db))
	registry.Register(subscription.NewIsolateGenerator(panelURL, db))

	return &SubscriptionService{
		db:       db,
		panelURL: panelURL,
		cache:    subCache,
		registry: registry,
	}
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
func (s *SubscriptionService) GetUserSubscriptionData(token string) (*subscription.UserSubscriptionData, error) {
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
		return &subscription.UserSubscriptionData{User: *user, Inbounds: []models.Inbound{}}, nil
	}

	inboundIDs := make([]uint, len(mappings))
	for i, m := range mappings {
		inboundIDs[i] = m.InboundID
	}

	var inbounds []models.Inbound
	if err := s.db.Where("id IN ? AND is_enabled = ?", inboundIDs, true).Preload("Core").Find(&inbounds).Error; err != nil {
		return nil, fmt.Errorf("failed to get inbounds: %w", err)
	}

	return &subscription.UserSubscriptionData{User: *user, Inbounds: inbounds}, nil
}

// GenerateV2Ray generates a V2Ray/auto-detect format subscription
func (s *SubscriptionService) GenerateV2Ray(data *subscription.UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "v2ray", filterHash); ok {
		return cached, nil
	}

	result, err := s.registry.Generate("v2ray", data)
	if err != nil {
		return "", err
	}

	s.SetCachedSubscription(data.User.ID, "v2ray", filterHash, result)
	return result, nil
}

// GenerateClash generates a Clash YAML format subscription
func (s *SubscriptionService) GenerateClash(data *subscription.UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "clash", filterHash); ok {
		return cached, nil
	}

	result, err := s.registry.Generate("clash", data)
	if err != nil {
		return "", err
	}

	s.SetCachedSubscription(data.User.ID, "clash", filterHash, result)
	return result, nil
}

// GenerateSingbox generates a Sing-box JSON format subscription
func (s *SubscriptionService) GenerateSingbox(data *subscription.UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "singbox", filterHash); ok {
		return cached, nil
	}

	result, err := s.registry.Generate("singbox", data)
	if err != nil {
		return "", err
	}

	s.SetCachedSubscription(data.User.ID, "singbox", filterHash, result)
	return result, nil
}

// GenerateIsolate generates an Isolate custom JSON format subscription
func (s *SubscriptionService) GenerateIsolate(data *subscription.UserSubscriptionData) (string, error) {
	filterHash := ""
	if data.Filter != nil {
		filterHash = data.Filter.Hash()
	}
	if cached, ok := s.GetCachedSubscription(data.User.ID, "isolate", filterHash); ok {
		return cached, nil
	}

	result, err := s.registry.Generate("isolate", data)
	if err != nil {
		return "", err
	}

	s.SetCachedSubscription(data.User.ID, "isolate", filterHash, result)
	return result, nil
}

// GetCachedSubscription retrieves cached subscription content
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
	newToken := generateSubscriptionToken()

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	user.SubscriptionToken = newToken

	if err := s.db.Save(&user).Error; err != nil {
		return "", fmt.Errorf("failed to update user token: %w", err)
	}

	s.db.Where("user_id = ?", userID).Delete(&models.SubscriptionShortURL{})
	s.InvalidateUserCache(userID)

	return newToken, nil
}

func generateShortCode(length int) (string, error) {
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
