# Caching Strategy for Isolate Panel

## Overview

Isolate Panel uses a multi-layer caching strategy to improve performance and reduce database load. The caching system is built on top of [Ristretto](https://github.com/dgraph-io/ristretto), a high-performance Go cache library.

## Architecture

### Cache Layers

```
┌─────────────────────────────────────────┐
│         Application Layer                │
├─────────────────────────────────────────┤
│  Settings Cache (1MB, 1min TTL)         │
│  Config Cache (10MB, 5min TTL)          │
│  Subscription Cache (50MB, 5min TTL)    │
│  User Cache (10MB, 10min TTL)           │
├─────────────────────────────────────────┤
│         Cache Manager                    │
├─────────────────────────────────────────┤
│         Database Layer                   │
└─────────────────────────────────────────┘
```

### Cache Configuration

| Cache Layer | Max Size | TTL | Use Case |
|-------------|----------|-----|----------|
| **Settings** | 1MB | 1 minute | Application settings, frequently accessed |
| **Config** | 10MB | 5 minutes | Core configurations (Sing-box, Xray, Mihomo) |
| **Subscription** | 50MB | 5 minutes | Generated subscription configs (V2Ray, Clash, Sing-box) |
| **User** | 10MB | 10 minutes | User data, credentials |

## Implementation

### Using Cache Manager

```go
package main

import (
    "github.com/vovk4morkovk4/isolate-panel/internal/cache"
)

func main() {
    // Create cache manager
    cacheManager, err := cache.NewCacheManager()
    if err != nil {
        log.Fatal(err)
    }
    defer cacheManager.Close()
    
    // Get specific cache layer
    settingsCache := cacheManager.GetSettingsCache()
    
    // Set value with default TTL
    settingsCache.Set("key", "value")
    
    // Set value with custom TTL
    settingsCache.SetWithTTL("key", "value", 10*time.Minute)
    
    // Get value
    if val, found := settingsCache.Get("key"); found {
        // Use cached value
    }
    
    // Delete value
    settingsCache.Delete("key")
}
```

### Service Integration

#### Settings Service

```go
package services

type SettingsService struct {
    db    *gorm.DB
    cache *cache.Cache
}

func NewSettingsService(db *gorm.DB, cacheManager *cache.CacheManager) *SettingsService {
    return &SettingsService{
        db:    db,
        cache: cacheManager.GetSettingsCache(),
    }
}

func (s *SettingsService) GetSettingValue(key string) (string, error) {
    // Try cache first
    if s.cache != nil {
        if cached, found := s.cache.GetString("setting_value:" + key); found {
            return cached, nil
        }
    }
    
    // Query database
    setting, err := s.GetSetting(key)
    if err != nil {
        return "", err
    }
    
    // Cache the value
    if s.cache != nil {
        s.cache.Set("setting_value:"+key, setting.Value)
    }
    
    return setting.Value, nil
}

func (s *SettingsService) UpdateSetting(key string, value string) error {
    // Update database
    // ...
    
    // Invalidate cache
    if s.cache != nil {
        s.cache.Delete("setting:" + key)
        s.cache.Delete("setting_value:" + key)
    }
    
    return nil
}
```

#### Subscription Service

```go
package services

type SubscriptionService struct {
    db    *gorm.DB
    cache *cache.Cache
}

func NewSubscriptionService(db *gorm.DB, panelURL string, cacheManager *cache.CacheManager) *SubscriptionService {
    return &SubscriptionService{
        db:    db,
        cache: cacheManager.GetSubscriptionCache(),
        // ...
    }
}

func (s *SubscriptionService) GenerateV2Ray(data *UserSubscriptionData) (string, error) {
    cacheKey := fmt.Sprintf("subscription:v2ray:%d", data.User.ID)
    
    // Try cache first
    if s.cache != nil {
        if cached, found := s.cache.GetString(cacheKey); found {
            return cached, nil
        }
    }
    
    // Generate subscription
    config, err := s.generateV2RayConfig(data)
    if err != nil {
        return "", err
    }
    
    // Cache the result
    if s.cache != nil {
        s.cache.Set(cacheKey, config)
    }
    
    return config, nil
}

func (s *SubscriptionService) InvalidateUserCache(userID uint) {
    if s.cache != nil {
        s.cache.Delete(fmt.Sprintf("subscription:v2ray:%d", userID))
        s.cache.Delete(fmt.Sprintf("subscription:clash:%d", userID))
        s.cache.Delete(fmt.Sprintf("subscription:singbox:%d", userID))
    }
}
```

## Cache Invalidation

### Strategies

1. **TTL-based**: Automatic expiration after TTL
2. **Write-through**: Update cache on write
3. **Invalidate-on-write**: Delete cache on write, repopulate on next read

### When to Invalidate

| Event | Cache Layers to Invalidate |
|-------|---------------------------|
| User created/updated | User cache, Subscription cache |
| User deleted | User cache, Subscription cache |
| Inbound created/updated/deleted | Config cache |
| Setting updated | Settings cache |
| Core restarted | Config cache |

### Example: Cache Invalidation on User Update

```go
func (us *UserService) UpdateUser(id uint, req *UpdateUserRequest) (*models.User, error) {
    // Update user in database
    user, err := us.UpdateUserInDB(id, req)
    if err != nil {
        return nil, err
    }
    
    // Invalidate user cache
    if us.cache != nil {
        us.cache.Delete(fmt.Sprintf("user:%d", id))
        us.cache.Delete(fmt.Sprintf("user:uuid:%s", user.UUID))
    }
    
    // Invalidate subscription cache
    if us.subscriptionCache != nil {
        us.subscriptionCache.Delete(fmt.Sprintf("subscription:v2ray:%d", id))
        us.subscriptionCache.Delete(fmt.Sprintf("subscription:clash:%d", id))
        us.subscriptionCache.Delete(fmt.Sprintf("subscription:singbox:%d", id))
    }
    
    return user, nil
}
```

## Performance Metrics

### Monitoring Cache Performance

```go
// Get cache metrics
metrics := cacheManager.GetMetrics()

// Settings cache metrics
settingsMetrics := metrics["settings"].(*ristretto.Metrics)
fmt.Printf("Hit ratio: %.2f%%\n", settingsMetrics.Ratio()*100)
fmt.Printf("Keys added: %d\n", settingsMetrics.KeysAdded())
fmt.Printf("Keys evicted: %d\n", settingsMetrics.KeysEvicted())
fmt.Printf("Cost added: %d\n", settingsMetrics.CostAdded())
fmt.Printf("Cost evicted: %d\n", settingsMetrics.CostEvicted())
```

### Target Metrics

| Metric | Target | Warning | Critical |
|--------|--------|---------|----------|
| Hit Ratio | > 80% | 60-80% | < 60% |
| Eviction Rate | < 10% | 10-20% | > 20% |
| Memory Usage | < 80% | 80-90% | > 90% |

## Best Practices

### Do's

✅ **Use cache for frequently accessed data**
```go
// Good: Settings are accessed frequently
setting := cache.Get("setting:monitoring_mode")
```

✅ **Invalidate cache on writes**
```go
// Good: Invalidate after update
db.Update(setting)
cache.Delete("setting:" + key)
```

✅ **Use appropriate TTL**
```go
// Good: Short TTL for volatile data
cache.SetWithTTL("stats", data, 1*time.Minute)

// Good: Longer TTL for stable data
cache.SetWithTTL("config", data, 10*time.Minute)
```

✅ **Use cache key prefixes**
```go
// Good: Clear key structure
cache.Set("setting:"+key, value)
cache.Set("user:"+string(id), user)
cache.Set("subscription:"+format+":"+string(userID), config)
```

### Don'ts

❌ **Don't cache everything**
```go
// Bad: One-time data doesn't need caching
cache.Set("migration:status", status)
```

❌ **Don't use very long TTLs**
```go
// Bad: Data might become stale
cache.SetWithTTL("user:1", user, 24*time.Hour)
```

❌ **Don't forget to invalidate**
```go
// Bad: Cache becomes stale
db.Update(user)
// Missing: cache.Delete("user:" + id)
```

❌ **Don't cache large objects without cost**
```go
// Bad: No cost specified
cache.Set("large:data", hugeData)

// Good: Specify cost
cache.SetWithTTL("large:data", hugeData, 100) // 100 bytes cost
```

## Troubleshooting

### Low Hit Ratio

**Symptoms:** Hit ratio < 60%

**Possible Causes:**
1. TTL too short
2. Cache size too small
3. High data variability

**Solutions:**
1. Increase TTL
2. Increase MaxCost
3. Review caching strategy

### High Eviction Rate

**Symptoms:** Eviction rate > 20%

**Possible Causes:**
1. Cache size too small
2. Too many unique keys

**Solutions:**
1. Increase MaxCost
2. Review what's being cached
3. Use more selective caching

### Memory Issues

**Symptoms:** High memory usage

**Possible Causes:**
1. Cache too large
2. Objects not being evicted

**Solutions:**
1. Reduce MaxCost
2. Check for memory leaks
3. Use ristretto metrics to monitor

## Future Improvements

### Planned Enhancements

1. **Distributed Cache**: Redis integration for multi-instance deployments
2. **Cache Warming**: Pre-populate cache on startup
3. **Adaptive TTL**: Dynamic TTL based on access patterns
4. **Cache Analytics**: Dashboard for cache metrics
5. **Query Result Cache**: Cache database query results

### Post-MVP Considerations

- **Redis Integration**: For horizontal scaling
- **Cache Compression**: For large objects
- **Cache Tiering**: Hot/warm/cold cache layers
- **Predictive Caching**: ML-based cache preloading

## References

- [Ristretto Documentation](https://github.com/dgraph-io/ristretto)
- [Cache Performance Patterns](https://martinfowler.com/bliki/CachePattern.html)
- [Caching Best Practices](https://aws.amazon.com/caching/)
