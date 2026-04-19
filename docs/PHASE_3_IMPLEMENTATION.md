# ФАЗА 3: ДОРАБОТКИ - ТЕХНИЧЕСКАЯ СПЕЦИФИКАЦИЯ

> **Дата**: 2026-04-18  
> **Статус**: Исследование завершено, готово к реализации  
> **Приоритет**: P1 (High) - Post-MVP фичи

---

## 📋 СОДЕРЖАНИЕ

1. [HWID Tracking (защита от шаринга)](#1-hwid-tracking)
2. [HAProxy / SNI-based Routing](#2-haproxy--sni-based-routing)
3. [Advanced Routing UI](#3-advanced-routing-ui)
4. [Приоритеты реализации](#4-приоритеты-реализации)
5. [Архитектурные решения](#5-архитектурные-решения)

---

## 1. HWID Tracking (защита от шаринга аккаунтов)

### 1.1 Обзор конкурентов

| Панель | HWID Support | Метод | Лимит устройств | Статус |
|--------|-------------|-------|-----------------|--------|
| **3X-UI v3.0** | ✅ Native | `X-HWID` header + metadata | `MaxHWID` configurable | Draft/Beta |
| **Marzban** | ❌ Core | IP-based only | N/A | Community add-ons |
| **Marzban + Device Limiter** | ✅ Add-on | SHA256(UA+Token) | Default 3 | Active |
| **Hiddify** | ❌ Explicitly rejected | N/A | N/A | "Not our goal" |
| **V2RayA/Clash Verge** | ❌ Client-only | N/A | N/A | N/A |

### 1.2 Рекомендуемая архитектура

**Три уровня идентификации** (graceful degradation):

```
Уровень 1 (Высокая надежность): X-HWID header от клиента
   ↓ fallback
Уровень 2 (Средняя надежность): TLS fingerprint (JA4 hash)
   ↓ fallback  
Уровень 3 (Низкая надежность): IP + User-Agent hash
```

### 1.3 Database Schema

```sql
-- Extension to existing users table
ALTER TABLE users ADD COLUMN max_devices INTEGER DEFAULT 3;
ALTER TABLE users ADD COLUMN hwid_enabled BOOLEAN DEFAULT TRUE;
ALTER TABLE users ADD COLUMN hwid_mode VARCHAR(20) DEFAULT 'client_header'; 
-- 'off', 'client_header', 'fingerprint', 'ip_based'

-- Device tracking table
CREATE TABLE user_devices (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    device_hash VARCHAR(64) NOT NULL, -- SHA256 truncated to 16 chars
    device_type VARCHAR(50), -- iOS, Android, Windows, etc.
    device_model VARCHAR(100),
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_seen_at TIMESTAMP DEFAULT NOW(),
    last_ip INET,
    is_active BOOLEAN DEFAULT TRUE,
    tls_fingerprint VARCHAR(64), -- JA4 hash if available
    user_agent TEXT,
    metadata JSONB -- extensible
);

CREATE INDEX idx_user_devices_user ON user_devices(user_id);
CREATE INDEX idx_user_devices_hash ON user_devices(device_hash);
CREATE UNIQUE INDEX idx_user_device_unique ON user_devices(user_id, device_hash);
```

### 1.4 Go Implementation

**File**: `backend/internal/hwid/tracker.go`

```go
package hwid

import (
    "crypto/sha256"
    "encoding/hex"
    "net/http"
    "strings"
    "time"
    
    "github.com/samber/lo"
    "gorm.io/gorm"
)

type Tracker struct {
    db *gorm.DB
}

type DeviceInfo struct {
    HWID           string // X-HWID header
    UserAgent      string
    TLSFingerprint string // JA4 hash
    IP             string
}

// GenerateFingerprint creates device hash using 3-level fallback
func (t *Tracker) GenerateFingerprint(info DeviceInfo) string {
    // Level 1: Client-provided HWID (most reliable)
    if info.HWID != "" {
        hash := sha256.Sum256([]byte(info.HWID))
        return "hwid_" + hex.EncodeToString(hash[:8])
    }
    
    // Level 2: TLS fingerprint (JA4)
    if info.TLSFingerprint != "" {
        // JA4 is already a hash, just truncate
        return "ja4_" + lo.Substring(info.TLSFingerprint, 0, 12)
    }
    
    // Level 3: IP + User-Agent (least reliable, may have false positives)
    data := info.UserAgent + "|" + info.IP
    hash := sha256.Sum256([]byte(data))
    return "ua_" + hex.EncodeToString(hash[:8])
}

// CheckAndRecordDevice validates device limit and records usage
func (t *Tracker) CheckAndRecordDevice(
    userID uint, 
    fingerprint string,
    info DeviceInfo,
) (allowed bool, err error) {
    // Get user config
    var user struct {
        MaxDevices   int
        HWIDEnabled  bool
    }
    if err := t.db.Raw(
        "SELECT max_devices, hwid_enabled FROM users WHERE id = ?", 
        userID,
    ).Scan(&user).Error; err != nil {
        return false, err
    }
    
    if !user.HWIDEnabled {
        return true, nil // HWID tracking disabled
    }
    
    // Upsert device
    _, err = t.db.Exec(`
        INSERT INTO user_devices 
            (user_id, device_hash, device_type, device_model, last_ip, 
             tls_fingerprint, user_agent, last_seen_at, is_active)
        VALUES (?, ?, ?, ?, ?::inet, ?, ?, NOW(), TRUE)
        ON CONFLICT (user_id, device_hash) 
        DO UPDATE SET 
            last_seen_at = NOW(),
            last_ip = EXCLUDED.last_ip,
            is_active = TRUE
    `, userID, fingerprint, detectDeviceType(info.UserAgent),
       extractModel(info.UserAgent), info.IP, info.TLSFingerprint, info.UserAgent)
    
    if err != nil {
        return false, err
    }
    
    // Count active devices
    var count int64
    if err := t.db.Raw(`
        SELECT COUNT(*) FROM user_devices 
        WHERE user_id = ? AND is_active = TRUE
    `, userID).Scan(&count).Error; err != nil {
        return false, err
    }
    
    return int(count) <= user.MaxDevices, nil
}

// CleanupInactive removes old inactive devices (GDPR compliance)
func (t *Tracker) CleanupInactive(maxAge time.Duration) error {
    return t.db.Exec(`
        UPDATE user_devices 
        SET is_active = FALSE 
        WHERE last_seen_at < NOW() - ?::interval
    `, maxAge).Error
}

func detectDeviceType(ua string) string {
    ua = strings.ToLower(ua)
    switch {
    case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
        return "iOS"
    case strings.Contains(ua, "android"):
        return "Android"
    case strings.Contains(ua, "windows"):
        return "Windows"
    case strings.Contains(ua, "macintosh"), strings.Contains(ua, "mac os"):
        return "macOS"
    case strings.Contains(ua, "linux"):
        return "Linux"
    default:
        return "Unknown"
    }
}
```

### 1.5 API Endpoints

```go
// GET /api/user/devices - List user's devices
func (h *Handler) ListDevices(c *fiber.Ctx) error {
    userID := c.Locals("userID").(uint)
    
    var devices []DeviceResponse
    h.db.Raw(`
        SELECT id, device_type, device_model, first_seen_at, 
               last_seen_at, last_ip, is_active
        FROM user_devices
        WHERE user_id = ?
        ORDER BY last_seen_at DESC
    `, userID).Scan(&devices)
    
    return c.JSON(fiber.Map{"devices": devices})
}

// DELETE /api/user/devices/:id - Revoke device
func (h *Handler) RevokeDevice(c *fiber.Ctx) error {
    userID := c.Locals("userID").(uint)
    deviceID := c.Params("id")
    
    result := h.db.Exec(`
        UPDATE user_devices SET is_active = FALSE 
        WHERE id = ? AND user_id = ?
    `, deviceID, userID)
    
    if result.RowsAffected == 0 {
        return c.Status(404).JSON(fiber.Map{"error": "device not found"})
    }
    
    return c.JSON(fiber.Map{"success": true})
}

// POST /api/admin/users/:id/devices/cleanup - Admin cleanup
func (h *Handler) AdminCleanupDevices(c *fiber.Ctx) error {
    // Admin only
    userID := c.Params("id")
    h.hwidTracker.CleanupInactive(90 * 24 * time.Hour) // 90 days
    return c.JSON(fiber.Map{"success": true})
}
```

### 1.6 Middleware Integration

**File**: `backend/internal/middleware/hwid.go`

```go
func HWIDMiddleware(tracker *hwid.Tracker) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Only check for subscription endpoints
        if !strings.HasPrefix(c.Path(), "/sub/") {
            return c.Next()
        }
        
        // Extract token from path
        token := strings.TrimPrefix(c.Path(), "/sub/")
        
        // Lookup user by subscription token
        userID, err := lookupUserBySubToken(token)
        if err != nil {
            return c.Next() // Fail open or closed based on policy
        }
        
        // Collect device info
        info := hwid.DeviceInfo{
            HWID:           c.Get("X-HWID"),
            UserAgent:      c.Get("User-Agent"),
            TLSFingerprint: extractJA4(c), // If TLS inspection enabled
            IP:             c.IP(),
        }
        
        fingerprint := tracker.GenerateFingerprint(info)
        
        allowed, err := tracker.CheckAndRecordDevice(userID, fingerprint, info)
        if err != nil {
            log.Error().Err(err).Msg("HWID check failed")
            return c.Next() // Fail open
        }
        
        if !allowed {
            return c.Status(403).JSON(fiber.Map{
                "error": "Device limit exceeded. Please revoke unused devices.",
                "max_devices": getUserMaxDevices(userID),
                "current_devices": getUserDeviceCount(userID),
            })
        }
        
        return c.Next()
    }
}
```

### 1.7 Privacy & GDPR Compliance

**Обязательные требования**:

1. **Data Minimization**:
   - Хранить только hash (8-16 chars), не raw HWID
   - Auto-expire после 90 дней неактивности
   - Пользователь может revoke устройства самостоятельно

2. **Transparency**:
   - Privacy policy disclosure
   - UI показывает активные устройства
   - Email notification при добавлении нового устройства (optional)

3. **User Control**:
   - Self-service device management UI
   - "Revoke all devices" button
   - Whitelist trusted devices

4. **Security**:
   - Per-user salt для hash (prevents cross-service tracking)
   - Rate limiting на device registration
   - Audit log всех device events

### 1.8 Рекомендации по внедрению

**Phase 1**: Базовая реализация
- [ ] DB schema migration
- [ ] Tracker service
- [ ] Middleware для subscription endpoints
- [ ] API endpoints (list/revoke)

**Phase 2**: UI интеграция  
- [ ] Device management page
- [ ] Real-time device count display
- [ ] Revoke confirmation dialogs

**Phase 3**: Advanced features
- [ ] JA4 TLS fingerprinting (if using reverse proxy)
- [ ] Email notifications
- [ ] Device geolocation (country-level)

**Оценка времени**: 3-4 недели (frontend + backend)

---

## 2. HAProxy / SNI-based Routing + МУЛЬТИ-ПОРТ

> **Детальный план реализации**: [HAPROXY_IMPLEMENTATION_PLAN.md](./HAPROXY_IMPLEMENTATION_PLAN.md)  
> **Вердикт**: ✅ **GO** — Мульти-порт + кросс-ядерная маршрутизация + Smart Warning UI
> **Примечание**: Версия 3.0 включает **динамическую систему назначения портов** (BackendPort) вместо фиксированных диапазонов (10001-10010, 9090-9099, 9091-9099)

### 2.0 Ключевые возможности (Consolidated Plan v3.0)

**Все требования реализуемы**:
- ✅ **Мульти-инбаунд на один порт** (любой порт, не только 443)
- ✅ **Кросс-ядерная маршрутизация** (Xray/Sing-box/Mihomo на одном порту)
- ✅ **Smart Warning UI валидация** (Severity: Info/Warning/Error)
- ✅ **Динамическое назначение портов** (BackendPort assignment system)
- ✅ **Полные таблицы совместимости** (25+ протоколов для 3 ядер)
- ⚠️ UDP/QUIC — direct ports (не через HAProxy)

### 2.1 Быстрые ссылки

| Документация | Описание |
|--------------|----------|
| **Protocol Tables** | Полные таблицы совместимости (Xray/Sing-box/Mihomo) |
| **Core Domain Models** | Go structs с динамическими портами |
| **Code Examples** | Шаблоны HAProxy, Manager, Docker Compose |
| **Smart Warning UI** | Валидация портов с severity levels |
| **6-Week Roadmap** | План реализации по неделям |

### 2.2 Критические ограничения (Summary)

| Ядро | Ограничение | Влияние | Решение |
|------|-------------|---------|---------|
| **Xray** | gRPC + PROXY protocol v2 ([Issue #2204](https://github.com/XTLS/Xray-core/issues/2204)) | 🟡 Среднее | Использовать XHTTP/WS; gRPC без PROXY v2 |
| **Sing-box** | **PROXY protocol REMOVED в 1.6.0+** ([deprecated](https://sing-box.sagernet.org/deprecated/)) | 🟡 Среднее | Только X-Forwarded-For headers |
| **HAProxy** | Нет UDP/QUIC forwarding ([Issue #2748](https://github.com/haproxy/haproxy/issues/2748)) | 🟡 Среднее | Direct ports для Hysteria2/TUIC/KCP |
| **Mihomo** | Нет PROXY protocol v2 | 🟢 Низкое | Только X-Forwarded-For headers |

**Вывод**: Real client IP передаётся только для Xray (TCP-based через PROXY v2). Sing-box и Mihomo — только X-Forwarded-For.

### 2.3 Архитектура: Динамические порты + Кросс-ядро

**Один порт 443, разные ядра (с динамическими BackendPort):**
```
Internet → HAProxy :443 (SNI inspection)
              │
              ├── SNI: vless.example.com ──→ Backend: xray_vless (:{{.XrayBackendPort}})
              ├── SNI: vmess.example.com ──→ Backend: singbox_vmess (:{{.SingboxBackendPort}})  
              ├── SNI: trojan.example.com ──→ Backend: mihomo_trojan (:{{.MihomoBackendPort}})
              └── default ──→ Backend: panel_web (:{{.PanelPort}})
```

**Мульти-порт конфигурация (динамические порты):**
```
HAProxy :{{.MainPort}}   → SNI routing → Все 3 ядра
HAProxy :{{.AltPort}}    → SNI routing → Все 3 ядра (доп. инбаунды)  
HAProxy :{{.HTTPPort}}   → HTTP redirect → Redirect to HTTPS
Direct UDP ports         → Hysteria2/TUIC (Sing-box/Mihomo) - auto-assigned
```

> **Полная документация**: См. [HAPROXY_IMPLEMENTATION_PLAN.md](./HAPROXY_IMPLEMENTATION_PLAN.md) для:
> - Полных таблиц протоколов (25+ протоколов)
> - Go structs с динамическими портами
> - Примеров кода (шаблоны, Manager, Docker)
> - Smart Warning UI спецификации
> - 6-недельного плана реализации

### 2.3 Сравнение решений

| Решение | Производительность | Простота | SNI | Мульти-порт | Cross-core | Для нашего проекта |
|---------|---------------------|----------|-----|-------------|------------|-------------------|
| **HAProxy** | ⭐⭐⭐⭐⭐ 100K+ conn/s | ⭐⭐⭐ | ✅ | ✅ Динамич. | ✅ | **✅ Рекомендуется** |
| **NGINX Stream** | ⭐⭐⭐⭐ 80K+ conn/s | ⭐⭐⭐⭐ | ✅ | ✅ | ✅ | Альтернатива |
| **Traefik** | ⭐⭐⭐ 20K+ conn/s | ⭐⭐⭐⭐⭐ | ✅ | ✅ | ✅ | Cloud-native |
| **Envoy** | ⭐⭐⭐⭐ 85K conn/s | ⭐⭐ | ✅ | ✅ | ✅ | Enterprise |

**Выбор HAProxy обоснован**:
1. **Лучшая производительность** (100K+ conn/s, -40% RAM vs NGINX)
2. **Runtime API** для динамического управления backend
3. **Hitless reload** — zero-downtime конфигурация
4. **SNI Routing** — нативная `req.ssl_sni` без хаков
5. **Доказанная стабильность** — 20+ лет production (GitHub, Reddit, StackOverflow)

### 2.4 Smart Warning UI Валидация

**User Story**: При выборе порта в UI пользователь видит уведомление:
- ✅ **Port free**: "Порт свободен" (зелёный)
- ⚠️ **Port in use, HAProxy compatible**: "HAProxy может обеспечить совместную работу через SNI/Path routing" (жёлтый, требует подтверждения)
- ❌ **Port in use, incompatible**: "Протоколы несовместимы. Выберите другой порт." (красный, блокировка)

**Алгоритм severity:**

```go
func (v *PortValidator) ValidatePortConflict(port int, protocol, transport, coreType string) *PortConflictCheck {
    result := &PortConflictCheck{Port: port}
    
    // Check if new inbound supports HAProxy
    result.HaproxyCompatible = v.isHaproxyCompatible(protocol, transport, coreType)
    
    // Find conflicts
    for _, existing := range existingInbounds {
        if !isPortOverlap(port, existing.Port) {
            continue
        }
        
        // Can they share via HAProxy?
        existingCompatible := v.isHaproxyCompatible(existing.Protocol, existing.Transport, existing.CoreType)
        if result.HaproxyCompatible && existingCompatible {
            result.CanSharePort = true
            
            // Determine sharing mechanism
            if v.supportsSNI(protocol) && v.supportsSNI(existing.Protocol) {
                result.SharingMechanism = "sni"
            } else if v.supportsPath(transport) && v.supportsPath(existing.Transport) {
                result.SharingMechanism = "path"
            }
        }
    }
    
    // Determine severity
    if len(result.Conflicts) == 0 {
        result.Severity = "info"
        result.Message = "✓ Порт свободен"
        result.Action = "allow"
    } else if result.CanSharePort {
        result.Severity = "warning"
        result.Message = fmt.Sprintf(
            "⚠ Порт %d уже используется. HAProxy может обеспечить совместную работу через %s-based routing.",
            port, result.SharingMechanism,
        )
        result.Action = "confirm"
    } else {
        result.Severity = "error"
        result.Message = fmt.Sprintf(
            "✗ Порт %d уже используется. Протоколы несовместимы для совместной работы через HAProxy.",
            port,
        )
        result.Action = "block"
    }
    
    return result
}
```

**Frontend (Preact):**
```tsx
// PortValidationField.tsx
const [validation, setValidation] = useState({
    status: 'idle', // 'idle' | 'checking' | 'success' | 'warning' | 'error'
    message: '',
    action: 'allow', // 'allow' | 'confirm' | 'block'
});

// Debounced API call (500ms)
const validatePort = debounce(async (port: number) => {
    const result = await fetch('/api/inbounds/check-port', {
        method: 'POST',
        body: JSON.stringify({ port, protocol, transport, coreType }),
    }).then(r => r.json());
    
    setValidation({
        status: result.severity === 'error' ? 'error' : 
               result.severity === 'warning' ? 'warning' : 'success',
        message: result.message,
        action: result.action,
    });
}, 500);
```

**Severity Levels:**

| Severity | Icon | Color | Сообщение | Действие |
|----------|------|-------|-----------|----------|
| **Info** | ✓ | 🟢 Зелёный | "Порт свободен" | ✅ Allow |
| **Warning** | ⚠️ | 🟡 Жёлтый | "HAProxy может обеспечить совместную работу" | ⚠️ Confirm modal |
| **Error** | ✗ | 🔴 Красный | "Протоколы несовместимы. Выберите другой порт." | ❌ Block |

### 2.5 Рекомендуемая архитектура: HAProxy

**Почему HAProxy**:
1. **Производительность**: Лучший для L4 routing (100K+ conn/s)
2. **Стабильность**: 20+ лет production (GitHub, Reddit, StackOverflow)
3. **Низкое потребление**: 15-30MB RAM idle
4. **SNI Routing**: Нативная поддержка `req.ssl_sni`
5. **PROXY Protocol**: Передача реального IP в Xray/Sing-box

### 2.3 Архитектура потока трафика

```
                      ┌─────────────────────────────────────┐
                      │  HAProxy (Port 443) - SNI Router    │
                      │  ├── TLS Inspection (delay 5s)    │
                      │  ├── SNI Routing (req.ssl_sni)      │
                      │  ├── ALPN Detection (h2/h3)         │
                      │  └── Rate Limiting (DDoS)           │
                      └─────────────────────────────────────┘
                                         │
            ┌────────────────────────────┼────────────────────────────┐
            │                            │                            │
            ▼                            ▼                            ▼
    ┌───────────────┐          ┌───────────────┐          ┌───────────────┐
    │  Xray Core    │          │  Sing-box     │          │   Go Panel    │
    │  (VLESS/      │          │  (Reality,    │          │   (Admin UI,  │
    │   VMess,      │          │   Hysteria)   │          │   Fallback)   │
    │   Trojan)     │          │               │          │               │
    │  Port: 10001  │          │  Port: 10002  │          │  Port: 9000   │
    └───────────────┘          └───────────────┘          └───────────────┘
```

### 2.4 Конфигурация HAProxy

**File**: `docker/haproxy/haproxy.cfg` (генерируемый)

```haproxy
global
    maxconn 50000
    daemon
    nbproc 1
    nbthread 4
    cpu-map auto:1/1-4 0-3

defaults
    mode tcp
    timeout connect 5s
    timeout client 50s
    timeout server 50s
    option tcplog
    option dontlognull

# Stats page (optional)
listen stats
    bind :8404
    mode http
    stats enable
    stats uri /stats
    stats refresh 10s

# Main HTTPS frontend with SNI inspection
frontend https-in
    bind :443 v4v6 tfo
    
    # Inspect TLS ClientHello to extract SNI
    tcp-request inspect-delay 5s
    tcp-request content accept if { req.ssl_hello_type 1 }
    
    # Connection rate limiting (DDoS protection)
    stick-table type ip size 100k expire 30s store conn_rate(10s),conn_cur
    tcp-request connection track-sc0 src
    tcp-request connection reject if { sc_conn_rate gt 100 }
    tcp-request connection reject if { sc_conn_cur gt 50 }
    
    # SNI-based routing
    # VLESS on vless.example.com
    acl is_vless req.ssl_sni -i vless.example.com
    use_backend vless_backend if is_vless
    
    # VMess on vmess.example.com
    acl is_vmess req.ssl_sni -i vmess.example.com
    use_backend vmess_backend if is_vmess
    
    # Trojan on trojan.example.com
    acl is_trojan req.ssl_sni -i trojan.example.com
    use_backend trojan_backend if is_trojan
    
    # Reality protocol (XTLS Vision) on reality.example.com
    acl is_reality req.ssl_sni -i reality.example.com
    use_backend reality_backend if is_reality
    
    # Sing-box protocols
    acl is_shadowtls req.ssl_sni -i shadowtls.example.com
    use_backend shadowtls_backend if is_shadowtls
    
    # Web fallback (decoy website)
    default_backend web_backend

# Xray backends with PROXY protocol
backend vless_backend
    server vless 127.0.0.1:10001 send-proxy-v2 check inter 5s rise 2 fall 3

backend vmess_backend
    server vmess 127.0.0.1:10001 send-proxy-v2 check inter 5s rise 2 fall 3
    # Note: Same Xray instance, different inbounds

backend trojan_backend
    server trojan 127.0.0.1:10002 send-proxy-v2 check inter 5s rise 2 fall 3

backend reality_backend
    server reality 127.0.0.1:10003 send-proxy-v2 check inter 5s rise 2 fall 3

# Sing-box backend
backend shadowtls_backend
    server shadowtls 127.0.0.1:10004 send-proxy-v2 check inter 5s rise 2 fall 3

# Web backend (Nginx/Go panel)
backend web_backend
    server web 127.0.0.1:8080 check inter 5s rise 2 fall 3
```

### 2.5 Интеграция с панелью

**Service**: `backend/internal/haproxy/manager.go`

```go
package haproxy

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    "text/template"
)

type Config struct {
    Domains map[string]DomainConfig
    Cores   map[string]CoreBackend
}

type DomainConfig struct {
    Domain    string
    Protocol  string // vless, vmess, trojan, reality, shadowtls
    CoreType  string // xray, singbox
    Port      int
}

type CoreBackend struct {
    Type   string
    Port   int
    Health bool
}

type Manager struct {
    configPath string
    tmpl       *template.Template
}

func NewManager(configPath string) (*Manager, error) {
    tmpl, err := template.ParseFiles("templates/haproxy.cfg.tmpl")
    if err != nil {
        return nil, err
    }
    
    return &Manager{
        configPath: configPath,
        tmpl:       tmpl,
    }, nil
}

func (m *Manager) GenerateConfig(cfg Config) error {
    f, err := os.Create(m.configPath + ".new")
    if err != nil {
        return err
    }
    defer f.Close()
    
    if err := m.tmpl.Execute(f, cfg); err != nil {
        return err
    }
    
    // Validate config
    if err := m.ValidateConfig(m.configPath + ".new"); err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }
    
    // Atomic replace
    return os.Rename(m.configPath+".new", m.configPath)
}

func (m *Manager) ValidateConfig(path string) error {
    cmd := exec.Command("haproxy", "-c", "-f", path)
    return cmd.Run()
}

func (m *Manager) Reload() error {
    // Try graceful reload first
    cmd := exec.Command("systemctl", "reload", "haproxy")
    if err := cmd.Run(); err != nil {
        // Fallback to kill -USR2
        pid, _ := os.ReadFile("/var/run/haproxy.pid")
        cmd = exec.Command("kill", "-USR2", strings.TrimSpace(string(pid)))
        return cmd.Run()
    }
    return nil
}
```

### 2.6 Xray Core Configuration with PROXY Protocol

**Generated config**: `/app/data/cores/xray/config.json`

```json
{
  "inbounds": [
    {
      "tag": "vless-tcp",
      "port": 10001,
      "listen": "127.0.0.1",
      "protocol": "vless",
      "settings": {
        "clients": [],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "tls",
        "tlsSettings": {
          "certificates": [
            {
              "certificateFile": "/app/data/certs/fullchain.pem",
              "keyFile": "/app/data/certs/privkey.pem"
            }
          ]
        }
      },
      "sniffing": {
        "enabled": true,
        "destOverride": ["http", "tls"]
      },
      "allocate": {
        "strategy": "always"
      },
      "proxyProtocol": 2  // PROXY protocol v2
    }
  ]
}
```

### 2.7 Docker Compose Integration

**File**: `docker/docker-compose.yml` (additions)

```yaml
services:
  haproxy:
    image: haproxy:2.8-alpine
    container_name: isolate-haproxy
    restart: unless-stopped
    network_mode: host  # Required for PROXY protocol
    volumes:
      - ./haproxy/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
      - ./haproxy/certs:/etc/certs:ro
      - ./haproxy/iplists:/etc/haproxy/iplists:ro
      - haproxy-socket:/var/run/haproxy
    depends_on:
      - isolate-panel
      - xray
      - singbox
    healthcheck:
      test: ["CMD", "haproxy", "-c", "-f", "/usr/local/etc/haproxy/haproxy.cfg"]
      interval: 30s
      timeout: 10s
      retries: 3
    labels:
      - "com.isolate.role=haproxy"

  # Xray with multiple inbounds
  xray:
    image: ghcr.io/xtls/xray-core:latest
    container_name: isolate-xray
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./data/cores/xray:/app/data/cores/xray:rw
      - ./data/certs:/app/data/certs:ro
    command: ["run", "-config=/app/data/cores/xray/config.json"]
    
  # Sing-box
  singbox:
    image: ghcr.io/sagernet/sing-box:latest
    container_name: isolate-singbox
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./data/cores/singbox:/app/data/cores/singbox:rw
    command: ["run", "-c", "/app/data/cores/singbox/config.json"]

volumes:
  haproxy-socket:
```

### 2.8 Альтернатива: NGINX Stream (если HAProxy не подходит)

```nginx
# /etc/nginx/nginx.conf
stream {
    # Map SNI to upstream
    map $ssl_preread_server_name $backend {
        vless.example.com     xray_backend;
        vmess.example.com     xray_backend;
        trojan.example.com    xray_backend;
        reality.example.com   xray_reality_backend;
        shadowtls.example.com singbox_backend;
        default               web_backend;
    }
    
    upstream xray_backend {
        server 127.0.0.1:10001;
    }
    
    upstream xray_reality_backend {
        server 127.0.0.1:10003;
    }
    
    upstream singbox_backend {
        server 127.0.0.1:10004;
    }
    
    upstream web_backend {
        server 127.0.0.1:8080;
    }
    
    server {
        listen 443 reuseport;
        ssl_preread on;
        proxy_pass $backend;
        proxy_connect_timeout 5s;
        proxy_timeout 50s;
    }
}

# HTTP server for web UI
http {
    server {
        listen 8080;
        server_name panel.example.com;
        # ... regular HTTP config
    }
}
```

### 2.9 Рекомендации по внедрению (6 недель)

**Неделя 1-2**: Core HAProxy Infrastructure
- [ ] Domain models: `PortGroup`, `InboundPortBinding`
- [ ] Database migrations для мульти-порт
- [ ] Template engine: динамическая генерация frontend per port
- [ ] Manager: generate, validate, reload
- [ ] Runtime API client

**Неделя 3**: Multi-Port + Cross-Core Routing
- [ ] Dynamic frontend generation для любых портов
- [ ] SNI-based routing на все 3 ядра
- [ ] Path-based routing для HTTP транспортов
- [ ] Backend health checks

**Неделя 4**: Core Integration + Docker
- [ ] Xray: PROXY v2 (TCP/WS only, НЕ gRPC)
- [ ] Sing-box: X-Forwarded-For (НЕТ PROXY v2)
- [ ] Mihomo: X-Forwarded-For (НЕТ PROXY v2)
- [ ] Docker Compose: `network_mode: host`

**Неделя 5**: Smart Warning UI
- [ ] `PortValidator` с severity levels
- [ ] API endpoint: `POST /api/inbounds/check-port`
- [ ] Frontend `PortValidationField` компонент
- [ ] Real-time validation с debounce (500ms)
- [ ] Modal confirmation для warning state

**Неделя 6**: Testing & Documentation
- [ ] Integration tests: SNI routing, multi-port
- [ ] Cross-core routing tests
- [ ] UI validation tests (e2e)
- [ ] Prometheus metrics
- [ ] Grafana dashboard

**Оценка времени**: 6 недель

---

## 3. Advanced Routing UI

### 3.1 Анализ UI паттернов

| Панель | Rule Builder | Визуализация | Drag-drop | Testing | Редактор |
|--------|-------------|--------------|-----------|---------|----------|
| **3X-UI** | JSON only | ❌ | ❌ | ❌ | Monaco |
| **Hiddify** | Presets only | ❌ | ❌ | ❌ | JSON |
| **Clash Verge** | Таблица | ⚠️ List | ❌ | ❌ | Monaco |
| **V2RayA** | RoutingA DSL | ❌ | ❌ | ❌ | Monaco |
| **XrayUI** | Forms + autocomplete | ❌ | ✅ | ❌ | Raw |
| **DDS-Xray-Routing** | Vue Table | ❌ | ✅ | ❌ | JSON |
| **Proxy Rule Manager** | ✅ Sankey | ✅ Sankey | ✅ | ❌ | ✅ |

### 3.2 Рекомендуемая архитектура UI

**Гибридный подход** (best of both worlds):

```
┌─────────────────────────────────────────────────────────────┐
│                    Routing Rules Manager                       │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Presets    │  │ Visual Builder│  │ Code Editor │       │
│  │   (Beginner) │  │ (Intermediate)│  │  (Expert)    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┬─────────────────────────────────────┐    │
│  │                 │                                     │    │
│  │  Rule List      │     Rule Editor / Sankey Flow      │    │
│  │  (drag-drop)    │                                     │    │
│  │                 │  ┌─────────────────────────────┐   │    │
│  │  1. Netflix →   │  │    Traffic Sankey Diagram    │   │    │
│  │     US Proxy    │  │                             │   │    │
│  │                 │  │   ┌─────┐    ┌─────┐        │   │    │
│  │  2. Google →    │  │   │Client│───▶│ Rule │───▶│   │    │
│  │     Direct      │  │   └─────┘    └─────┘        │   │    │
│  │                 │  │                      │        │   │    │
│  │  3. CN Sites →  │  │                      ▼        │   │    │
│  │     Direct      │  │                   ┌─────┐     │   │    │
│  │                 │  │                   │Proxy│     │   │    │
│  │  4. * →         │  │                   └─────┘     │   │    │
│  │     Default     │  └─────────────────────────────┘   │    │
│  │                 │                                     │    │
│  │  [+ Add Rule]   │  [Test Domain] [google.com] [Test] │    │
│  └─────────────────┴─────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 Component Architecture (Preact)

**Directory structure**:
```
src/
├── components/
│   ├── routing/
│   │   ├── RuleBuilder.tsx          # Main container
│   │   ├── RuleList.tsx             # Drag-drop list
│   │   ├── RuleEditor.tsx           # Form-based editor
│   │   ├── RuleTester.tsx           # Live testing
│   │   ├── RuleSankey.tsx           # Sankey diagram
│   │   ├── GeoSelector.tsx          # GeoIP/GeoSite picker
│   │   └── BalancerConfig.tsx       # Load balancer UI
│   ├── shared/
│   │   ├── MonacoEditor.tsx         # Code editor
│   │   ├── ConditionBuilder.tsx     # AND/OR conditions
│   │   └── AutocompleteInput.tsx    # Geo suggestions
│   └── visualization/
│       └── SankeyDiagram.tsx        # Reusable Sankey
```

### 3.4 Rule Types Support

**Domain-based rules**:
```typescript
interface DomainRule {
  type: 'domain' | 'full' | 'keyword' | 'regex' | 'wildcard';
  value: string;
  caseSensitive?: boolean;
}

// Examples:
// { type: 'full', value: 'google.com' }      → Exact match
// { type: 'domain', value: 'google.com' }    → google.com + *.google.com
// { type: 'keyword', value: 'google' }         → *google*
// { type: 'regex', value: '.*\\.ru$' }        → Regex
// { type: 'wildcard', value: '*.google.*' }  → Glob pattern
```

**IP-based rules**:
```typescript
interface IPRule {
  type: 'cidr' | 'geoip' | 'private';
  value: string; // CIDR or country code
  invert?: boolean; // NOT matching
}

// Examples:
// { type: 'geoip', value: 'CN' }             → GeoIP China
// { type: 'cidr', value: '192.168.0.0/16' }  → CIDR range
// { type: 'private' }                        → Private IP ranges
```

**Advanced rules**:
```typescript
interface AdvancedRule {
  type: 'port' | 'protocol' | 'process' | 'source';
  operator: 'eq' | 'ne' | 'in' | 'range';
  value: string | number | string[];
}

// Examples:
// { type: 'port', operator: 'in', value: [80, 443] }
// { type: 'protocol', operator: 'eq', value: 'tls' }
// { type: 'process', operator: 'eq', value: 'chrome.exe' }
```

### 3.5 Rule Builder UI Component

```tsx
// RuleBuilder.tsx - Form-based visual builder
export function RuleBuilder({ onSave, initialRule }: Props) {
  const [conditions, setConditions] = useState<Condition[]>([]);
  const [action, setAction] = useState<Action>('proxy');
  const [target, setTarget] = useState<string>('');

  return (
    <div class="rule-builder">
      <h3>Match Conditions</h3>
      
      {/* Condition list with AND/OR logic */}
      <ConditionBuilder
        conditions={conditions}
        onChange={setConditions}
        supportedTypes={['domain', 'ip', 'port', 'protocol']}
        autocompleteSource={geoDatabase}
      />
      
      <h3>Action</h3>
      <div class="action-selector">
        <select value={action} onChange={e => setAction(e.target.value)}>
          <option value="proxy">Route to Proxy</option>
          <option value="direct">Direct Connection</option>
          <option value="block">Block</option>
          <option value="balancer">Load Balancer</option>
        </select>
        
        {action === 'proxy' && (
          <ProxySelector value={target} onChange={setTarget} />
        )}
        
        {action === 'balancer' && (
          <BalancerSelector value={target} onChange={setTarget} />
        )}
      </div>
      
      <div class="rule-preview">
        <h4>Preview</h4>
        <pre>{generateRulePreview(conditions, action, target)}</pre>
      </div>
      
      <button onClick={() => onSave(buildRule())}>Save Rule</button>
      <button onClick={testRule}>Test Rule</button>
    </div>
  );
}
```

### 3.6 Sankey Diagram Component

```tsx
// RuleSankey.tsx - Traffic flow visualization
import { useEffect, useRef } from 'preact/hooks';
import * as d3 from 'd3';
import { sankey, sankeyLinkHorizontal } from 'd3-sankey';

export function RuleSankey({ rules, trafficData }: Props) {
  const svgRef = useRef<SVGSVGElement>(null);
  
  useEffect(() => {
    if (!svgRef.current || !trafficData) return;
    
    // Transform rules and traffic into sankey data
    const nodes = [
      { name: 'Client', category: 'source' },
      ...rules.map((r, i) => ({ 
        name: r.name || `Rule ${i + 1}`, 
        category: 'rule' 
      })),
      { name: 'Direct', category: 'target' },
      { name: 'Proxy', category: 'target' },
      { name: 'Block', category: 'target' },
    ];
    
    const links = trafficData.map(t => ({
      source: 0, // Client
      target: t.ruleIndex + 1,
      value: t.bytes,
    }));
    
    // Create Sankey layout
    const layout = sankey()
      .nodeWidth(20)
      .nodePadding(20)
      .extent([[0, 0], [800, 400]]);
    
    const { nodes: sankeyNodes, links: sankeyLinks } = layout({
      nodes: nodes.map(d => ({ ...d })),
      links: links.map(d => ({ ...d })),
    });
    
    // Render with D3
    const svg = d3.select(svgRef.current);
    // ... rendering code
  }, [rules, trafficData]);
  
  return <svg ref={svgRef} width={800} height={400} />;
}
```

### 3.7 Rule Tester Component

```tsx
// RuleTester.tsx - Live rule testing
export function RuleTester({ rules }: Props) {
  const [testInput, setTestInput] = useState('');
  const [results, setResults] = useState<TestResult[]>([]);

  const runTest = async () => {
    // Simulate matching
    const matches = rules.map((rule, index) => {
      const matched = matchRule(rule, testInput);
      return {
        ruleIndex: index,
        ruleName: rule.name,
        matched,
        action: rule.action,
        latency: matched ? Math.random() * 100 : null,
      };
    });
    
    setResults(matches);
  };

  return (
    <div class="rule-tester">
      <input
        type="text"
        placeholder="Enter domain or IP (e.g., google.com)"
        value={testInput}
        onChange={e => setTestInput(e.target.value)}
      />
      <button onClick={runTest}>Test</button>
      
      <div class="results">
        {results.map((r, i) => (
          <div key={i} class={`result ${r.matched ? 'matched' : ''}`}>
            <span class="status">{r.matched ? '✓' : '✗'}</span>
            <span class="rule">{r.ruleName}</span>
            {r.matched && (
              <span class="action">→ {r.action} ({r.latency.toFixed(0)}ms)</span>
            )}
          </div>
        ))}
      </div>
      
      {/* Visual flow */}
      <div class="flow-visualization">
        <div class="node">Client</div>
        <div class="arrow">→</div>
        {results.find(r => r.matched) ? (
          <>
            <div class="node matched">
              {results.find(r => r.matched).ruleName}
            </div>
            <div class="arrow">→</div>
            <div class="node action">
              {results.find(r => r.matched).action}
            </div>
          </>
        ) : (
          <div class="node">Default Route</div>
        )}
      </div>
    </div>
  );
}
```

### 3.8 Recommended Libraries

| Component | Library | Size | Notes |
|-----------|---------|------|-------|
| Drag-drop | `@dnd-kit/core` | ~15KB | Modern, accessible |
| Sankey | `d3-sankey` + `react-sankey` | ~40KB | Industry standard |
| Monaco | `@monaco-editor/react` | Lazy | Full editor features |
| Forms | `react-hook-form` | ~10KB | Performance |
| Virtual list | `react-virtual` | ~5KB | For 1000+ rules |
| Icons | `lucide-react` | Tree-shake | Clean design |

### 3.9 Рекомендации по внедрению

**Phase 1**: Базовый rule builder
- [ ] Rule list with drag-drop priority
- [ ] Form-based rule editor
- [ ] Domain/IP/Port conditions
- [ ] Basic action selector (proxy/direct/block)

**Phase 2**: Advanced features
- [ ] GeoIP/GeoSite autocomplete
- [ ] Load balancer configuration
- [ ] Rule tester with live preview
- [ ] Monaco editor for power users

**Phase 3**: Визуализация
- [ ] Sankey traffic flow diagram
- [ ] World map with ASN visualization
- [ ] Real-time traffic animation
- [ ] Rule performance metrics

**Оценка времени**: 5-7 недель

---

## 4. Приоритеты реализации

### P1: HAProxy / SNI Routing + Мульти-порт + Smart Warning
**Приоритет**: 🔴 Высокий  
**Обоснование**: 
- **Мульти-порт**: Не только 443, а любые порты с мульти-инбаундами
- **Кросс-ядро**: Один порт → все 3 ядра (Xray/Sing-box/Mihomo)
- **Smart Warning**: UI валидация при выборе порта (ключевое UX улучшение)
- Ключевая дифференцирующая фича (Hiddify имеет базовый HAProxy, но без Smart Warning)
- Решает проблему множества портов (упрощает firewall)
- Позволяет маскировать под обычный HTTPS

**Зависимости**: CoreAdapter ✅ (готово)  
**Время**: 6 недель (включая Smart Warning UI)

### P2: HWID Tracking
**Приоритет**: 🟡 Средний  
**Обоснование**:
- 3X-UI v3.0 продвигает как ключевую фичу
- Защита от abuse (sharing аккаунтов)
- Privacy concerns требуют careful implementation
- Не все пользователи хотят (Hiddify отказались)

**Зависимости**: Subscriptions API, User management  
**Время**: 3-4 недели

### P3: Advanced Routing UI
**Приоритет**: 🟢 Низкий  
**Обоснование**:
- Улучшает UX, но не критично для MVP
- Можно обойтись JSON editor в начале
- Визуализация требует данных (нужен traffic collector)

**Зависимости**: Traffic metrics, API endpoints  
**Время**: 5-7 недель

### Recommended Order
```
Week 1-6:  HAProxy SNI Routing (P1)
Week 7-10: HWID Tracking (P2)  
Week 11-17: Advanced Routing UI (P3)
```

---

## 5. Архитектурные решения

### 5.1 HAProxy vs NGINX (После детального исследования)

**Выбор**: ✅ **HAProxy с гибридной архитектурой**

**Причины**:
1. **Производительность**: 100K+ conn/s, -40% RAM vs NGINX (170MB vs 275MB на 10K conn)
2. **Runtime API**: Go клиент для динамического управления бэкендами
3. **SNI Routing**: Нативная `req.ssl_sni` без хаков
4. **PROXY Protocol v2**: Лучшая поддержка передачи реального IP
5. **Hitless Reload**: Zero-downtime через Unix socket transfer
6. **Stability**: 20+ лет production use (GitHub, Reddit, StackOverflow)

**Критические ограничения найдены**:
| Проблема | Влияние | Обходной путь |
|----------|---------|---------------|
| gRPC + PROXY protocol не работает (Xray #2204) | 🟡 Среднее | Использовать XHTTP вместо gRPC |
| QUIC/Hysteria2 не проксируется (HAProxy #2748) | 🟡 Среднее | Host networking для QUIC |
| Нет Auto HTTPS | 🟢 Низкое | Certbot или Caddy sidecar |

**Архитектура**:
```
HAProxy (Port 443) → 95% трафика (TCP/WS/XHTTP/Reality)
Direct (Port 8443) → QUIC/Hysteria2, gRPC API
```

**Детальный план**: [HAPROXY_IMPLEMENTATION_PLAN.md](./HAPROXY_IMPLEMENTATION_PLAN.md) — Consolidated v3.0
- Динамическое распределение портов (Backend Port Pool)
- Полные таблицы протоколов (25+ для Xray/Sing-box/Mihomo)
- Кросс-ядерная маршрутизация
- Smart Warning UI спецификация
- 6-недельный план реализации

### 5.2 HWID Hash Algorithm

**Выбор**: SHA256 с truncation до 8-16 chars

**Причины**:
1. **Privacy**: Не храним raw HWID (GDPR-compliant)
2. **Uniqueness**: 8 hex chars = 4B combinations (достаточно для устройств)
3. **Performance**: SHA256 fast в Go
4. **Security**: Per-user salt предотвращает cross-service tracking

**Trade-off**: Нельзя восстановить оригинал (by design)

### 5.3 Routing UI Pattern

**Выбор**: Гибрид (Visual Builder + Monaco Editor)

**Причины**:
1. **Accessibility**: 80% пользователей не хотят изучать JSON
2. **Power Users**: Monaco editor для сложных случаев
3. **Maintenance**: Один компонент для всех паттернов

**Trade-off**: Двойная работа (form validation + schema validation)

---

## 6. Риски и mitigation

| Риск | Вероятность | Влияние | Mitigation |
|------|-------------|---------|------------|
| **HAProxy config syntax errors** | Средняя | Высокое | Validation перед reload, rollback |
| **HWID false positives** | Средняя | Среднее | Graceful degradation, user revocation |
| **Sankey performance (1000+ rules)** | Низкая | Среднее | Virtual rendering, pagination |
| **GDPR complaints** | Низкая | Высокое | Privacy policy, auto-expiry, user control |
| **TLS fingerprinting reliability** | Средняя | Среднее | Fallback levels, не использовать как primary |

---

## 7. Итоговый план реализации (Объединённый)

### 7.1 Roadmap и зависимости

```
[CoreAdapter] ───────────────────────────────────────────────┐
      (✅ Завершенно в Фазе 2)                                  │
                                                                 ▼
Неделя 1-2: HAProxy Core Infrastructure                         │
├── Domain models: PortGroup, InboundPortBinding, PortConflictCheck
├── Database migrations для мульти-порт                          │
├── Template engine: динамическая генерация frontend per port  │
├── Manager: generate, validate, reload (hitless)            │
└── Runtime API client: Data Plane API                        │
                                                                 │
Неделя 3: Multi-Port + Cross-Core Routing                       │
├── Dynamic frontend generation для любых портов             │
├── SNI-based routing на все 3 ядра (Xray/Sing/Mihomo)       │
├── Path-based routing для HTTP транспортов                    │
└── Backend health checks                                     │
                                                                 │
Неделя 4: Core Integration + Docker                           │
├── Xray: PROXY v2 (TCP/WS/XHTTP only, НЕ gRPC)              │
├── Sing-box: X-Forwarded-For (НЕТ PROXY v2)                 │
├── Mihomo: X-Forwarded-For (НЕТ PROXY v2)                     │
├── Docker Compose: network_mode: host                        │
└── Integration tests                                         │
                                                                 │
Неделя 5: Smart Warning UI Validation                         │
├── PortValidator с severity levels (info/warning/error)      │
├── API endpoint: POST /api/inbounds/check-port            │
├── Frontend PortValidationField компонент                    │
├── Real-time validation с debounce (500ms)                 │
└── Modal confirmation для warning state                      │
                                                                 │
Неделя 6: Testing & Documentation                             │
├── Integration tests: SNI routing, multi-port scenarios      │
├── Cross-core routing tests                                  │
├── UI validation tests (e2e)                                │
├── Prometheus metrics + Grafana dashboard                    │
└── Documentation updates                                    │
                                                                 │
Неделя 7-8: HWID Tracking                                     │
├── Database schema (user_devices)                           │
├── Tracker service (3-level fingerprinting)                 │
├── Middleware для subscription endpoints                       │
└── API + UI для управления устройствами                      │
                                                                 │
Неделя 9-12: Advanced Routing UI                              │
├── Rule builder (form-based)                                │
├── Drag-drop priority list                                  │
├── Monaco editor (JSON/YAML)                                 │
├── Rule tester (live preview)                                │
└── Sankey diagram MVP                                       │
```

### 7.2 Ключевые зависимости

| Фича | Зависит от | Статус |
|------|-----------|--------|
| **HAProxy Мульти-порт + Smart Warning** | CoreAdapter ✅ | Готово к старту |
| **HWID Tracking** | Subscriptions API | Нужен endpoint |
| **Advanced Routing UI** | Traffic metrics | Нужен collector |
| **Sankey Diagram** | Traffic data | Нужен aggregation |

### 7.3 Критический путь

**Путь к Production-ready Multi-Port SNI Routing**:
1. ✅ CoreAdapter (завершенно в Фазе 2)
2. 🔄 HAProxy интеграция с мульти-порт (6 недель)
   - Week 1-2: Core infrastructure
   - Week 3: Multi-port + cross-core routing
   - Week 4: Core integration + Docker
   - Week 5: Smart Warning UI
   - Week 6: Testing + documentation
3. ⏸️ Domain management UI (параллельно)
4. ⏸️ Let's Encrypt integration (Certbot или Caddy sidecar)
5. ⏸️ CDN integration (Cloudflare IP lists)

**Минимальный viable продукт** (MVP HAProxy v2.0):
- ✅ Мульти-порт: любые порты (443, 8443, 8080)
- ✅ Кросс-ядерная маршрутизация (Xray/Sing/Mihomo на одном порту)
- ✅ Smart Warning UI валидация
- ✅ PROXY protocol v2 для Xray (TCP/WS)
- ✅ X-Forwarded-For для Sing-box/Mihomo
- ✅ Docker Compose с host network

---

## 8. Ссылки и источники

### HWID Tracking
- [3X-UI HWID PR #3635](https://github.com/MHSanaei/3x-ui/pull/3635)
- [Marzban Device Limiter](https://github.com/kets-kets/marzban-device-limiter)
- [Fingerprint.com](https://fingerprint.com/banking/)
- [JA4+ Fingerprinting](https://github.com/FoxIO-LLC/ja4)
- [Netflix Sharing Detection](https://www.educative.io/newsletter/system-design/how-netflix-built-system-level-enforcement-for-password-sharing)

### SNI Routing
- [HAProxy SNI Docs](https://www.haproxy.com/blog/enhanced-ssl-load-balancing-with-server-name-indication-sni-tls-extension)
- [Hiddify HAProxy Config](https://github.com/hiddify/Hiddify-Manager/tree/main/haproxy)
- [NGINX Stream SNI](https://nginx.org/docs/stream/ngx_stream_ssl_preread_module.html)
- [Caddy Layer4 Plugin](https://github.com/mholt/caddy-l4)

### Routing UI
- [Clash Verge Rev Rules](https://clashvergerev.com/en/guide.html)
- [DDS Xray Routing Editor](https://github.com/azavaxhuman/DDS-Xray-Routing-Editor)
- [V2RayA RoutingA](https://v2raya.org/en/docs/manual/routinga/)

---

**Document Version**: 2.0  
**Last Updated**: 2026-04-19  
**Status**: Ready for implementation review (Consolidated with HAPROXY_IMPLEMENTATION_PLAN.md v3.0)
