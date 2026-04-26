# Isolate Panel — Architecture

v1.0.0

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Backend Architecture](#backend-architecture)
3. [Frontend Architecture](#frontend-architecture)
4. [Real-time Data Flow](#real-time-data-flow)
5. [Security Architecture](#security-architecture)
6. [Core Integration](#core-integration)
7. [Background Workers](#background-workers)
8. [Deployment Architecture](#deployment-architecture)

---

## System Overview

Isolate Panel is a single-binary web application (Go backend + Preact SPA) accessible **only via SSH tunnel**. It never listens on public interfaces.

```
┌──────────────────────────────────────────────────────────────────────┐
│  Admin Browser (via SSH tunnel localhost:8080)                        │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Preact SPA (lazy-loaded routes, Zustand, Tailwind CSS 4)    │   │
│  │  WebSocket ──────────────────────────────────────────────►   │   │
│  │  REST API  ──────────────────────────────────────────────►   │   │
│  └──────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
                              │ SSH tunnel
┌──────────────────────────────────────────────────────────────────────┐
│  VPS / Server                                                         │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Isolate Panel Backend (Go 1.25 + Fiber v3)                  │   │
│  │                                                               │   │
│  │  Middleware chain:                                            │   │
│  │  SecurityHeaders → Recovery → CORS → RequestLogger           │   │
│  │                                                               │   │
│  │  ┌────────────┐  ┌───────────────┐  ┌──────────────────┐   │   │
│  │  │ HTTP       │  │ WebSocket     │  │ Subscription     │   │   │
│  │  │ Handlers   │  │ Hub           │  │ Endpoints        │   │   │
│  │  │ /api/*     │  │ /api/ws/*     │  │ /sub/:token      │   │   │
│  │  └─────┬──────┘  └──────┬────────┘  └──────────────────┘   │   │
│  │        │                │                                    │   │
│  │  ┌─────▼────────────────▼──────────────────────────────┐   │   │
│  │  │  Services Layer                                       │   │   │
│  │  │  UserService, CoreLifecycle, SubscriptionService,    │   │   │
│  │  │  BackupService, CertificateService, WARPService,     │   │   │
│  │  │  TrafficCollector, ConnectionTracker, QuotaEnforcer  │   │   │
│  │  └─────────────────────────┬───────────────────────────┘   │   │
│  │                             │                                │   │
│  │  ┌──────────────────────────▼──────────────────────────┐   │   │
│  │  │  GORM + SQLite  (data/isolate.db)                    │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Supervisord                                                   │   │
│  │  ├── xray (process)                                           │   │
│  │  ├── sing-box (process)                                       │   │
│  │  └── mihomo (process)                                         │   │
│  └──────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Backend Architecture

### Package Structure

```
backend/
├── cmd/server/main.go          # Entry point (startup, graceful shutdown)
└── internal/
    ├── app/
    │   ├── providers.go        # DI: wires all services and handlers into App{}
    │   ├── routes.go           # All HTTP routes registration
    │   └── background.go       # StartWorkers / StopWorkers
    ├── api/                    # Fiber v3 HTTP handlers (one file per domain)
    │   ├── auth.go             # Login, Refresh, Logout, Me, TOTP endpoints
    │   ├── users.go            # User CRUD + credential management
    │   ├── cores.go            # Core start/stop/restart/status
    │   ├── inbounds.go         # Inbound CRUD + user assignment
    │   ├── outbounds.go        # Outbound CRUD
    │   ├── certificates.go     # ACME + manual cert management
    │   ├── stats.go            # Traffic stats, connections, dashboard stats
    │   ├── subscriptions.go    # Subscription URL generation
    │   ├── settings.go         # Monitoring, traffic reset schedule
    │   ├── warp.go             # WARP routes + Geo rules
    │   ├── backup.go           # Backup CRUD + schedule
    │   ├── notifications.go    # Notification history + settings
    │   ├── audit.go            # Audit log listing
    │   ├── protocols.go        # Protocol schema registry
    │   └── websocket.go        # DashboardHub — WS broadcast every 5s
    ├── services/               # Business logic
    │   ├── user_service.go
    │   ├── core_lifecycle.go   # Start/stop cores via Supervisord
    │   ├── config_service.go   # Generate core config from DB state
    │   ├── traffic_collector.go
    │   ├── connection_tracker.go
    │   ├── quota_enforcer.go   # Auto traffic quota enforcement + reset
    │   ├── subscription_service.go
    │   ├── backup_service.go   # AES-256-GCM streaming encrypt/decrypt
    │   ├── certificate_service.go
    │   ├── warp_service.go
    │   ├── geo_service.go
    │   ├── notification_service.go
    │   ├── settings_service.go
    │   └── audit_service.go
    ├── models/                 # GORM models
    ├── database/
    │   ├── migrations/         # 000001–000029 SQL migrations (append-only)
    │   └── seeds/              # Dev data seeders
    ├── middleware/
    │   ├── auth.go             # JWT validation (AuthMiddleware)
    │   ├── rate_limiter.go     # LoginRateLimiter, AuthRateLimiter, HeavyRL
    │   ├── audit.go            # AuditAction middleware
    │   ├── security.go         # CSP, X-Frame-Options, security headers
    │   ├── validator.go        # BindAndValidate[T] generic helper
    │   ├── cors.go
    │   ├── recovery.go
    │   └── logger.go
    ├── scheduler/
    │   ├── backup_scheduler.go         # Cron-based backup scheduler
    │   └── traffic_reset_scheduler.go  # Weekly / monthly traffic reset
    ├── auth/
    │   └── token_service.go    # JWT access + refresh token management
    ├── cache/
    │   └── manager.go          # Ristretto in-memory cache
    ├── cores/
    │   └── core_manager.go     # Supervisord XML-RPC client
    ├── config/                 # Viper config loading
    ├── logger/                 # Zerolog initialization
    └── version/                # Version string (injected via ldflags)
```

### Dependency Injection

All dependencies are wired in `internal/app/providers.go`:

```
NewApp(cfg, db) → *App
  ├── Auth:         TokenService, AuthHandler
  ├── Users:        UserService, UsersHandler
  ├── Cores:        CoreManager, CoresHandler
  ├── Inbounds:     InboundService, InboundsHandler
  ├── Outbounds:    OutboundService, OutboundsHandler
  ├── Stats:        TrafficCollector, ConnectionTracker, StatsHandler
  ├── Certs:        CertificateService, CertificatesHandler
  ├── Subscriptions: SubscriptionService, SubscriptionsHandler
  ├── Settings:     SettingsService, SettingsHandler
  ├── WARP:         WARPService, GeoService, WarpHandler
  ├── Backup:       BackupService, BackupScheduler, BackupHandler
  ├── Notifications: NotificationService, NotificationHandler
  ├── Audit:        AuditService, AuditHandler
  ├── Quota:        QuotaEnforcer
  ├── DashboardHub: WebSocket broadcast hub
  └── TrafficResetSched: Traffic reset scheduler
```

### Request flow

```
HTTP Request
  → SecurityHeaders middleware
  → Recovery middleware
  → CORS middleware
  → RequestLogger middleware
  → Auth middleware (for /api/* protected routes)
  → Rate limiter middleware
  → Handler function
  → Service layer
  → GORM / SQLite
  → JSON response
```

---

## Frontend Architecture

```
frontend/src/
├── app.tsx                     # Root: lazy-loaded routes, Suspense, ErrorBoundary
├── pages/                      # 17 page components (all lazy-loaded via preact/compat)
│   ├── Dashboard.tsx           # Real-time stats (WebSocket + polling fallback)
│   ├── Users.tsx / UserCreate / UserEdit
│   ├── Cores.tsx
│   ├── Inbounds.tsx / InboundCreate / InboundEdit
│   ├── Outbounds.tsx / OutboundCreate / OutboundEdit
│   ├── Certificates.tsx
│   ├── Backups.tsx
│   ├── Settings.tsx
│   ├── AuditLogs.tsx
│   ├── Login.tsx
│   └── NotFound.tsx
├── components/
│   ├── layout/                 # PageLayout, PageHeader, Sidebar, Navbar
│   ├── ui/                     # Atomic UI: Button, Card, Badge, Input, Table...
│   └── features/               # Domain-specific: RAMPanicButton, DashboardCharts...
├── hooks/
│   ├── useUsers.ts, useCores.ts, useInbounds.ts, ...  # Data-fetching hooks
│   ├── useWebSocket.ts         # Generic WS hook with reconnect
│   ├── useSystem.ts            # System resources polling
│   └── useConnections.ts
├── stores/
│   ├── authStore.ts            # Zustand: access/refresh tokens, admin info
│   ├── themeStore.ts           # Zustand: light/dark theme
│   └── toastStore.ts           # Zustand: toast notification queue
├── api/
│   ├── client.ts               # Axios instance, interceptors (token refresh)
│   └── endpoints/              # Typed endpoint functions per domain
├── i18n/                       # react-i18next: en/ru locale files
├── lib/
│   └── utils.ts                # cn() classname utility, formatters
└── types/                      # Shared TypeScript interfaces
```

### Code splitting

All 17 pages are lazy-loaded via `preact/compat`'s `lazy()`:

```tsx
const Dashboard = lazy(() =>
  import('./pages/Dashboard').then(m => ({ default: m.Dashboard }))
)
```

Vite creates per-route JS chunks, reducing the initial bundle from ~800 KB to ~120 KB.

---

## Real-time Data Flow

### Dashboard WebSocket

```
Frontend (Dashboard.tsx)
  useWebSocket('/api/ws/dashboard?token=<JWT>')
       │
       │  WebSocket upgrade
       ▼
Backend: DashboardHub.DashboardWS()
  → validate JWT from ?token= query param
  → wsUpgrader.Upgrade(c.RequestCtx(), handler)
  → register conn in hub
       │
       │  every 5 seconds
       ▼
DashboardHub.collectStats()
  → DB: COUNT users, active users, running cores
  → ConnectionTracker: active connections count
  → DB: SUM traffic_used_bytes
  → broadcast JSON to all connected clients
       │
       ▼
Frontend receives DashboardWSPayload
  → updates totalUsers, activeUsers, activeConnections, totalTraffic, runningCores
  → falls back to polling hooks when WS disconnected
```

### Traffic collection

```
TrafficCollector (goroutine, configurable interval)
  → polls each running core's API (Xray stats API, etc.)
  → updates users.traffic_used_bytes in DB
  → DataAggregator writes hourly/daily records to traffic_stats table
```

---

## Security Architecture

| Layer | Mechanism |
|---|---|
| Transport | SSH tunnel only (no public binding) |
| Authentication | JWT access tokens (15 min) + refresh tokens (7 days, stored hashed) |
| Passwords | Argon2id (memory=64MB, iterations=3, parallelism=4) |
| 2FA | TOTP (pquerna/otp, RFC 6238) |
| Authorization | JWT claims (admin_id, is_super_admin) |
| Rate limiting | Login: 5/min · Protected: 60/min · Heavy ops: 10/min |
| Input validation | `go-playground/validator/v10` via `BindAndValidate[T]` |
| Audit log | AuditAction middleware on all state-changing operations |
| HTTP headers | CSP, X-Frame-Options: DENY, X-XSS-Protection, Referrer-Policy |
| Subscriptions | Token-based, rate limited (10/hour), returns 404 on invalid token |

### JWT flow

```
POST /auth/login { username, password, totp_code? }
  → verify Argon2id hash
  → if TOTP enabled: validate TOTP code
  → generate access_token (JWT, 15min) + refresh_token (opaque, hashed in DB)
  → return { access_token, refresh_token, expires_in }

POST /auth/refresh { refresh_token }
  → lookup hashed token in DB
  → generate new access_token
  → return { access_token }

POST /auth/logout { refresh_token }
  → mark refresh token as revoked in DB
```

---

## Core Integration

Proxy cores (Xray, Sing-box, Mihomo) run as Supervisord-managed processes.

```
CoreManager (Supervisord XML-RPC client)
  ├── StartCore(name)    → supervisord.startProcess(name)
  ├── StopCore(name)     → supervisord.stopProcess(name)
  ├── RestartCore(name)  → supervisord.restartProcess(name)
  └── GetCoreStatus(name) → parse supervisord state

ConfigService
  ├── GenerateConfig(coreName) → builds JSON/YAML from DB inbounds + outbounds
  └── RegenerateAndReload(name) → generate + write config + restart core

TrafficCollector + ConnectionTracker
  └── poll Xray Stats API / Sing-box / Mihomo management APIs
```

---

## Background Workers

Started at launch, stopped gracefully on SIGTERM/SIGINT:

| Worker | Purpose | Interval |
|---|---|---|
| DashboardHub | Broadcast WS stats to connected clients | 5 s |
| TrafficCollector | Poll core traffic APIs, update user bytes | configurable |
| ConnectionTracker | Track active connections | configurable |
| DataAggregator | Aggregate raw stats into hourly/daily records | configurable |
| DataRetention | Delete old stats records | daily |
| BackupScheduler | Run scheduled backups (cron) | per schedule |
| TrafficResetScheduler | Reset all user traffic (weekly / monthly) | per schedule |
| QuotaEnforcer | Check quotas, expire users | 5 min |
| WARPService.AutoRefresh | Refresh WARP tokens | 24 h |
| GeoService.AutoUpdate | Download updated GeoIP/GeoSite DBs | 7 days |

---

## Deployment Architecture

### Production (Docker)

```
docker-compose up -d
  │
  └── isolate-panel container (Alpine Linux, non-root user isolate)
        ├── supervisord (PID 1)
        │     ├── isolate-server  → Go binary on :8080
        │     ├── xray            → /usr/local/bin/xray
        │     ├── singbox         → /usr/local/bin/sing-box
        │     └── mihomo          → /usr/local/bin/mihomo
        └── volumes
              ├── /data           → SQLite database
              ├── /etc/isolate-panel/certs   → certificates
              └── /var/log/isolate-panel     → log files
```

### Port bindings

| Port | Service | Binding |
|---|---|---|
| 8080 | Isolate Panel API | `127.0.0.1:8080` (localhost only) |
| Proxy ports | Core inbounds | `0.0.0.0:PORT` (defined per inbound) |

Admin access requires `ssh -L 8080:localhost:8080 user@server`.
