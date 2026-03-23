# 🔒 План безопасности Isolate Panel

**Дата:** 23 марта 2026  
**Версия:** 1.0  
**Статус:** Утверждено для MVP

---

## 📋 Оглавление

1. [Обзор безопасности](#обзор-безопасности)
2. [Защита панели управления](#защита-панели-управления)
3. [Защита subscription endpoints](#защита-subscription-endpoints)
4. [Хеширование паролей](#хеширование-паролей)
5. [Защита от атак](#защита-от-атак)
6. [Мониторинг и алерты](#мониторинг-и-алерты)

---

## Обзор безопасности

### Модель угроз

**Панель управления (localhost:8080):**
- ✅ Доступна ТОЛЬКО через SSH tunnel
- ✅ Не требует дополнительной защиты от DDoS
- ✅ JWT аутентификация для админов

**Subscription endpoints (/sub/*):**
- ⚠️ Публично доступны из интернета
- ⚠️ Потенциальная цель для DDoS атак
- ⚠️ Риск brute force токенов
- ⚠️ Требуют многоуровневой защиты

---

## Защита панели управления

### JWT аутентификация

**Конфигурация:**
```go
type JWTConfig struct {
    Secret           string        // Из переменной окружения JWT_SECRET
    AccessTokenTTL   time.Duration // 15 минут
    RefreshTokenTTL  time.Duration // 7 дней
    Issuer           string        // "isolate-panel"
}
```

**Генерация секрета:**
```bash
# При первом запуске
JWT_SECRET=$(openssl rand -base64 64)

# Проверка длины (должно быть >= 64 символов)
if [ ${#JWT_SECRET} -lt 64 ]; then
    echo "ERROR: JWT secret too short"
    exit 1
fi
```

**Middleware:**
```go
func JWTMiddleware() fiber.Handler {
    return jwtware.New(jwtware.Config{
        SigningKey: jwtware.SigningKey{
            Key: []byte(os.Getenv("JWT_SECRET")),
        },
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            return c.Status(401).JSON(fiber.Map{
                "error": "Unauthorized",
            })
        },
    })
}
```

### Rate limiting для API

**Конфигурация:**
```go
// Для login endpoint
app.Post("/api/auth/login", 
    limiter.New(limiter.Config{
        Max:        5,                    // 5 попыток
        Expiration: 1 * time.Minute,      // в минуту
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP()
        },
    }),
    authHandler.Login,
)

// Для создания пользователей
app.Post("/api/users",
    JWTMiddleware(),
    limiter.New(limiter.Config{
        Max:        10,
        Expiration: 1 * time.Minute,
    }),
    userHandler.Create,
)
```

---

## Защита subscription endpoints

### Архитектура защиты

**Уровни защиты:**
1. Request Validation (User-Agent, IP блокировка)
2. Rate Limiting (IP + Token + Global)
3. Failed Request Tracking (автоблокировка)
4. Access Logging (полное логирование)
5. Anomaly Detection (фоновый мониторинг)

### 1. Request Validation

**Middleware:**
```go
// internal/middleware/subscription_validator.go
type SubscriptionValidator struct {
    failedTracker *FailedRequestTracker
}

func (sv *SubscriptionValidator) Validate() fiber.Handler {
    return func(c *fiber.Ctx) error {
        ip := c.IP()
        
        // 1. Проверка блокировки IP
        if sv.failedTracker.IsBlocked(ip) {
            log.Warn().Str("ip", ip).Msg("Blocked IP attempted access")
            return c.Status(403).SendString("Your IP has been blocked")
        }
        
        // 2. Проверка User-Agent
        userAgent := c.Get("User-Agent")
        if userAgent == "" {
            sv.failedTracker.RecordFailedAttempt(ip)
            return c.Status(400).SendString("User-Agent required")
        }
        
        // 3. Проверка подозрительных User-Agent
        if sv.isSuspiciousUserAgent(userAgent) {
            log.Warn().
                Str("ua", userAgent).
                Str("ip", ip).
                Msg("Suspicious User-Agent detected")
            sv.failedTracker.RecordFailedAttempt(ip)
            return c.Status(403).SendString("Forbidden")
        }
        
        return c.Next()
    }
}

func (sv *SubscriptionValidator) isSuspiciousUserAgent(ua string) bool {
    suspicious := []string{
        "curl", "wget", "python", "go-http-client",
        "scanner", "bot", "crawler", "scraper",
        "nikto", "nmap", "masscan", "sqlmap",
    }
    
    uaLower := strings.ToLower(ua)
    for _, s := range suspicious {
        if strings.Contains(uaLower, s) {
            return true
        }
    }
    return false
}
```

### 2. Multi-level Rate Limiting

**Конфигурация:**
```go
// internal/middleware/subscription_rate_limiter.go
type SubscriptionRateLimiter struct {
    ipLimiter     *cache.Cache  // IP-based: 30 req/hour
    tokenLimiter  *cache.Cache  // Token-based: 10 req/hour
    globalCounter *atomic.Int64 // Global: 1000 req/hour
}

func NewSubscriptionRateLimiter() *SubscriptionRateLimiter {
    srl := &SubscriptionRateLimiter{
        ipLimiter:     cache.New(1*time.Hour, 2*time.Hour),
        tokenLimiter:  cache.New(1*time.Hour, 2*time.Hour),
        globalCounter: &atomic.Int64{},
    }
    
    // Сброс глобального счетчика каждый час
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        for range ticker.C {
            srl.globalCounter.Store(0)
        }
    }()
    
    return srl
}

func (srl *SubscriptionRateLimiter) Middleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        ip := c.IP()
        token := c.Params("token")
        
        // Проверка 1: Global rate limit
        globalCount := srl.globalCounter.Add(1)
        if globalCount > 1000 {
            log.Error().Int64("count", globalCount).Msg("Global rate limit exceeded")
            return c.Status(503).SendString("Service temporarily unavailable")
        }
        
        // Проверка 2: IP rate limit
        if !srl.checkIPLimit(ip) {
            log.Warn().Str("ip", ip).Msg("IP rate limit exceeded")
            return c.Status(429).SendString("Too many requests from your IP")
        }
        
        // Проверка 3: Token rate limit
        if token != "" && !srl.checkTokenLimit(token) {
            log.Warn().Str("token", token).Msg("Token rate limit exceeded")
            return c.Status(429).SendString("Too many requests for this subscription")
        }
        
        return c.Next()
    }
}

func (srl *SubscriptionRateLimiter) checkIPLimit(ip string) bool {
    key := fmt.Sprintf("ip:%s", ip)
    count, found := srl.ipLimiter.Get(key)
    if !found {
        srl.ipLimiter.Set(key, 1, cache.DefaultExpiration)
        return true
    }
    
    currentCount := count.(int)
    if currentCount >= 30 {
        return false
    }
    
    srl.ipLimiter.Set(key, currentCount+1, cache.DefaultExpiration)
    return true
}

func (srl *SubscriptionRateLimiter) checkTokenLimit(token string) bool {
    key := fmt.Sprintf("token:%s", token)
    count, found := srl.tokenLimiter.Get(key)
    if !found {
        srl.tokenLimiter.Set(key, 1, cache.DefaultExpiration)
        return true
    }
    
    currentCount := count.(int)
    if currentCount >= 10 {
        return false
    }
    
    srl.tokenLimiter.Set(key, currentCount+1, cache.DefaultExpiration)
    return true
}
```

### 3. Failed Request Tracking

**Реализация:**
```go
// internal/services/failed_request_tracker.go
type FailedRequestTracker struct {
    cache *cache.Cache
    db    *gorm.DB
}

func NewFailedRequestTracker(db *gorm.DB) *FailedRequestTracker {
    return &FailedRequestTracker{
        cache: cache.New(1*time.Hour, 2*time.Hour),
        db:    db,
    }
}

func (frt *FailedRequestTracker) RecordFailedAttempt(ip string) {
    key := fmt.Sprintf("failed:%s", ip)
    count, found := frt.cache.Get(key)
    if !found {
        frt.cache.Set(key, 1, 1*time.Hour)
        return
    }
    
    newCount := count.(int) + 1
    frt.cache.Set(key, newCount, 1*time.Hour)
    
    // Блокировка после 20 неудачных попыток
    if newCount >= 20 {
        frt.BlockIP(ip, 24*time.Hour, "excessive_failed_requests")
        log.Warn().
            Str("ip", ip).
            Int("attempts", newCount).
            Msg("IP blocked due to excessive failed requests")
    }
}

func (frt *FailedRequestTracker) IsBlocked(ip string) bool {
    // Проверка в кэше
    key := fmt.Sprintf("blocked:%s", ip)
    if _, found := frt.cache.Get(key); found {
        return true
    }
    
    // Проверка в БД
    var blockedIP models.BlockedIP
    err := frt.db.Where("ip_address = ? AND expires_at > ?", ip, time.Now()).
        First(&blockedIP).Error
    
    if err == nil {
        // Добавить в кэш для быстрой проверки
        ttl := time.Until(blockedIP.ExpiresAt)
        frt.cache.Set(key, true, ttl)
        return true
    }
    
    return false
}

func (frt *FailedRequestTracker) BlockIP(ip string, duration time.Duration, reason string) {
    expiresAt := time.Now().Add(duration)
    
    // Сохранить в БД
    blockedIP := models.BlockedIP{
        IPAddress: ip,
        Reason:    reason,
        ExpiresAt: expiresAt,
    }
    frt.db.Create(&blockedIP)
    
    // Добавить в кэш
    key := fmt.Sprintf("blocked:%s", ip)
    frt.cache.Set(key, true, duration)
}

func (frt *FailedRequestTracker) GetFailedCount(ip string) int {
    key := fmt.Sprintf("failed:%s", ip)
    count, found := frt.cache.Get(key)
    if !found {
        return 0
    }
    return count.(int)
}
```

### 4. Enhanced Access Logging

**Модель:**
```go
// internal/models/subscription_access.go
type SubscriptionAccess struct {
    ID              uint      `gorm:"primaryKey"`
    UserID          uint      `gorm:"not null;index"`
    IPAddress       string    `gorm:"size:45;not null"`
    UserAgent       string    `gorm:"size:512"`
    Country         string    `gorm:"size:2"`
    Format          string    `gorm:"size:20"` // v2ray, clash, singbox
    IsSuspicious    bool      `gorm:"default:false"`
    ResponseTimeMs  int       `gorm:"default:0"`
    AccessedAt      time.Time `gorm:"not null;index"`
}
```

**Логирование:**
```go
func (h *SubscriptionHandler) logAccess(c *fiber.Ctx, userID uint, format string, startTime time.Time) {
    responseTime := time.Since(startTime).Milliseconds()
    
    access := models.SubscriptionAccess{
        UserID:         userID,
        IPAddress:      c.IP(),
        UserAgent:      c.Get("User-Agent"),
        Format:         format,
        ResponseTimeMs: int(responseTime),
        AccessedAt:     time.Now(),
    }
    
    // Опционально: определить страну через GeoIP
    if h.geoIP != nil {
        access.Country = h.geoIP.GetCountry(c.IP())
    }
    
    // Пометить как подозрительный если нужно
    if h.validator.isSuspiciousUserAgent(access.UserAgent) {
        access.IsSuspicious = true
    }
    
    h.db.Create(&access)
}
```

### 5. Middleware Stack

**Применение:**
```go
// internal/routes/subscription.go
func SetupSubscriptionRoutes(app *fiber.App, handler *SubscriptionHandler) {
    validator := middleware.NewSubscriptionValidator(handler.FailedTracker)
    rateLimiter := middleware.NewSubscriptionRateLimiter()
    
    // V2Ray format
    app.Get("/sub/:token",
        validator.Validate(),
        rateLimiter.Middleware(),
        handler.GetSubscription,
    )
    
    // Clash format
    app.Get("/sub/:token/clash",
        validator.Validate(),
        rateLimiter.Middleware(),
        handler.GetSubscription,
    )
    
    // Sing-box format
    app.Get("/sub/:token/singbox",
        validator.Validate(),
        rateLimiter.Middleware(),
        handler.GetSubscription,
    )
    
    // Short URL
    app.Get("/s/:short_code",
        validator.Validate(),
        rateLimiter.Middleware(),
        handler.GetByShortURL,
    )
    
    // QR Code
    app.Get("/sub/:token/qr",
        validator.Validate(),
        rateLimiter.Middleware(),
        handler.GetQRCode,
    )
}
```

---

## Хеширование паролей

### Argon2id для админов

**Конфигурация:**
```go
// internal/auth/password.go
type Argon2Config struct {
    Time        uint32 // 1 iteration
    Memory      uint32 // 64 MB
    Threads     uint8  // 4 threads
    KeyLength   uint32 // 32 bytes
    SaltLength  uint32 // 16 bytes
}

var DefaultArgon2Config = Argon2Config{
    Time:       1,
    Memory:     64 * 1024,
    Threads:    4,
    KeyLength:  32,
    SaltLength: 16,
}
```

**Хеширование:**
```go
func HashPassword(password string) (string, error) {
    // Генерировать уникальный salt
    salt := make([]byte, DefaultArgon2Config.SaltLength)
    if _, err := rand.Read(salt); err != nil {
        return "", fmt.Errorf("failed to generate salt: %w", err)
    }
    
    // Хешировать с Argon2id
    hash := argon2.IDKey(
        []byte(password),
        salt,
        DefaultArgon2Config.Time,
        DefaultArgon2Config.Memory,
        DefaultArgon2Config.Threads,
        DefaultArgon2Config.KeyLength,
    )
    
    // Формат: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
    encoded := fmt.Sprintf(
        "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version,
        DefaultArgon2Config.Memory,
        DefaultArgon2Config.Time,
        DefaultArgon2Config.Threads,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    )
    
    return encoded, nil
}
```

**Верификация:**
```go
func VerifyPassword(password, encodedHash string) (bool, error) {
    // Парсинг encoded hash
    parts := strings.Split(encodedHash, "$")
    if len(parts) != 6 {
        return false, errors.New("invalid hash format")
    }
    
    // Извлечь параметры
    var memory, time uint32
    var threads uint8
    _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
    if err != nil {
        return false, err
    }
    
    // Декодировать salt и hash
    salt, err := base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return false, err
    }
    
    expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return false, err
    }
    
    // Хешировать введенный пароль с теми же параметрами
    hash := argon2.IDKey(
        []byte(password),
        salt,
        time,
        memory,
        threads,
        uint32(len(expectedHash)),
    )
    
    // Constant-time сравнение
    return subtle.ConstantTimeCompare(hash, expectedHash) == 1, nil
}
```

---

## Защита от атак

### SQL Injection

**Защита:**
- ✅ Использование GORM (параметризованные запросы)
- ✅ Валидация всех входных данных
- ✅ Никаких raw SQL запросов с конкатенацией

**Пример безопасного кода:**
```go
// ПРАВИЛЬНО
user, err := userService.GetBySubscriptionToken(token)

// НЕПРАВИЛЬНО (никогда не делать)
// db.Raw("SELECT * FROM users WHERE subscription_token = '" + token + "'")
```

### XSS (Cross-Site Scripting)

**Защита:**
- ✅ Preact автоматически экранирует вывод
- ✅ Валидация всех входных данных
- ✅ Content-Security-Policy headers

**CSP Headers:**
```go
app.Use(func(c *fiber.Ctx) error {
    c.Set("Content-Security-Policy", 
        "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
    c.Set("X-Content-Type-Options", "nosniff")
    c.Set("X-Frame-Options", "DENY")
    c.Set("X-XSS-Protection", "1; mode=block")
    return c.Next()
})
```

### CSRF (Cross-Site Request Forgery)

**Защита:**
- ✅ CSRF tokens для всех POST/PUT/DELETE
- ✅ SameSite cookies

**Middleware:**
```go
app.Use(csrf.New(csrf.Config{
    KeyLookup:      "header:X-CSRF-Token",
    CookieName:     "csrf_",
    CookieSameSite: "Strict",
    Expiration:     1 * time.Hour,
}))
```

---

## Мониторинг и алерты

### Метрики безопасности

**Критические метрики:**
```go
type SecurityMetrics struct {
    // Subscription endpoints
    SubscriptionRequestsPerMinute int64
    FailedRequestsRate            float64
    BlockedIPsCount               int64
    SuspiciousUserAgentsCount     int64
    
    // API endpoints
    LoginAttemptsPerMinute        int64
    FailedLoginsRate              float64
    
    // Performance
    AverageResponseTime           time.Duration
    P95ResponseTime               time.Duration
}
```

**Сбор метрик:**
```go
func (sm *SecurityMetrics) CollectMetrics() {
    ticker := time.NewTicker(1 * time.Minute)
    
    for range ticker.C {
        // Subscription metrics
        sm.SubscriptionRequestsPerMinute = sm.getRequestCount()
        sm.FailedRequestsRate = sm.getFailedRate()
        sm.BlockedIPsCount = sm.getBlockedIPCount()
        
        // Проверка пороговых значений
        sm.checkThresholds()
    }
}

func (sm *SecurityMetrics) checkThresholds() {
    // Failed requests > 100/min
    if sm.SubscriptionRequestsPerMinute > 100 && sm.FailedRequestsRate > 0.5 {
        log.Warn().
            Int64("requests", sm.SubscriptionRequestsPerMinute).
            Float64("failed_rate", sm.FailedRequestsRate).
            Msg("High failed request rate detected")
    }
    
    // Blocked IPs > 50
    if sm.BlockedIPsCount > 50 {
        log.Warn().
            Int64("blocked_ips", sm.BlockedIPsCount).
            Msg("High number of blocked IPs")
    }
    
    // Response time > 1s
    if sm.AverageResponseTime > 1*time.Second {
        log.Warn().
            Dur("avg_response_time", sm.AverageResponseTime).
            Msg("High response time detected")
    }
}
```

### Алерты

**Уровни алертов:**
- **INFO**: Обычная активность
- **WARNING**: Подозрительная активность
- **ERROR**: Атака в процессе
- **CRITICAL**: Сервис недоступен

**Примеры:**
```go
// WARNING: Высокий процент неудачных запросов
if failedRate > 0.5 {
    log.Warn().
        Float64("rate", failedRate).
        Msg("High failed request rate")
}

// ERROR: DDoS атака
if requestsPerMinute > 2000 {
    log.Error().
        Int64("rpm", requestsPerMinute).
        Msg("Possible DDoS attack")
}

// CRITICAL: Сервис перегружен
if globalRateLimit > 1000 {
    log.Error().
        Int64("count", globalRateLimit).
        Msg("Global rate limit exceeded - service unavailable")
}
```

---

## Чеклист безопасности MVP

### ✅ Обязательно реализовать:

**Панель управления:**
- [x] JWT аутентификация
- [x] Argon2id для паролей админов
- [x] Rate limiting для login (5 req/min)
- [x] HTTPS only (через SSH tunnel)
- [x] CSRF protection

**Subscription endpoints:**
- [x] IP-based rate limiting (30 req/hour)
- [x] Token-based rate limiting (10 req/hour)
- [x] Global rate limiting (1000 req/hour)
- [x] Failed request tracking
- [x] IP blocking (20 failed attempts)
- [x] User-Agent validation
- [x] Enhanced access logging
- [x] Таблица blocked_ips в БД

**Общее:**
- [x] SQLite WAL mode
- [x] Валидация всех входных данных
- [x] Параметризованные SQL запросы
- [x] Security headers (CSP, X-Frame-Options, etc.)
- [x] Логирование всех критических операций

### ⏸️ Post-MVP:

- [ ] CAPTCHA для подозрительных запросов
- [ ] Geographic restrictions
- [ ] Advanced anomaly detection
- [ ] Automatic token rotation
- [ ] Email alerts для админов
- [ ] Webhook notifications
- [ ] 2FA для админов

---

**Документ утвержден для реализации в MVP.**
