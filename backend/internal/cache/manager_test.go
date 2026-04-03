package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheManager(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	require.NotNil(t, cm)
	defer cm.Close()

	assert.NotNil(t, cm.GetSettingsCache())
	assert.NotNil(t, cm.GetConfigCache())
	assert.NotNil(t, cm.GetSubscriptionCache())
	assert.NotNil(t, cm.GetUserCache())
}

func TestCacheManager_SettingsCache(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	sc := cm.GetSettingsCache()
	sc.Set("key1", "value1")
	time.Sleep(10 * time.Millisecond)

	val, ok := sc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestCacheManager_ConfigCache(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	cc := cm.GetConfigCache()
	cc.Set("cfg:core1", map[string]string{"port": "1080"})
	time.Sleep(10 * time.Millisecond)

	val, ok := cc.Get("cfg:core1")
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestCacheManager_SubscriptionCache(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	sub := cm.GetSubscriptionCache()
	sub.Set("sub:user:1:v2ray", "vmess://...")
	time.Sleep(10 * time.Millisecond)

	val, ok := sub.Get("sub:user:1:v2ray")
	assert.True(t, ok)
	assert.Equal(t, "vmess://...", val)
}

func TestCacheManager_UserCache(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	uc := cm.GetUserCache()
	uc.Set("user:42", struct{ Name string }{"Alice"})
	time.Sleep(10 * time.Millisecond)

	val, ok := uc.Get("user:42")
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestCacheManager_ClearAll(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	cm.GetSettingsCache().Set("k", "v")
	cm.GetConfigCache().Set("k", "v")
	time.Sleep(10 * time.Millisecond)

	cm.ClearAll()
	time.Sleep(10 * time.Millisecond)

	_, ok := cm.GetSettingsCache().Get("k")
	assert.False(t, ok)
}

func TestCacheManager_ClearConfig(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	cm.GetConfigCache().Set("cfg:x", "data")
	time.Sleep(10 * time.Millisecond)

	cm.ClearConfig()
	time.Sleep(10 * time.Millisecond)

	_, ok := cm.GetConfigCache().Get("cfg:x")
	assert.False(t, ok)
}

func TestCacheManager_ClearSubscription(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	sc := cm.GetSubscriptionCache()
	var userID uint = 7
	for _, format := range []string{"v2ray", "clash", "singbox"} {
		sc.Set(getSubscriptionKey(userID, format), "data:"+format)
	}
	time.Sleep(10 * time.Millisecond)

	cm.ClearSubscription(userID)
	time.Sleep(10 * time.Millisecond)

	for _, format := range []string{"v2ray", "clash", "singbox"} {
		_, ok := sc.Get(getSubscriptionKey(userID, format))
		assert.False(t, ok, "key for format %s should be cleared", format)
	}
}

func TestCacheManager_GetMetrics(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	defer cm.Close()

	metrics := cm.GetMetrics()
	assert.Contains(t, metrics, "settings")
	assert.Contains(t, metrics, "config")
	assert.Contains(t, metrics, "subscription")
	assert.Contains(t, metrics, "user")
}

func TestCacheManager_Close(t *testing.T) {
	cm, err := NewCacheManager()
	require.NoError(t, err)
	// Should not panic
	assert.NotPanics(t, func() { cm.Close() })
}

func TestGetSubscriptionKey(t *testing.T) {
	tests := []struct {
		userID uint
		format string
	}{
		{1, "v2ray"},
		{42, "clash"},
		{100, "singbox"},
	}
	for _, tc := range tests {
		key := getSubscriptionKey(tc.userID, tc.format)
		assert.Contains(t, key, "subscription:")
		assert.Contains(t, key, tc.format)
		_ = fmt.Sprintf("key=%s", key) // use key
	}
}
