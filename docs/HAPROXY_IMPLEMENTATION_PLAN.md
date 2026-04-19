# HAPROXY МУЛЬТИ-ПОРТ + КРОСС-ЯДЕРНАЯ РЕАЛИЗАЦИЯ

> **Дата**: 2026-04-19  
> **Версия**: 3.0 (Consolidated - Dynamic Port Allocation)  
> **Статус**: Готов к реализации  
> **Приоритет**: P1 (High)

---

## 📋 СОДЕРЖАНИЕ

1. [Резюме: Возможно ли всё?](#1-резюме-возможно-ли-всё)
2. [Полная матрица совместимости протоколов (25+)](#2-полная-матрица-совместимости-протоколов-25)
3. [Архитектура динамического распределения портов](#3-архитектура-динамического-распределения-портов)
4. [Кросс-ядерная маршрутизация](#4-кросс-ядерная-маршрутизация)
5. [UI логика валидации (Smart Warning)](#5-ui-логика-валидации-smart-warning)
6. [Детальная реализация](#6-детальная-реализация)
7. [План реализации](#7-план-реализации)

---

## 1. РЕЗЮМЕ: Возможно ли всё?

### ✅ **ДА, все задумки реализуемы**

| Требование | Статус | Реализация |
|------------|--------|------------|
| **Мульти-инбаунд на один порт** | ✅ Реализуемо | HAProxy SNI/Path-based routing |
| **Любые порты (не только 443)** | ✅ Реализуемо | Динамическое распределение |
| **Любое ядро на любом порту** | ✅ Реализуемо | Backend binding к любому порту |
| **Кросс-ядерная маршрутизация** | ✅ Реализуемо | Backend routing на все 3 ядра |
| **UI валидация Smart Warning** | ✅ Реализуемо | Severity-based validation |
| **Поддержка ВСЕХ протоколов (25+)** | ⚠️ Частично | UDP/QUIC — direct ports |

### 🔴 КРИТИЧЕСКИЕ ОГРАНИЧЕНИЯ

| Ядро | Ограничение | Влияние | Решение |
|------|-------------|---------|---------|
| **Xray** | gRPC + PROXY protocol v2 ([Issue #2204](https://github.com/XTLS/Xray-core/issues/2204)) | 🟡 Среднее | Использовать XHTTP/WS; gRPC без PROXY v2 |
| **Sing-box** | **PROXY protocol REMOVED в 1.6.0+** ([deprecated](https://sing-box.sagernet.org/deprecated/)) | 🟡 Среднее | Только X-Forwarded-For headers |
| **HAProxy** | Нет UDP/QUIC forwarding ([Issue #2748](https://github.com/haproxy/haproxy/issues/2748)) | 🟡 Среднее | Direct ports для Hysteria2/TUIC/KCP/QUIC |
| **Mihomo** | Нет PROXY protocol v2 | 🟢 Низкое | Только X-Forwarded-For headers |

**Вывод**: Real client IP передаётся только для Xray TCP-based через PROXY v2. Sing-box и Mihomo — только X-Forwarded-For. UDP/QUIC протоколы работают на direct портах, минуя HAProxy.

---

## 2. ПОЛНАЯ МАТРИЦА СОВМЕСТИМОСТИ ПРОТОКОЛОВ (25+)

### 2.1 Xray Core (v26.3.27) — 12 протоколов

**Входящие (Inbounds):**

| Protocol | Transport | Через HAProxy | PROXY v2 | SNI | Path | Reality | Примечания |
|----------|-----------|---------------|----------|-----|------|---------|------------|
| **http** | TCP | ✅ | ✅ | ❌ | ❌ | ❌ | HTTP proxy |
| **socks5** (socks) | TCP | ✅ | ✅ | ❌ | ❌ | ❌ | SOCKS5 proxy |
| **shadowsocks** | TCP | ✅ | ✅ | ❌ | ❌ | ❌ | AEAD шифрование |
| **shadowsocks** | WebSocket | ✅ | ✅ | ❌ | ✅ | ❌ | WS transport |
| **shadowsocks** | gRPC | ✅ | ⚠️ | ❌ | ✅ | ❌ | Без PROXY v2 |
| **vmess** | TCP+TLS | ✅ | ✅ | ✅ | ❌ | ❌ | V2Ray protocol |
| **vmess** | WebSocket+TLS | ✅ | ✅ | ✅ | ✅ | ❌ | Path-based |
| **vmess** | HTTPUpgrade+TLS | ✅ | ✅ | ✅ | ✅ | ❌ | Path-based |
| **vmess** | XHTTP (H1/H2) | ✅ | ❌ | ✅ | ✅ | ❌ | X-Forwarded-Only |
| **vmess** | gRPC | ✅ | ❌ | ✅ | ✅ | ❌ | [Issue #2204](https://github.com/XTLS/Xray-core/issues/2204) |
| **vmess** | QUIC | ❌ | ❌ | ❌ | ❌ | ❌ | UDP-based, direct port |
| **vless** | TCP+TLS | ✅ | ✅ | ✅ | ❌ | ❌ | Flow: none/xtls |
| **vless** | TCP+Reality | ✅ | ✅ | ⚠️ | ❌ | ✅ | Special SNI flow |
| **vless** | WebSocket+TLS | ✅ | ✅ | ✅ | ✅ | ❌ | Полная поддержка |
| **vless** | WebSocket+Reality | ✅ | ✅ | ⚠️ | ✅ | ✅ | Reality over WS |
| **vless** | HTTPUpgrade+TLS | ✅ | ✅ | ✅ | ✅ | ❌ | Path: `/upgrade` |
| **vless** | HTTPUpgrade+Reality | ✅ | ✅ | ⚠️ | ✅ | ✅ | Reality over HUP |
| **vless** | XHTTP | ✅ | ❌ | ✅ | ✅ | ❌ | X-Forwarded-Only |
| **vless** | XHTTP+Reality | ✅ | ❌ | ⚠️ | ✅ | ✅ | Reality over XHTTP |
| **vless** | gRPC | ✅ | ❌ | ✅ | ✅ | ❌ | Без PROXY v2 |
| **vless** | gRPC+Reality | ✅ | ❌ | ⚠️ | ✅ | ✅ | Reality over gRPC |
| **vless** | QUIC | ❌ | ❌ | ❌ | ❌ | ❌ | UDP-based, direct port |
| **trojan** | TCP+TLS | ✅ | ✅ | ✅ | ❌ | ❌ | Standard |
| **trojan** | WebSocket+TLS | ✅ | ✅ | ✅ | ✅ | ❌ | Path-based |
| **trojan** | HTTPUpgrade+TLS | ✅ | ✅ | ✅ | ✅ | ❌ | Path-based |
| **trojan** | XHTTP | ✅ | ❌ | ✅ | ✅ | ❌ | X-Forwarded-Only |
| **trojan** | gRPC | ✅ | ❌ | ✅ | ✅ | ❌ | Без PROXY v2 |
| **hysteria2** | UDP/QUIC | ❌ | ❌ | ❌ | ❌ | ❌ | **Direct port** |
| **tun** | — | ❌ | ❌ | ❌ | ❌ | ❌ | TUN device |

**Исходящие (Outbounds):** direct, block, dns, hysteria (outbound only)

### 2.2 Sing-box Core (v1.13.8) — 15 протоколов

> ⚠️ **ВАЖНО**: PROXY protocol **DEPRECATED и REMOVED** в Sing-box 1.6.0+
> Real client IP только через `X-Forwarded-For` заголовки для HTTP-based транспортов.

| Protocol | Transport | Через HAProxy | SNI | Path | Примечания |
|----------|-----------|---------------|-----|------|------------|
| **http** | TCP | ✅ | ❌ | ❌ | HTTP proxy |
| **socks5** (socks) | TCP | ✅ | ❌ | ❌ | SOCKS5 proxy |
| **mixed** | TCP | ✅ | ❌ | ❌ | HTTP+SOCKS5 combo |
| **shadowsocks** | TCP | ✅ | ❌ | ❌ | 2022 ciphers supported |
| **vmess** | tcp+tls | ✅ | ✅ | ❌ | V2Ray protocol |
| **vmess** | websocket | ✅ | ✅ | ✅ | Path: `ws-path` |
| **vmess** | httpupgrade | ✅ | ✅ | ✅ | Path: `path` |
| **vmess** | grpc | ✅ | ✅ | ✅ | Service name routing |
| **vless** | tcp+tls | ✅ | ✅ | ❌ | Vision flow |
| **vless** | websocket | ✅ | ✅ | ✅ | Path-based |
| **vless** | httpupgrade | ✅ | ✅ | ✅ | Path-based |
| **vless** | grpc | ✅ | ✅ | ✅ | gRPC routing |
| **trojan** | tcp+tls | ✅ | ✅ | ❌ | Fallback support |
| **trojan** | websocket | ✅ | ✅ | ✅ | Path-based |
| **trojan** | httpupgrade | ✅ | ✅ | ✅ | Path-based |
| **trojan** | grpc | ✅ | ✅ | ✅ | gRPC routing |
| **shadowtls** | tcp+tls | ✅ | ✅ | ❌ | TLS v1/v2/v3 |
| **naive** | TCP | ✅ | ✅ | ❌ | **Sing-box exclusive** |
| **anytls** | tcp+tls | ✅ | ✅ | ❌ | sing-box 1.12+ |
| **redirect** | TCP | ✅ | ❌ | ❌ | Transparent redirect |
| **hysteria2** | UDP/QUIC | ❌ | ❌ | ❌ | **Direct port** |
| **tuic** (v4/v5) | UDP/QUIC | ❌ | ❌ | ❌ | **Direct port** |
| **tun** | — | ❌ | ❌ | ❌ | TUN device |
| **tproxy** | TCP/UDP | ⚠️ | ❌ | ❌ | Linux TProxy |

**Исходящие (Outbounds):** direct, block, dns, tor, naive, anytls

### 2.3 Mihomo (Clash Meta) (v1.19.23) — 19 протоколов

> ⚠️ **ВАЖНО**: Mihomo **не поддерживает PROXY protocol v2** для inbounds.
> Real client IP только через `X-Forwarded-For`.

| Protocol | Network | Через HAProxy | SNI | Path | Примечания |
|----------|---------|---------------|-----|------|------------|
| **http** | TCP | ✅ | ❌ | ❌ | Basic auth |
| **socks5** | TCP | ✅ | ❌ | ❌ | UDP associate отдельно |
| **mixed** | TCP | ✅ | ❌ | ❌ | HTTP+SOCKS5 |
| **shadowsocks** (ss) | TCP | ✅ | ❌ | ❌ | AEAD + 2022 ciphers |
| **shadowsocksr** (ssr) | TCP | ✅ | ❌ | ❌ | **Mihomo exclusive** Legacy |
| **vmess** | tcp | ✅ | ✅ | ❌ | Optional TLS |
| **vmess** | ws/wss | ✅ | ✅ | ✅ | `ws-path` config |
| **vmess** | grpc | ✅ | ✅ | ✅ | `grpc-service-name` |
| **vmess** | h2 | ✅ | ✅ | ❌ | TLS required |
| **vless** | tcp+tls | ✅ | ✅ | ❌ | Reality поддержка |
| **vless** | ws/wss | ✅ | ✅ | ✅ | `ws-path` config |
| **vless** | grpc | ✅ | ✅ | ✅ | `grpc-service-name` |
| **trojan** | tcp+tls | ✅ | ✅ | ❌ | **Обязательно TLS** |
| **trojan** | ws/wss | ✅ | ✅ | ✅ | `ws-path` config |
| **trojan** | grpc | ✅ | ✅ | ✅ | `grpc-service-name` |
| **anytls** | tcp+tls | ✅ | ✅ | ❌ | Mihomo v1.19.3+ |
| **mieru** | TCP | ✅ | ❌ | ❌ | **Mihomo exclusive** |
| **sudoku** | TCP | ✅ | ❌ | ❌ | **Mihomo exclusive** |
| **snell** | TCP | ✅ | ❌ | ❌ | **Mihomo exclusive** |
| **trusttunnel** | TCP | ✅ | ❌ | ❌ | **Mihomo exclusive** |
| **redirect** | TCP | ✅ | ❌ | ❌ | Transparent |
| **tunnel** | TCP | ✅ | ❌ | ❌ | Port forwarding |
| **tuic** | UDP/QUIC | ❌ | ❌ | ❌ | **Direct port** |
| **hysteria** | UDP/QUIC | ❌ | ❌ | ❌ | **Direct port** |
| **tproxy** | Linux | ❌ | ❌ | ❌ | Kernel-level |
| **tun** | Kernel | ❌ | ❌ | ❌ | TUN device |
| **masque** | HTTP/3 | ❌ | ❌ | ❌ | **Outbound only** |

**Исходящие (Outbounds):** DIRECT, REJECT, DNS, masque

### 2.4 Сводная таблица всех 25+ протоколов

| Protocol | Xray | Sing-box | Mihomo | Категория | Требует TLS |
|----------|------|----------|--------|-----------|-------------|
| http | ✅ | ✅ | ✅ | proxy | ❌ |
| socks5 | ✅ | ✅ | ✅ | proxy | ❌ |
| mixed | — | ✅ | ✅ | proxy | ❌ |
| shadowsocks | ✅ | ✅ | ✅ | proxy | ❌ |
| vmess | ✅ | ✅ | ✅ | proxy | ⚠️ |
| vless | ✅ | ✅ | ✅ | proxy | ⚠️ |
| trojan | ✅ | ✅ | ✅ | proxy | ✅ |
| hysteria2 | ✅ | ✅ | ✅ | tunnel | ✅ |
| hysteria | ✅ (out) | — | ✅ (out) | tunnel | ✅ |
| tuic | — | ✅ | ✅ | tunnel | ✅ |
| anytls | — | ✅ | ✅ | proxy | ✅ |
| naive | — | ✅ (excl) | — | proxy | ✅ |
| xhttp | ✅ (excl) | — | — | proxy | ❌ |
| tun | ✅ (excl) | ✅ | ✅ | tunnel | ❌ |
| mieru | — | — | ✅ (excl) | tunnel | ❌ |
| sudoku | — | — | ✅ (excl) | tunnel | ❌ |
| snell | — | — | ✅ (excl) | tunnel | ❌ |
| trusttunnel | — | — | ✅ (excl) | tunnel | ❌ |
| shadowsocksr | — | — | ✅ (excl) | proxy | ❌ |
| shadowtls | — | ✅ | — | proxy | ✅ |
| redirect | — | ✅ | ✅ | utility | ❌ |
| direct | ✅ (out) | ✅ (out) | ✅ (out) | utility | ❌ |
| block | ✅ (out) | ✅ (out) | ✅ (out) | utility | ❌ |
| dns | ✅ (out) | ✅ (out) | ✅ (out) | utility | ❌ |
| tor | — | ✅ (out) | — | tunnel | ❌ |
| masque | — | — | ✅ (out) | proxy | ❌ |

**Эксклюзивные протоколы:**
- **Xray**: xhttp (SplitHTTP), tun (inbound)
- **Sing-box**: naive, shadowtls
- **Mihomo**: shadowsocksr, mieru, sudoku, snell, trusttunnel, masque (outbound)

---

## 3. АРХИТЕКТУРА ДИНАМИЧЕСКОГО РАСПРЕДЕЛЕНИЯ ПОРТОВ

### 3.1 Концепция: Любой порт для любого ядра

**Устаревший подход (фиксированные диапазоны):**
```
❌ Xray: 10001-10010
❌ Sing-box: 9090-9099
❌ Mihomo: 9091-9099
```

**Новый подход (динамическое распределение):**
```
✅ Пользователь выбирает любой порт (1-65535)
✅ Система динамически назначает backend port из пула
✅ Любое ядро на любом порту
✅ Несколько ядер на одном порту через HAProxy
✅ Порт не привязан к ядру — ядро привязано к порту
```

### 3.2 Архитектура потока трафика

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            INTERNET                                      │
└───────────────────────────────┬───────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    HAProxy (Host Network, Multi-Port)                   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Frontend :443 (Пользовательский выбор)                         │   │
│  │  ├── SNI: vless.example.com ──→ Backend: Xray (:DP1)          │   │
│  │  ├── SNI: vmess.example.com ──→ Backend: Sing (:DP2)           │   │
│  │  └── SNI: trojan.example.com ──→ Backend: Mihomo (:DP3)       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Frontend :8443 (Пользовательский выбор)                      │   │
│  │  ├── Path: /vless-ws ──→ Backend: Xray (:DP4)                 │   │
│  │  └── Path: /trojan-grpc ──→ Backend: Mihomo (:DP5)            │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Frontend :8080 (Пользовательский выбор)                       │   │
│  │  └── Direct HTTP proxy inbounds                               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Direct Ports (UDP/QUIC - не через HAProxy)                     │   │
│  │  :8444 Hysteria2 ──→ Sing-box (:DP6)                          │   │
│  │  :8445 TUIC ──→ Sing-box (:DP7)                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   Xray       │      │  Sing-box    │      │   Mihomo     │
│  (Любые      │      │  (Любые      │      │  (Любые      │
│   протоколы  │      │   протоколы) │      │   протоколы) │
│   на портах  │      │   на портах  │      │   на портах  │
│   DP1, DP4)  │      │   DP2, DP6-7)│      │   DP3, DP5)  │
└──────────────┘      └──────────────┘      └──────────────┘

DP = Dynamic Port (40000-50000 пул)
```

### 3.3 Dynamic Port Allocation Model

**PortAssignment Struct:**
```go
type PortAssignment struct {
    ID              uint      `gorm:"primaryKey" json:"id"`
    InboundID       uint      `json:"inbound_id" gorm:"uniqueIndex;not null"`
    
    // Пользовательский выбор
    UserListenPort  int       `json:"user_listen_port" gorm:"index;not null"`      // Например: 443, 8443
    UserListenAddr  string    `json:"user_listen_addr" gorm:"default:'0.0.0.0'"`   // 0.0.0.0, 127.0.0.1
    
    // Системное назначение (динамическое)
    BackendPort     int       `json:"backend_port" gorm:"not null"`                 // 40001, 40002...
    CoreType        string    `json:"core_type" gorm:"index;not null"`            // "xray", "singbox", "mihomo"
    
    // HAProxy routing
    UseHAProxy      bool      `json:"use_haproxy" gorm:"default:true"`             // true = через HAProxy
    SNIMatch        string    `json:"sni_match,omitempty"`                           // vless.example.com
    PathMatch       string    `json:"path_match,omitempty"`                          // /vless-ws
    
    // PROXY protocol (только Xray TCP-based)
    SendProxyProtocol bool    `json:"send_proxy_protocol" gorm:"default:false"`
    ProxyProtocolVersion int   `json:"proxy_protocol_version,omitempty"`            // 2
    
    IsActive        bool      `json:"is_active" gorm:"default:true"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

**Backend Port Pool:**
```yaml
# config.yaml
backend_port_pool:
  start: 40000        # Начало динамического пула
  end: 50000          # Конец пула
  assignment: auto    # Автоназначение

# Примеры назначений:
# Inbound "VLESS-Reality-1": User port 443 → Backend 40001 (Xray)
# Inbound "VMess-WS-1": User port 443 → Backend 40002 (Sing-box)
# Inbound "Trojan-gRPC-1": User port 8443 → Backend 40003 (Mihomo)
# Inbound "Hysteria2-1": User port 8444 → Direct 40004 (Sing-box, не через HAProxy)
```

### 3.4 Пример: Пользователь создаёт 4 инбаунда на порту 443

**Инбаунд 1**: VLESS + Reality + Xray
- Пользователь выбирает: Port = 443, Protocol = VLESS, Transport = TCP, Core = Xray
- Система назначает: BackendPort = 40001
- HAProxy: SNI = vless.example.com → Backend 127.0.0.1:40001 (send-proxy-v2)

**Инбаунд 2**: VMess + WebSocket + Sing-box
- Пользователь выбирает: Port = 443, Protocol = VMess, Transport = WS, Core = Sing-box
- Система назначает: BackendPort = 40002
- HAProxy: Path = /vmess-ws → Backend 127.0.0.1:40002 (X-Forwarded-For)

**Инбаунд 3**: Trojan + gRPC + Mihomo
- Пользователь выбирает: Port = 443, Protocol = Trojan, Transport = gRPC, Core = Mihomo
- Система назначает: BackendPort = 40003
- HAProxy: Path = /trojan-grpc → Backend 127.0.0.1:40003 (X-Forwarded-For)

**Инбаунд 4**: Shadowsocks + TCP + Xray
- Пользователь выбирает: Port = 443, Protocol = Shadowsocks, Transport = TCP, Core = Xray
- Система назначает: BackendPort = 40004
- HAProxy: SNI = ss.example.com → Backend 127.0.0.1:40004 (send-proxy-v2)

**Результат HAProxy конфигурации:**
```haproxy
frontend ft_user_443
    bind :443 v4v6 tfo ssl crt /etc/certs/fullchain.pem
    mode tcp
    
    tcp-request inspect-delay 5s
    tcp-request content accept if { req_ssl_hello_type 1 }
    
    # SNI-based routing
    acl is_vless_reality req.ssl_sni -i vless.example.com
    use_backend bk_xray_40001 if is_vless_reality
    
    acl is_shadowsocks req.ssl_sni -i ss.example.com
    use_backend bk_xray_40004 if is_shadowsocks
    
    # Path-based routing (switch to http mode)
    use_backend bk_singbox_40002 if { path_beg /vmess-ws }
    use_backend bk_mihomo_40003 if { path_beg /trojan-grpc }
    
    default_backend bk_panel_ui

backend bk_xray_40001
    mode tcp
    server xray_40001 127.0.0.1:40001 send-proxy-v2 check inter 5s

backend bk_singbox_40002
    mode http
    option http-server-close
    timeout tunnel 1h
    http-request set-header X-Forwarded-For %[src]
    server sing_40002 127.0.0.1:40002 check inter 5s

backend bk_mihomo_40003
    mode http
    option http-server-close
    timeout tunnel 1h
    http-request set-header X-Forwarded-For %[src]
    server mihomo_40003 127.0.0.1:40003 check inter 5s

backend bk_xray_40004
    mode tcp
    server xray_40004 127.0.0.1:40004 send-proxy-v2 check inter 5s
```

---

## 4. КРОСС-ЯДЕРНАЯ МАРШРУТИЗАЦИЯ

### 4.1 Концепция: Один порт, разные ядра

**Пример: Порт 443 с 3 ядрами одновременно**
```
Internet → HAProxy :443
              │
              ├── SNI: vless.example.com ──→ Backend: xray (:40001)
              ├── SNI: vmess.example.com ──→ Backend: singbox (:40002)  
              ├── SNI: trojan.example.com ──→ Backend: mihomo (:40003)
              └── default ──→ Backend: panel_web (:8080)
```

**Реализация в HAProxy:**
```haproxy
frontend ft_main_https
    bind :443 v4v6 tfo ssl crt /etc/certs/fullchain.pem
    mode tcp
    
    tcp-request inspect-delay 5s
    tcp-request content accept if { req_ssl_hello_type 1 }
    
    # Cross-core SNI routing
    acl is_xray_vless req.ssl_sni -i vless.example.com
    use_backend bk_xray_40001 if is_xray_vless
    
    acl is_sing_vmess req.ssl_sni -i vmess.example.com
    use_backend bk_singbox_40002 if is_sing_vmess
    
    acl is_mihomo_trojan req.ssl_sni -i trojan.example.com
    use_backend bk_mihomo_40003 if is_mihomo_trojan
    
    default_backend bk_panel_ui

# Xray backend (TCP mode, PROXY v2)
backend bk_xray_40001
    mode tcp
    server xray1 127.0.0.1:40001 send-proxy-v2 check inter 5s

# Sing-box backend (TCP mode, NO PROXY v2 - removed in 1.6.0)
backend bk_singbox_40002
    mode tcp
    server sing1 127.0.0.1:40002 check inter 5s

# Mihomo backend (TCP mode, NO PROXY v2)
backend bk_mihomo_40003
    mode tcp
    server mihomo1 127.0.0.1:40003 check inter 5s

backend bk_panel_ui
    mode http
    server panel 127.0.0.1:8080 check inter 5s
```

### 4.2 Database Schema для кросс-ядерной маршрутизации

```sql
-- Расширение таблицы inbounds
ALTER TABLE inbounds ADD COLUMN core_type VARCHAR(20) NOT NULL DEFAULT 'xray';
ALTER TABLE inbounds ADD COLUMN protocol VARCHAR(50) NOT NULL;
ALTER TABLE inbounds ADD COLUMN transport VARCHAR(50) DEFAULT 'tcp';
ALTER TABLE inbounds ADD COLUMN tls_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE inbounds ADD COLUMN reality_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE inbounds ADD COLUMN sni_match VARCHAR(255);
ALTER TABLE inbounds ADD COLUMN path_match VARCHAR(255);
ALTER TABLE inbounds ADD COLUMN priority INTEGER DEFAULT 100;

-- Таблица динамических назначений портов
CREATE TABLE port_assignments (
    id BIGSERIAL PRIMARY KEY,
    inbound_id INTEGER REFERENCES inbounds(id) ON DELETE CASCADE,
    
    -- Пользовательский выбор
    user_listen_port INTEGER NOT NULL,
    user_listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    
    -- Динамическое назначение системой
    backend_port INTEGER NOT NULL,
    core_type VARCHAR(20) NOT NULL,
    
    -- HAProxy routing
    use_haproxy BOOLEAN DEFAULT TRUE,
    sni_match VARCHAR(255),
    path_match VARCHAR(255),
    send_proxy_protocol BOOLEAN DEFAULT FALSE,
    proxy_protocol_version INTEGER,
    
    -- Статус
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(inbound_id)
);

-- Таблица direct портов (UDP/QUIC - не через HAProxy)
CREATE TABLE direct_ports (
    id BIGSERIAL PRIMARY KEY,
    inbound_id INTEGER REFERENCES inbounds(id) ON DELETE CASCADE,
    listen_port INTEGER NOT NULL,
    listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    core_type VARCHAR(20) NOT NULL,
    backend_port INTEGER NOT NULL,  -- Dynamic port for core
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(inbound_id)
);

-- Indexes
CREATE INDEX idx_port_assignments_user_port ON port_assignments(user_listen_port);
CREATE INDEX idx_port_assignments_backend ON port_assignments(backend_port);
CREATE INDEX idx_port_assignments_core ON port_assignments(core_type);
CREATE INDEX idx_inbounds_sni ON inbounds(sni_match) WHERE sni_match IS NOT NULL;
CREATE INDEX idx_inbounds_path ON inbounds(path_match) WHERE path_match IS NOT NULL;
CREATE INDEX idx_inbounds_core ON inbounds(core_type);
```

---

## 5. UI ЛОГИКА ВАЛИДАЦИИ (SMART WARNING)

### 5.1 User Requirements

> "В UI у пользователя при выборе порта у инбаунда должно появляться уведомление:
> - Если HAProxy выбранный протокол поддерживает → 'всё хорошо, мульти-инбаунд OK'
> - Если НЕ поддерживает → запрет создания на занятом порту"

### 5.2 Алгоритм определения Severity

```go
// PortConflictCheck содержит результат валидации
type PortConflictCheck struct {
    Port                int                `json:"port"`
    ListenAddress       string             `json:"listen_address"`
    Protocol            string             `json:"protocol"`
    Transport           string             `json:"transport,omitempty"`
    CoreType            string             `json:"core_type"`
    
    IsAvailable         bool               `json:"is_available"`
    HaproxyCompatible   bool               `json:"haproxy_compatible"`
    CanSharePort        bool               `json:"can_share_port"`
    SharingMechanism    string             `json:"sharing_mechanism,omitempty"` // "sni", "path", "none"
    
    Severity            ValidationSeverity `json:"severity"`   // "info", "warning", "error"
    Action              ValidationAction   `json:"action"`     // "allow", "confirm", "block"
    Message             string             `json:"message"`    // Russian message
    Conflicts           []PortConflict     `json:"conflicts,omitempty"`
}

type PortConflict struct {
    InboundID           uint   `json:"inbound_id"`
    InboundName         string `json:"inbound_name"`
    Protocol            string `json:"protocol"`
    Transport           string `json:"transport,omitempty"`
    CoreType            string `json:"core_type"`
    Port                int    `json:"port"`
    
    HaproxyCompatible   bool   `json:"haproxy_compatible"`
    CanShare           bool   `json:"can_share"`
    SharingMechanism   string `json:"sharing_mechanism,omitempty"`
    RequiresConfirm    bool   `json:"requires_confirm"`
}

// ValidatePortConflict проверяет возможность использования порта
func (v *PortValidator) ValidatePortConflict(
    port int,
    listenAddr string,
    protocol string,
    transport string,
    coreType string,
    existingInbounds []Inbound,
) *PortConflictCheck {
    result := &PortConflictCheck{
        Port:          port,
        ListenAddress: listenAddr,
        Protocol:      protocol,
        Transport:     transport,
        CoreType:      coreType,
    }
    
    // Проверяем поддержку HAProxy для нового инбаунда
    result.HaproxyCompatible = v.isHaproxyCompatible(protocol, transport, coreType)
    
    // Ищем конфликты
    for _, existing := range existingInbounds {
        if !v.isPortOverlap(port, listenAddr, existing.Port, existing.Listen) {
            continue
        }
        
        conflict := PortConflict{
            InboundID:   existing.ID,
            InboundName: existing.Remark,
            Protocol:    existing.Protocol,
            Transport:   existing.Transport,
            CoreType:    existing.CoreType,
            Port:        existing.Port,
        }
        
        // Проверяем HAProxy совместимость существующего инбаунда
        existingCompatible := v.isHaproxyCompatible(
            existing.Protocol, 
            existing.Transport, 
            existing.CoreType,
        )
        
        // Могут ли они работать совместно через HAProxy?
        if result.HaproxyCompatible && existingCompatible {
            conflict.HaproxyCompatible = true
            conflict.CanShare = true
            result.CanSharePort = true
            
            // Определяем механизм разделения
            if v.supportsSNI(protocol) && v.supportsSNI(existing.Protocol) {
                conflict.SharingMechanism = "sni"
                result.SharingMechanism = "sni"
            } else if v.supportsPath(transport) && v.supportsPath(existing.Transport) {
                conflict.SharingMechanism = "path"
                result.SharingMechanism = "path"
            }
        } else {
            conflict.HaproxyCompatible = existingCompatible
            conflict.CanShare = false
        }
        
        // Нужно ли подтверждение?
        if conflict.CanShare && protocol != existing.Protocol {
            conflict.RequiresConfirm = true
        }
        
        result.Conflicts = append(result.Conflicts, conflict)
    }
    
    // Определяем severity и сообщение
    result.IsAvailable = len(result.Conflicts) == 0 || result.CanSharePort
    
    if len(result.Conflicts) == 0 {
        // Порт свободен
        result.Severity = SeverityInfo
        result.Message = "✓ Порт свободен. Можно создавать инбаунд."
        result.Action = ActionAllow
        
    } else if result.CanSharePort {
        // Порт занят, но HAProxy может маршрутизировать
        if result.SharingMechanism == "sni" {
            result.Severity = SeverityInfo
            result.Message = fmt.Sprintf(
                "ℹ Порт %d используется %d инбаундом(ами). " +
                "HAProxy обеспечит корректную маршрутизацию через SNI.",
                port, len(result.Conflicts),
            )
            result.Action = ActionAllow
        } else if result.SharingMechanism == "path" {
            result.Severity = SeverityInfo
            result.Message = fmt.Sprintf(
                "ℹ Порт %d используется %d инбаундом(ами). " +
                "HAProxy обеспечит корректную маршрутизацию через Path.",
                port, len(result.Conflicts),
            )
            result.Action = ActionAllow
        } else {
            // Нет чёткого механизма разделения
            result.Severity = SeverityWarning
            result.Message = fmt.Sprintf(
                "⚠ Порт %d уже используется инбаундом '%s' (%s/%s). " +
                "HAProxy может обеспечить совместную работу, но убедитесь, " +
                "что SNI/Path отличаются от существующих.",
                port,
                result.Conflicts[0].InboundName,
                result.Conflicts[0].Protocol,
                result.Conflicts[0].Transport,
            )
            result.Action = ActionConfirm
        }
        
    } else {
        // Порт занят и несовместим с HAProxy
        result.Severity = SeverityError
        
        // Определяем причину
        var reasons []string
        for _, c := range result.Conflicts {
            if !c.HaproxyCompatible {
                reasons = append(reasons, fmt.Sprintf("%s не поддерживает HAProxy", c.InboundName))
            }
        }
        
        if len(reasons) > 0 {
            result.Message = fmt.Sprintf(
                "✗ Порт %d уже используется и НЕ может быть совместно использован: %s. " +
                "Выберите другой порт или удалите конфликтующие инбаунды.",
                port,
                strings.Join(reasons, ", "),
            )
        } else {
            result.Message = fmt.Sprintf(
                "✗ Порт %d уже используется инбаундом '%s'. " +
                "Протоколы несовместимы для совместной работы через HAProxy.",
                port,
                result.Conflicts[0].InboundName,
            )
        }
        result.Action = ActionBlock
    }
    
    return result
}

// isHaproxyCompatible проверяет поддержку HAProxy
func (v *PortValidator) isHaproxyCompatible(protocol, transport, coreType string) bool {
    // UDP-based транспорты никогда не совместимы
    if isUDPTransport(transport) {
        return false
    }
    
    // Все TCP-based протоколы могут работать через HAProxy
    return true
}

// supportsSNI проверяет поддержку SNI
func (v *PortValidator) supportsSNI(protocol string) bool {
    // SNI работает только с TLS-based протоколами
    tlsProtocols := []string{"vless", "vmess", "trojan", "shadowtls", "anytls", "naive"}
    return contains(tlsProtocols, protocol)
}

// supportsPath проверяет поддержку path-based routing
func (v *PortValidator) supportsPath(transport string) bool {
    pathTransports := []string{"ws", "websocket", "httpupgrade", "xhttp", "grpc"}
    return contains(pathTransports, transport)
}

// isUDPTransport проверяет UDP-based транспорт
func isUDPTransport(transport string) bool {
    udpTransports := []string{"quic", "kcp", "hysteria", "hysteria2", "tuic"}
    return contains(udpTransports, transport)
}
```

### 5.3 Severity Levels Summary

| Severity | Icon | Color | Условие | Сообщение (RU) | Действие |
|----------|------|-------|---------|----------------|----------|
| **Info** | ✓ | 🟢 Зелёный | Порт свободен | "Порт свободен. Можно создавать инбаунд." | ✅ Allow |
| **Info** | ℹ | 🟢 Зелёный | Порт занят, SNI routing | "Порт используется N инбаундами. HAProxy обеспечит маршрутизацию через SNI." | ✅ Allow |
| **Info** | ℹ | 🟢 Зелёный | Порт занят, Path routing | "Порт используется N инбаундами. HAProxy обеспечит маршрутизацию через Path." | ✅ Allow |
| **Warning** | ⚠️ | 🟡 Жёлтый | Порт занят, нечёткий механизм | "Порт используется. HAProxy может обеспечить совместную работу, но убедитесь что SNI/Path отличаются." | ⚠️ Confirm |
| **Error** | ✗ | 🔴 Красный | Несовместимый протокол | "Порт уже используется и НЕ может быть совместно использован. Выберите другой порт." | ❌ Block |

### 5.4 Frontend Component (Preact)

```tsx
// File: frontend/src/components/inbound/PortValidationField.tsx

import { h } from 'preact';
import { useState, useCallback } from 'preact/hooks';
import { debounce } from 'lodash-es';

interface PortValidationProps {
    value: number;
    onChange: (port: number) => void;
    protocol: string;
    transport: string;
    coreType: string;
    listenAddress?: string;
}

type ValidationState = {
    status: 'idle' | 'checking' | 'success' | 'warning' | 'error';
    message: string;
    action: 'allow' | 'confirm' | 'block';
    canShare: boolean;
    conflicts?: Array<{
        inboundId: number;
        inboundName: string;
        protocol: string;
        transport: string;
        canShare: boolean;
    }>;
};

export function PortValidationField({
    value,
    onChange,
    protocol,
    transport,
    coreType,
    listenAddress = '0.0.0.0',
}: PortValidationProps) {
    const [state, setState] = useState<ValidationState>({
        status: 'idle',
        message: '',
        action: 'allow',
        canShare: false,
    });
    
    // Debounced validation (500ms)
    const validatePort = useCallback(
        debounce(async (port: number) => {
            if (!port || port < 1 || port > 65535) {
                setState({
                    status: 'error',
                    message: 'Порт должен быть от 1 до 65535',
                    action: 'block',
                    canShare: false,
                });
                return;
            }
            
            setState(prev => ({ ...prev, status: 'checking' }));
            
            try {
                const result = await fetch('/api/inbounds/check-port', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        port,
                        listen: listenAddress,
                        protocol,
                        transport,
                        coreType,
                    }),
                }).then(r => r.json());
                
                setState({
                    status: result.severity === 'error' ? 'error' : 
                           result.severity === 'warning' ? 'warning' : 'success',
                    message: result.message,
                    action: result.action,
                    canShare: result.canSharePort,
                    conflicts: result.conflicts,
                });
            } catch (e) {
                setState({
                    status: 'error',
                    message: 'Ошибка проверки порта',
                    action: 'block',
                    canShare: false,
                });
            }
        }, 500),
        [protocol, transport, coreType, listenAddress]
    );
    
    const handleChange = (newPort: number) => {
        onChange(newPort);
        validatePort(newPort);
    };
    
    const getStatusColor = () => {
        switch (state.status) {
            case 'success': return '#52c41a';
            case 'warning': return '#faad14';
            case 'error': return '#f5222d';
            case 'checking': return '#1890ff';
            default: return undefined;
        }
    };
    
    const getIcon = () => {
        switch (state.status) {
            case 'success': return '✓';
            case 'warning': return '⚠️';
            case 'error': return '✗';
            case 'checking': return '⏳';
            default: return '';
        }
    };
    
    const isBlocked = state.action === 'block';
    
    return (
        <div class="port-validation-field">
            <label>Порт</label>
            <div class="input-wrapper">
                <input
                    type="number"
                    value={value}
                    onInput={e => handleChange(Number(e.currentTarget.value))}
                    min={1}
                    max={65535}
                    style={{ borderColor: getStatusColor() }}
                    disabled={state.status === 'checking'}
                />
                {state.status === 'checking' && <span class="spinner">⏳</span>}
            </div>
            
            {state.message && (
                <div 
                    class={`validation-message ${state.status}`}
                    style={{ color: getStatusColor() }}
                >
                    <span class="icon">{getIcon()}</span>
                    {state.message}
                    
                    {state.conflicts && state.conflicts.length > 0 && (
                        <div class="conflict-details">
                            <small>Существующие инбаунды на этом порту:</small>
                            <ul>
                                {state.conflicts.map(c => (
                                    <li key={c.inboundId}>
                                        {c.inboundName} ({c.protocol}/{c.transport})
                                        {c.canShare && <span class="badge badge-success">✓ HAProxy OK</span>}
                                        {!c.canShare && <span class="badge badge-error">✗ Несовместим</span>}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}
                </div>
            )}
            
            {state.status === 'warning' && state.action === 'confirm' && (
                <div class="confirm-actions">
                    <button 
                        class="btn-secondary" 
                        onClick={() => { onChange(0); validatePort(0); }}
                    >
                        Выбрать другой порт
                    </button>
                    <button 
                        class="btn-primary" 
                        onClick={() => {/* proceed with creation */}}
                    >
                        Создать с HAProxy
                    </button>
                </div>
            )}
            
            {isBlocked && (
                <div class="error-help">
                    <small>
                        UDP/QUIC протоколы (Hysteria2, TUIC) должны использовать отдельные порты.
                        Используйте порт 8444+ для таких инбаундов.
                    </small>
                </div>
            )}
        </div>
    );
}
```

### 5.5 API Endpoint

```go
// File: backend/internal/api/handlers/port_validation.go

package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/isolate-project/isolate-panel/backend/internal/haproxy"
)

// CheckPortAvailability validates if a port can be used for a new inbound
func (h *Handler) CheckPortAvailability(c *fiber.Ctx) error {
    var req struct {
        Port      int    `json:"port"`
        Listen    string `json:"listen"`
        Protocol  string `json:"protocol"`
        Transport string `json:"transport"`
        CoreType  string `json:"coreType"`
    }
    
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }
    
    // Validate port range
    if req.Port < 1 || req.Port > 65535 {
        return c.Status(400).JSON(fiber.Map{
            "error": "Port must be between 1 and 65535",
        })
    }
    
    // Fetch all active inbounds
    var inbounds []haproxy.Inbound
    if err := h.db.Where("is_active = ?", true).Find(&inbounds).Error; err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to fetch inbounds",
        })
    }
    
    // Run validation
    validator := haproxy.NewPortValidator()
    result := validator.ValidatePortConflict(
        req.Port,
        req.Listen,
        req.Protocol,
        req.Transport,
        req.CoreType,
        inbounds,
    )
    
    return c.JSON(result)
}
```

---

## 6. ДЕТАЛЬНАЯ РЕАЛИЗАЦИЯ

### 6.1 Domain Models

**PortAssignment (динамическое назначение):**
```go
// File: backend/internal/haproxy/types.go

type PortAssignment struct {
    ID              uint      `gorm:"primaryKey" json:"id"`
    InboundID       uint      `json:"inbound_id" gorm:"uniqueIndex;not null"`
    
    // User selection
    UserListenPort  int       `json:"user_listen_port" gorm:"index;not null"`
    UserListenAddr  string    `json:"user_listen_addr" gorm:"default:'0.0.0.0'"`
    
    // System assignment
    BackendPort     int       `json:"backend_port" gorm:"not null"`
    CoreType        string    `json:"core_type" gorm:"index;not null"`
    
    // HAProxy routing
    UseHAProxy      bool      `json:"use_haproxy" gorm:"default:true"`
    SNIMatch        string    `json:"sni_match,omitempty"`
    PathMatch       string    `json:"path_match,omitempty"`
    
    // Settings
    SendProxyProtocol bool    `json:"send_proxy_protocol" gorm:"default:false"`
    ProxyProtocolVersion int  `json:"proxy_protocol_version,omitempty"`
    
    IsActive        bool      `json:"is_active" gorm:"default:true"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

**PortPool Manager:**
```go
// File: backend/internal/haproxy/port_pool.go

type PortPool struct {
    Start     int
    End       int
    Used      map[int]bool
    mu        sync.RWMutex
}

// AllocatePort назначает свободный backend порт
func (p *PortPool) AllocatePort() (int, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    for port := p.Start; port <= p.End; port++ {
        if !p.Used[port] {
            p.Used[port] = true
            return port, nil
        }
    }
    
    return 0, fmt.Errorf("no available ports in pool %d-%d", p.Start, p.End)
}

// ReleasePort освобождает порт
func (p *PortPool) ReleasePort(port int) {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.Used, port)
}
```

### 6.2 Template Engine

```go
// File: backend/internal/haproxy/template.go

type TemplateFuncs struct{}

func (tf TemplateFuncs) FuncMap() template.FuncMap {
    return template.FuncMap{
        "join":        strings.Join,
        "backendMode": tf.backendMode,
    }
}

func (tf TemplateFuncs) backendMode(coreType string) string {
    // All cores use TCP mode for PROXY protocol compatibility
    return "tcp"
}

// Generate creates HAProxy config from assignments
func (g *Generator) Generate(assignments []PortAssignment) (string, error) {
    cfg := &FullConfig{
        Global:   g.defaultGlobal(),
        Defaults: g.defaultDefaults(),
    }
    
    // Group by user listen port
    portGroups := make(map[int][]PortAssignment)
    for _, a := range assignments {
        if !a.UseHAProxy {
            continue
        }
        portGroups[a.UserListenPort] = append(portGroups[a.UserListenPort], a)
    }
    
    // Create frontends
    for port, group := range portGroups {
        frontend := FrontendConfig{
            Name:        fmt.Sprintf("ft_user_%d", port),
            BindAddress: "*",
            BindPort:    port,
            Mode:        "tcp",
            TLSEnabled:  true,
            TLSCertPath: "/etc/certs/fullchain.pem",
            SNIInspectionDelay: 5 * time.Second,
            RateLimitEnabled:   true,
            RateLimitRequests:  100,
            RateLimitWindow:    10 * time.Second,
            DefaultBackend:     "bk_default_web",
        }
        
        // Create routes
        for _, a := range group {
            route := RouteRule{
                Name:        fmt.Sprintf("inbound_%d", a.InboundID),
                BackendName: fmt.Sprintf("bk_%s_%d", a.CoreType, a.BackendPort),
                BackendPort: a.BackendPort,
                Priority:    100,
            }
            
            if a.SNIMatch != "" {
                route.Type = "sni"
                route.Match = a.SNIMatch
            } else if a.PathMatch != "" {
                route.Type = "path"
                route.Match = a.PathMatch
                frontend.Mode = "http"
            }
            
            frontend.Routes = append(frontend.Routes, route)
        }
        
        cfg.Frontends = append(cfg.Frontends, frontend)
    }
    
    // Create backends
    for _, a := range assignments {
        if !a.UseHAProxy {
            continue
        }
        
        useProxyV2 := a.SendProxyProtocol && a.CoreType == "xray"
        useXFF := !useProxyV2 && (a.CoreType == "singbox" || a.CoreType == "mihomo")
        
        backend := BackendConfig{
            Name:            fmt.Sprintf("bk_%s_%d", a.CoreType, a.BackendPort),
            Type:            a.CoreType,
            ServerAddress:   "127.0.0.1",
            BackendPort:     a.BackendPort,
            SendProxyProtocol: useProxyV2,
            ProxyProtocolVersion: a.ProxyProtocolVersion,
            UseXForwardedFor: useXFF,
            CheckInterval:   5 * time.Second,
        }
        
        cfg.Backends = append(cfg.Backends, backend)
    }
    
    return g.tmpl.Execute(cfg)
}
```

### 6.3 Core Configuration Updates

**Xray Config with Dynamic Port:**
```go
// File: backend/internal/cores/xray/config.go

func (c *XrayConfig) GenerateInboundConfig(inbound models.Inbound, assignment haproxy.PortAssignment) (*Inbound, error) {
    protocol := mapXrayProtocol(inbound.Protocol)
    
    config := &Inbound{
        Tag:    fmt.Sprintf("inbound_%d", inbound.ID),
        Port:   assignment.BackendPort,  // Use dynamic backend port
        Listen: "127.0.0.1",             // Only accept from HAProxy
        Protocol: protocol,
    }
    
    // Enable PROXY protocol for TCP-based transports
    if assignment.SendProxyProtocol && isTCPTransport(inbound.Transport) {
        if config.StreamSettings == nil {
            config.StreamSettings = &StreamSettings{}
        }
        if config.StreamSettings.Sockopt == nil {
            config.StreamSettings.Sockopt = &SocketConfig{}
        }
        config.StreamSettings.Sockopt.AcceptProxyProtocol = true
    }
    
    return config, nil
}
```

---

## 7. ПЛАН РЕАЛИЗАЦИИ (6 НЕДЕЛЬ)

### Неделя 1: Domain Models + Database

**Backend:**
- [ ] `PortAssignment` struct с динамическим назначением
- [ ] Database migrations: `port_assignments`, `direct_ports` tables
- [ ] Port pool manager (40000-50000 диапазон)
- [ ] Port allocation service

**QA:** Миграции применяются без ошибок, таблицы создаются

### Неделя 2: Template Engine + Manager

**Backend:**
- [ ] Template engine для динамической генерации frontend per port
- [ ] Manager: generate, validate (`haproxy -c`), reload (hitless)
- [ ] Runtime API client: Data Plane API через Unix socket
- [ ] Integration with CoreAdapter

**QA:** Конфиг генерируется, валидируется, reload работает

### Неделя 3: Multi-Port + Cross-Core

**Backend:**
- [ ] Dynamic frontend generation для любых портов
- [ ] SNI-based routing на все 3 ядра
- [ ] Path-based routing для HTTP транспортов
- [ ] Backend configuration per core type
- [ ] Integration with CoreAdapter

**QA:** Несколько инбаундов на одном порту с разными ядрами работают

### Неделя 4: Core Integration + Docker

**Backend:**
- [ ] Xray: PROXY v2 (TCP/WS/XHTTP only, НЕ gRPC)
- [ ] Sing-box: X-Forwarded-For headers (НЕТ PROXY v2)
- [ ] Mihomo: X-Forwarded-For headers (НЕТ PROXY v2)
- [ ] Core config generation with dynamic ports

**Docker:**
- [ ] HAProxy Dockerfile (Alpine 3.3)
- [ ] Docker Compose: `network_mode: host`
- [ ] Entrypoint с bootstrap и validation

**QA:** Все 3 ядра стартуют с правильными портами

### Неделя 5: Smart Warning UI

**Backend:**
- [ ] `PortValidator` с severity levels
- [ ] API endpoint: `POST /api/inbounds/check-port`
- [ ] Debounced validation (500ms)

**Frontend (Preact):**
- [ ] `PortValidationField` компонент
- [ ] Real-time validation с debounce
- [ ] Severity states: info (зелёный), warning (жёлтый), error (красный)
- [ ] Modal confirmation для warning state
- [ ] Conflict details display

**QA:** Валидация работает в реальном времени, сообщения на русском

### Неделя 6: Testing & Documentation

**Testing:**
- [ ] Unit tests: template generation, validation logic
- [ ] Integration tests: SNI routing, multi-port scenarios
- [ ] Cross-core routing tests (Xray/Sing/Mihomo на одном порту)
- [ ] UI validation tests (e2e)

**Documentation:**
- [ ] API documentation (`/api/docs`)
- [ ] Docker deployment guide
- [ ] Troubleshooting: common errors
- [ ] Migration guide from single-port

**Monitoring:**
- [ ] Prometheus metrics endpoint
- [ ] Grafana dashboard template
- [ ] Alert rules для HAProxy

**QA:** Все тесты проходят, документация полная

---

## 📚 Ссылки и источники

### Protocol Documentation
- **Xray Protocols**: `/backend/internal/cores/xray/config.go` (mapXrayProtocol)
- **Sing-box Protocols**: `/backend/internal/cores/singbox/config.go` (mapSingboxProtocol)
- **Mihomo Protocols**: `/backend/internal/cores/mihomo/config.go` (mapMihomoProtocol)
- **Protocol Registry**: `/backend/internal/protocol/protocols.go`

### HAProxy Issues
- [Xray gRPC PROXY Issue #2204](https://github.com/XTLS/Xray-core/issues/2204)
- [Sing-box PROXY deprecated](https://sing-box.sagernet.org/deprecated/)
- [HAProxy QUIC Issue #2748](https://github.com/haproxy/haproxy/issues/2748)

### Core Versions
- Xray: v26.3.27
- Sing-box: v1.13.8
- Mihomo: v1.19.23

---

**Вердикт**: ✅ **ВСЕ ЗАДУМКИ РЕАЛИЗУЕМЫ**

С ограничениями:
- UDP/QUIC протоколы на direct портах (не через HAProxy)
- PROXY protocol v2 только для Xray TCP/WS/HUP (не для gRPC/XHTTP)
- Sing-box и Mihomo без PROXY protocol (X-Forwarded-For вместо этого)
- Динамическое распределение портов (40000-50000 пул)
- Кросс-ядерная маршрутизация на любом порту
- Smart Warning UI валидация в реальном времени

**Timeline**: 6 недель до production-ready мульти-порт + кросс-ядерной + Smart Warning системы.
