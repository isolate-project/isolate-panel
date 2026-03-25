# Isolate Panel Architecture

System architecture documentation for Isolate Panel v0.1.0

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Backend Architecture](#backend-architecture)
3. [Frontend Architecture](#frontend-architecture)
4. [Core Integration](#core-integration)
5. [Security Architecture](#security-architecture)
6. [Data Flow](#data-flow)
7. [Deployment Architecture](#deployment-architecture)

---

## System Overview

Isolate Panel is a lightweight proxy core management panel for Xray, Sing-box, and Mihomo.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     User Browser                         │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │           Preact Frontend (SPA)                  │  │
│  │  - Dashboard                                     │  │
│  │  - User Management                               │  │
│  │  - Inbound/Outbound Management                   │  │
│  │  - Settings                                      │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           │ HTTP/HTTPS
                           │
┌─────────────────────────────────────────────────────────┐
│              Isolate Panel Backend (Go + Fiber)         │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │              API Layer (Fiber v3)                │  │
│  │  - Authentication Middleware                     │  │
│  │  - Rate Limiting                                 │  │
│  │  - Request/Response Handlers                     │  │
│  └──────────────────────────────────────────────────┘  │
│                          │                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │              Service Layer                       │  │
│  │  - UserService                                   │  │
│  │  - InboundService                                │  │
│  │  - OutboundService                               │  │
│  │  - CoreLifecycleManager                          │  │
│  │  - TrafficCollector                              │  │
│  │  - NotificationService                           │  │
│  │  - BackupService                                 │  │
│  └──────────────────────────────────────────────────┘  │
│                          │                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │           Data Access Layer (GORM)               │  │
│  │  - SQLite Database                               │  │
│  │  - Migrations                                    │  │
│  │  - Models                                        │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           │ Supervisord API
                           │
┌─────────────────────────────────────────────────────────┐
│              Proxy Cores (Supervisord)                  │
│                                                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐       │
│  │  Sing-box  │  │    Xray    │  │   Mihomo   │       │
│  │  (1.13.3)  │  │  (26.2.6)  │  │  (1.19.21) │       │
│  └────────────┘  └────────────┘  └────────────┘       │
└─────────────────────────────────────────────────────────┘
```

---

## Backend Architecture

### Package Structure

```
backend/
├── cmd/
│   ├── server/           # Main entry point
│   └── migrate/          # Database migrations CLI
├── internal/
│   ├── acme/             # ACME/Let's Encrypt integration
│   ├── api/              # HTTP API handlers
│   │   ├── auth.go
│   │   ├── users.go
│   │   ├── inbounds.go
│   │   ├── cores.go
│   │   └── settings.go
│   ├── auth/             # Authentication logic
│   │   ├── token.go      # JWT token service
│   │   └── password.go   # Argon2id password hashing
│   ├── config/           # Configuration management
│   ├── cores/            # Core configuration generators
│   │   ├── singbox/
│   │   ├── xray/
│   │   └── mihomo/
│   ├── database/         # Database layer
│   │   ├── migrations/
│   │   ├── seeds/
│   │   └── db.go
│   ├── middleware/       # HTTP middleware
│   │   ├── auth.go
│   │   ├── ratelimit.go
│   │   └── cors.go
│   ├── models/           # GORM models
│   ├── protocol/         # Protocol definitions
│   ├── scheduler/        # Background jobs
│   ├── services/         # Business logic
│   │   ├── user_service.go
│   │   ├── inbound_service.go
│   │   ├── core_lifecycle.go
│   │   ├── traffic_collector.go
│   │   └── ...
│   └── stats/            # Statistics collection
└── tests/                # Test files
    ├── testutil/
    ├── unit/
    ├── integration/
    └── e2e/
```

### Service Layer Pattern

```
┌─────────────────────────────────────────┐
│          API Handler (Fiber)            │
│  - HTTP request/response                │
│  - Input validation                     │
│  - Authentication/Authorization         │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│          Service Layer                  │
│  - Business logic                       │
│  - Transaction management               │
│  - Cross-cutting concerns               │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│        Data Access Layer (GORM)         │
│  - Database operations                  │
│  - ORM mapping                          │
│  - Migrations                           │
└─────────────────────────────────────────┘
```

### Key Services

**UserService:**
- User CRUD operations
- Credential generation (UUID, tokens)
- Traffic quota management
- Expiry date management

**InboundService:**
- Inbound proxy configuration
- Protocol-specific config generation
- Port management
- User-inbound mapping

**CoreLifecycleManager:**
- Core process management (start/stop/restart)
- Supervisord integration
- Lazy loading (autostart=false)
- Health monitoring

**TrafficCollector:**
- Periodic stats collection (60s/10s intervals)
- Traffic aggregation
- Quota enforcement integration
- Monitoring mode support (lite/full)

**NotificationService:**
- Webhook notifications
- Telegram bot integration
- Event-based notifications
- Retry logic

---

## Frontend Architecture

### Technology Stack

- **Framework:** Preact 10.x (lightweight React alternative)
- **Build Tool:** Vite 6.x
- **UI Components:** Custom components with Tailwind CSS
- **State Management:** Zustand (lightweight state management)
- **HTTP Client:** Axios
- **Routing:** Preact Router
- **i18n:** react-i18next

### Component Structure

```
frontend/
├── src/
│   ├── api/                # API client
│   │   └── endpoints/      # API endpoint definitions
│   ├── components/
│   │   ├── ui/             # Reusable UI components
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Modal.tsx
│   │   │   └── ...
│   │   └── layout/         # Layout components
│   │       ├── Sidebar.tsx
│   │       ├── PageLayout.tsx
│   │       └── PageHeader.tsx
│   ├── pages/              # Page components
│   │   ├── Dashboard.tsx
│   │   ├── Users.tsx
│   │   ├── Inbounds.tsx
│   │   ├── Settings.tsx
│   │   └── ...
│   ├── stores/             # Zustand stores
│   │   ├── themeStore.ts
│   │   ├── toastStore.ts
│   │   └── authStore.ts
│   ├── i18n/               # Internationalization
│   │   └── locales/        # Translation files (en/ru/zh)
│   ├── utils/              # Utility functions
│   ├── app.tsx             # Root component
│   └── main.tsx            # Entry point
└── tests/                  # Test files
```

### State Management Flow

```
┌─────────────────────────────────────────┐
│           Component (Preact)            │
│  - Page/User Interface                  │
│  - User Interactions                    │
└─────────────────────────────────────────┘
         │                    │
         │                    │
         ▼                    ▼
┌─────────────────┐  ┌─────────────────┐
│  API Client     │  │  Zustand Store  │
│  (Axios)        │  │  (State)        │
└─────────────────┘  └─────────────────┘
         │                    │
         │                    │
         ▼                    ▼
┌─────────────────────────────────────────┐
│          Backend API (Fiber)            │
└─────────────────────────────────────────┘
```

---

## Core Integration

### Lazy Loading Architecture

Cores are NOT started automatically. They use lazy loading:

```ini
# Supervisord Configuration
[program:singbox]
command=/usr/local/bin/sing-box run -c /app/data/cores/singbox/config.json
autostart=false    # ← Lazy loading
autorestart=true

[program:xray]
command=/usr/local/bin/xray run -c /app/data/cores/xray/config.json
autostart=false    # ← Lazy loading
autorestart=true

[program:mihomo]
command=/usr/local/bin/mihomo -f /app/data/cores/mihomo/config.yaml
autostart=false    # ← Lazy loading
autorestart=true
```

### Core Lifecycle Flow

```
1. User creates inbound via UI
        │
        ▼
2. Backend generates core-specific config
        │
        ▼
3. CoreLifecycleManager checks if core is running
        │
        ├─► Not running → Start core via Supervisord
        │
        └─► Running → Apply config reload
        │
        ▼
4. Core starts/reloads with new configuration
        │
        ▼
5. TrafficCollector begins collecting stats
```

### Protocol Support

**Sing-box:**
- VMess
- VLESS
- Trojan
- Shadowsocks
- Hysteria2
- TUIC

**Xray:**
- VMess
- VLESS
- Trojan
- Shadowsocks
- Reality

**Mihomo:**
- VMess
- VLESS
- Trojan
- Shadowsocks
- Hysteria2
- WireGuard

---

## Security Architecture

### Authentication Flow

```
┌─────────┐                  ┌─────────┐
│  User   │                  │ Backend │
└────┬────┘                  └────┬────┘
     │                            │
     │  POST /api/auth/login      │
     │  {username, password}      │
     │───────────────────────────>│
     │                            │
     │  Argon2id password verify  │
     │                            │
     │  Generate JWT tokens       │
     │  (access + refresh)        │
     │                            │
     │  Response with tokens      │
     │<───────────────────────────│
     │                            │
     │  Subsequent requests       │
     │  Authorization: Bearer     │
     │───────────────────────────>│
     │                            │
     │  Validate JWT signature    │
     │  Check expiration          │
     │                            │
     │  Response with data        │
     │<───────────────────────────│
```

### Password Hashing (Argon2id)

```go
// Parameters
Time:    3 iterations
Memory:  64 MB
Threads: 4 parallel threads
KeyLen:  32 bytes
SaltLen: 16 bytes

// Example hash
$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG
```

### JWT Token Structure

**Access Token (15 minutes):**
```json
{
  "sub": "1",
  "username": "admin",
  "is_super_admin": true,
  "exp": 1648234567,
  "iat": 1648233667
}
```

**Refresh Token (7 days):**
```json
{
  "sub": "1",
  "username": "admin",
  "exp": 1648838467,
  "iat": 1648233667
}
```

### Rate Limiting

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/api/auth/login` | 5 requests | 1 minute |
| `/sub/:token` | 30 requests | 1 minute |
| All other endpoints | 100 requests | 1 minute |

### Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000
Content-Security-Policy: default-src 'self'
```

---

## Data Flow

### User Creation Flow

```
1. Admin creates user via UI
        │
        ▼
2. Frontend: POST /api/users
        │
        ▼
3. Backend: Validate input
        │
        ▼
4. Generate credentials:
   - UUID v4
   - Subscription token (32 bytes)
   - TUIC token (optional)
        │
        ▼
5. Hash password (Argon2id)
        │
        ▼
6. Save to database (SQLite)
        │
        ▼
7. Send notification (optional)
        │
        ▼
8. Return user data with credentials
```

### Traffic Collection Flow

```
┌─────────────────────────────────────────┐
│      TrafficCollector (Goroutine)       │
│                                         │
│  while running:                         │
│    sleep(interval)  // 60s or 10s       │
│    for each core:                       │
│      collect_stats()                    │
│    for each user:                       │
│      update_traffic_used()              │
│      check_quota()                      │
└─────────────────────────────────────────┘
           │
           │ Stats API
           ▼
┌─────────────────────────────────────────┐
│         Proxy Cores                     │
│  - Sing-box stats API                   │
│  - Xray stats API                       │
│  - Mihomo stats API                     │
└─────────────────────────────────────────┘
```

---

## Deployment Architecture

### Docker Container Structure

```
┌─────────────────────────────────────────┐
│         Isolate Panel Container         │
│                                         │
│  /usr/local/bin/isolate-panel  ← Backend│
│  /var/www/html                 ← Frontend│
│  /usr/local/bin/xray           ← Core   │
│  /usr/local/bin/sing-box       ← Core   │
│  /usr/local/bin/mihomo         ← Core   │
│  /app/data                     ← Volume │
│  /var/log/supervisor           ← Logs   │
│                                         │
│  Supervisord (PID 1)                    │
│  ├─ isolate-panel (backend)             │
│  ├─ singbox (lazy)                      │
│  ├─ xray (lazy)                         │
│  └─ mihomo (lazy)                       │
└─────────────────────────────────────────┘
```

### Network Architecture

```
Internet
    │
    ├── Port 443 ──► Proxy Cores (Xray/Sing-box/Mihomo)
    │
    └── Port 8080 ──► Isolate Panel (localhost only)
                           │
                           └── SSH Tunnel ──► Admin Browser
```

### Volume Structure

```
./data/
├── isolate-panel.db          # SQLite database
├── cores/
│   ├── singbox/
│   │   └── config.json
│   ├── xray/
│   │   └── config.json
│   └── mihomo/
│       └── config.yaml
├── backups/
│   └── backup-20260325.db
├── certificates/
│   └── example.com.crt
├── geo/
│   ├── geoip.dat
│   └── geosite.dat
└── warp/
    └── wgcf.conf
```

---

## Monitoring Architecture

### Monitoring Modes

**Lite Mode (default):**
- Collection interval: 60 seconds
- RAM usage: ~30MB
- Accuracy: ±1 minute
- Use case: Low-resource servers

**Full Mode:**
- Collection interval: 10 seconds
- RAM usage: ~100MB
- Accuracy: ±10 seconds
- Use case: Production servers requiring real-time stats

### Health Check Architecture

```
┌─────────────────────────────────────────┐
│      Docker HEALTHCHECK                 │
│  interval: 30s                          │
│  timeout: 5s                            │
│  retries: 3                             │
└─────────────────────────────────────────┘
           │
           │ GET /health
           ▼
┌─────────────────────────────────────────┐
│      Health Endpoint                    │
│                                         │
│  1. Check database connection (ping)    │
│  2. Check core status                   │
│  3. Calculate uptime                    │
│  4. Return status (healthy/unhealthy)   │
└─────────────────────────────────────────┘
```

---

**Architecture Version:** 0.1.0  
**Last Updated:** March 2026
