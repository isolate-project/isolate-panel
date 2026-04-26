package cache

import (
	"time"
)

// CacheManager provides different cache layers for the application
type CacheManager struct {
	settingsCache     *Cache
	configCache       *Cache
	subscriptionCache *Cache
	userCache         *Cache
}

// NewCacheManager creates a new cache manager with all cache layers
func NewCacheManager() (*CacheManager, error) {
	// Settings cache: small, fast, 1 minute TTL
	settingsCache, err := New(Config{
		NumCounters: 1000,
		MaxCost:     1 << 20, // 1MB
		BufferItems: 64,
		TTL:         1 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	// Config cache: medium size, 5 minute TTL
	configCache, err := New(Config{
		NumCounters: 10000,
		MaxCost:     10 << 20, // 10MB
		BufferItems: 64,
		TTL:         5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	// Subscription cache: larger, 5 minute TTL
	subscriptionCache, err := New(Config{
		NumCounters: 100000,
		MaxCost:     50 << 20, // 50MB
		BufferItems: 64,
		TTL:         5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	// User cache: medium size, 10 minute TTL
	userCache, err := New(Config{
		NumCounters: 10000,
		MaxCost:     10 << 20, // 10MB
		BufferItems: 64,
		TTL:         10 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return &CacheManager{
		settingsCache:     settingsCache,
		configCache:       configCache,
		subscriptionCache: subscriptionCache,
		userCache:         userCache,
	}, nil
}

// GetSettingsCache returns the settings cache
func (m *CacheManager) GetSettingsCache() *Cache {
	return m.settingsCache
}

// GetConfigCache returns the config cache
func (m *CacheManager) GetConfigCache() *Cache {
	return m.configCache
}

// GetSubscriptionCache returns the subscription cache
func (m *CacheManager) GetSubscriptionCache() *Cache {
	return m.subscriptionCache
}

// GetUserCache returns the user cache
func (m *CacheManager) GetUserCache() *Cache {
	return m.userCache
}

// Close closes all cache layers
func (m *CacheManager) Close() {
	m.settingsCache.Close()
	m.configCache.Close()
	m.subscriptionCache.Close()
	m.userCache.Close()
}

// GetMetrics returns metrics from all cache layers
func (m *CacheManager) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"settings":     m.settingsCache.Metrics(),
		"config":       m.configCache.Metrics(),
		"subscription": m.subscriptionCache.Metrics(),
		"user":         m.userCache.Metrics(),
	}
}

// ClearAll clears all cache layers
func (m *CacheManager) ClearAll() {
	m.settingsCache.Clear()
	m.configCache.Clear()
	m.subscriptionCache.Clear()
	m.userCache.Clear()
}

// ClearConfig clears config cache (useful when core config changes)
func (m *CacheManager) ClearConfig() {
	m.configCache.Clear()
}

// ClearSubscription clears the entire subscription cache.
// Ristretto lacks prefix-based deletion, so per-key deletion misses filtered variants
// (sub:{uid}:{format}:{hash}). Full cache clear is correct for limited-user panels.
func (m *CacheManager) ClearSubscription(userID uint) {
	m.subscriptionCache.Clear()
}

