package cache

import (
	"testing"
	"time"
)

func TestCacheManager(t *testing.T) {
	manager, err := NewCacheManager()
	if err != nil {
		t.Fatalf("NewCacheManager() error = %v", err)
	}
	defer manager.Close()

	if manager.GetSettingsCache() == nil {
		t.Error("expected settings cache, got nil")
	}
	if manager.GetConfigCache() == nil {
		t.Error("expected config cache, got nil")
	}
	if manager.GetSubscriptionCache() == nil {
		t.Error("expected subscription cache, got nil")
	}
	if manager.GetUserCache() == nil {
		t.Error("expected user cache, got nil")
	}

	// Test caching behavior
	cache := manager.GetSettingsCache()
	defer cache.Clear()

	cache.Set("test_key", "test_value")

	// Wait a tiny bit for ristretto to sync
	time.Sleep(10 * time.Millisecond)

	val, found := cache.Get("test_key")
	if !found {
		t.Error("expected to find test_key in cache")
	}
	if val != "test_value" {
		t.Errorf("expected test_value, got %v", val)
	}

	// Test ClearConfig
	configCache := manager.GetConfigCache()
	configCache.Set("cfg", "val")
	time.Sleep(10 * time.Millisecond) // Wait for sync
	manager.ClearConfig()
	time.Sleep(10 * time.Millisecond) // Wait for sync

	if _, found := configCache.Get("cfg"); found {
		t.Error("expected config cache to be clear, but found key")
	}

	// Test ClearSubscription
	subCache := manager.GetSubscriptionCache()
	subCache.Set(getSubscriptionKey(1, "v2ray"), "sub_data")
	time.Sleep(10 * time.Millisecond) // Wait for sync
	manager.ClearSubscription(1)
	time.Sleep(10 * time.Millisecond) // Wait for sync

	if _, found := subCache.Get(getSubscriptionKey(1, "v2ray")); found {
		t.Error("expected subscription to be cleared")
	}

	// Test metrics
	metrics := manager.GetMetrics()
	if len(metrics) != 4 {
		t.Errorf("expected 4 metrics maps, got %d", len(metrics))
	}
}
