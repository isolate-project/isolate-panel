package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

// Config holds cache configuration
type Config struct {
	NumCounters int64         // Number of keys to track frequency of
	MaxCost     int64         // Maximum cost of cache
	BufferItems int64         // Number of keys per Get buffer
	TTL         time.Duration // Default TTL for items
}

// DefaultConfig returns default cache configuration
func DefaultConfig() Config {
	return Config{
		NumCounters: 1e7,       // 10 million keys
		MaxCost:     100 << 20, // 100MB
		BufferItems: 64,
		TTL:         5 * time.Minute,
	}
}

// Cache is a wrapper around ristretto cache with TTL support
type Cache struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// New creates a new cache instance
func New(config Config) (*Cache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
		Metrics:     true,
	})
	if err != nil {
		return nil, err
	}

	return &Cache{
		cache: cache,
		ttl:   config.TTL,
	}, nil
}

// Set adds an item to the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) bool {
	return c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL adds an item to the cache with a specific TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) bool {
	return c.cache.SetWithTTL(key, value, 1, ttl)
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// GetString retrieves a string item from the cache
func (c *Cache) GetString(key string) (string, bool) {
	value, found := c.cache.Get(key)
	if !found {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.cache.Del(key)
}

// Clear clears all items from the cache
func (c *Cache) Clear() {
	c.cache.Clear()
}

// Wait blocks until all buffered writes have been applied
func (c *Cache) Wait() {
	c.cache.Wait()
}

// Metrics returns cache metrics
func (c *Cache) Metrics() *ristretto.Metrics {
	return c.cache.Metrics
}

// Close closes the cache
func (c *Cache) Close() {
	c.cache.Close()
}

// GetOrSet gets a value from cache or sets it if not found
func (c *Cache) GetOrSet(key string, value interface{}) (interface{}, bool) {
	if val, found := c.Get(key); found {
		return val, true
	}
	c.Set(key, value)
	return value, false
}

// GetOrSetWithTTL gets a value or sets it with specific TTL
func (c *Cache) GetOrSetWithTTL(key string, value interface{}, ttl time.Duration) (interface{}, bool) {
	if val, found := c.Get(key); found {
		return val, true
	}
	c.SetWithTTL(key, value, ttl)
	return value, false
}
