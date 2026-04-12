# Isolate Panel

> Lightweight proxy core management panel for Xray, Sing-box, and Mihomo

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/isolate-project/isolate-panel/actions/workflows/test.yml/badge.svg)](https://github.com/isolate-project/isolate-panel/actions)

Designed for VPS with limited resources (1 CPU / 1 GB RAM). Accessible **only via SSH tunnel** for maximum security — the panel never binds to a public interface.

---

## Features

| Category | What's included |
|---|---|
| **Proxy cores** | Xray, Sing-box, Mihomo — start / stop / restart via Supervisord |
| **Users** | CRUD, traffic quotas, expiry dates, auto quota enforcement, subscription links |
| **Inbounds / Outbounds** | 25+ protocol schemas (VLESS, VMess, Trojan, SS, TUIC, Hysteria2, …) with dynamic forms |
| **Subscriptions** | Auto-detect, Clash, Sing-box, QR code formats; short URLs |
| **Dashboard** | Real-time stats via WebSocket (active connections, traffic, cores), fallback polling |
| **Traffic charts** | 7-day traffic history, top-users chart |
| **Traffic reset** | Automatic weekly / monthly reset scheduler |
| **Certificates** | ACME/Let's Encrypt auto-provisioning, manual upload, renewal, revocation |
| **WARP** | Cloudflare WARP integration with auto-refresh |
| **GeoIP** | Automatic GeoIP database updates |
| **Backups** | Streaming AES-256-GCM encrypted backups, configurable retention policy |
| **Notifications** | Telegram and Webhook integrations with configurable triggers |
| **Audit log** | Immutable log of all admin actions (create / delete / start / stop …) |
| **2FA / TOTP** | TOTP-based two-factor authentication for admin login |
| **CLI** | Cobra-based CLI: `isolate-panel user list`, `core start xray`, `backup create` |
| **Security** | JWT + Argon2id, CSP headers, rate limiting, request validation, security headers |

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, Fiber v3, GORM, SQLite |
| Frontend | Preact 10, TypeScript 5, Vite 6, Tailwind CSS 4, Zustand |
| Auth | JWT (access + refresh tokens), Argon2id password hashing, TOTP (pquerna/otp) |
| Process mgmt | Supervisord (XML-RPC) |
| Deployment | Docker, Alpine Linux, multi-stage build |
| Logging | Zerolog (structured JSON) |

---

## Quick Start — Docker

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+

### One-liner install (recommended for VPS)

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/install.sh)
```

### Manual setup

```bash
mkdir -p /opt/isolate-panel && cd /opt/isolate-panel
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.production.yml -o docker-compose.yml
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/.env.example -o .env
nano .env   # set JWT_SECRET and ADMIN_PASSWORD
docker compose up -d
```

### Access via SSH tunnel

The panel only listens on `localhost:8080`. Open a tunnel from your local machine:

```bash
ssh -L 8080:localhost:8080 user@your-server-ip
```

Then open <http://localhost:8080> in your browser.

**Default login:** `admin` / value of `ADMIN_PASSWORD` from `.env`

### Update

```bash
cd /opt/isolate-panel
docker compose pull && docker compose up -d
```

See [QUICKSTART.md](QUICKSTART.md) for detailed step-by-step guide.

---

## SSH Tunnel Access

| Scenario | Command |
|---|---|
| Direct tunnel | `ssh -L 8080:localhost:8080 user@host` |
| Background tunnel | `ssh -fNL 8080:localhost:8080 user@host` |
| Via jump host | `ssh -J jump-host -L 8080:localhost:8080 user@host` |

The panel deliberately does **not** support direct internet access — this is a security feature.

---

## Manual Install (without Docker)

### Backend

```bash
# Prerequisites: Go 1.25+, gcc (CGO required for SQLite)
cd backend
make build          # binary → bin/server

# Run
CONFIG_PATH=configs/config.yaml ./bin/server
```

### Frontend

```bash
cd frontend
npm install
npm run build       # production build → dist/
```

Copy `dist/` to `/var/www/html` (or set `static_dir` in `config.yaml`).

### Proxy cores

Install core binaries and configure Supervisord to manage them.  
See [docs/CORES-MANUAL-INSTALL.md](docs/CORES-MANUAL-INSTALL.md) for detailed instructions.

---

## Development

```bash
# Full stack with hot reload (recommended)
cd docker
docker compose -f docker-compose.dev.yml up --build

# Or run separately:
cd backend && make run            # Go server on :8080 (no hot reload)
cd frontend && npm run dev        # Vite dev server on :5173 (proxies /api → :8080)
```

### Running tests

```bash
# Backend
cd backend
make test                         # all tests
go test -v -run TestFoo ./internal/api/...   # single test
go test ./... -coverprofile=coverage.out     # with coverage

# Frontend
cd frontend
npm test                          # Vitest unit tests
npm run test:e2e                  # Playwright e2e tests
```

### Linting

```bash
cd backend && make lint           # golangci-lint
cd frontend && npm run lint       # ESLint
cd frontend && npm run typecheck  # tsc --noEmit
```

---

## Project Structure

```
isolate-panel/
├── backend/
│   ├── cmd/server/           # Entry point (main.go)
│   ├── internal/
│   │   ├── api/              # Fiber HTTP handlers (one file per domain)
│   │   ├── services/         # Business logic layer
│   │   ├── models/           # GORM models
│   │   ├── middleware/       # Auth, rate limit, audit, security headers
│   │   ├── scheduler/        # Cron jobs (backup, traffic reset)
│   │   ├── app/              # DI wiring (providers, routes, background workers)
│   │   ├── auth/             # JWT token service
│   │   ├── cache/            # Ristretto cache manager
│   │   ├── cores/            # Supervisord XML-RPC core manager
│   │   ├── database/         # Migrations (000001–000029), seeds
│   │   └── config/           # Viper config loading
│   ├── tests/                # Integration, e2e, edge-case, leak tests
│   ├── benchmarks/           # Performance benchmarks
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── pages/            # 19 page components (lazy-loaded)
│   │   ├── components/       # UI components + features
│   │   ├── hooks/            # Custom hooks (useUsers, useCores, useWebSocket, …)
│   │   ├── stores/           # Zustand stores (auth, theme, toast)
│   │   └── api/              # Axios client + typed endpoint definitions
│   └── e2e/                  # Playwright tests
├── cli/                      # Cobra CLI tool (separate go.mod)
├── docker/                   # Dockerfile, docker-compose, supervisord.conf
└── docs/                     # Architecture, API, Contributing, Deployment
```

---

## API Documentation

Interactive API docs are available at `/api/docs` in development mode (disabled in production via `APP_ENV=production`).

Additional references:

- [docs/API.md](docs/API.md) — static endpoint reference
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — system architecture
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — deployment guide
- [docs/CLI.md](docs/CLI.md) — CLI reference

---

## Security

- Panel is only accessible via SSH tunnel — never exposed to the internet
- Subscriptions served on a dedicated public port (443) with auto-TLS
- Argon2id password hashing (industry standard for password storage)
- JWT access tokens (15 min) + refresh tokens (7 days)
- TOTP two-factor authentication
- Rate limiting: 60 req/min (standard), 10 req/min (heavy operations)
- Content Security Policy + security headers (X-Frame-Options, X-XSS-Protection, …)
- Auto-generated API keys for proxy cores (Sing-box, Mihomo)
- Swagger UI disabled in production
- Token format validation on subscription endpoints
- Audit log for all admin actions
- Automated data retention (expired tokens, old logs)

---

## License

MIT — see [LICENSE](LICENSE) for details.

## Contributing

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for development setup, code style, and PR process.
