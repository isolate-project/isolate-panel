# Isolate Panel

> Lightweight proxy core management panel for **Xray**, **Sing-box**, and **Mihomo** — designed for VPS with limited resources.

**[Читать на русском](README.ru.md)**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/isolate-project/isolate-panel/actions/workflows/test.yml/badge.svg)](https://github.com/isolate-project/isolate-panel/actions/workflows/test.yml)

---

## ✨ Key Features

| Category | Details |
|---|---|
| 🔄 **Multi-core** | Xray, Sing-box, Mihomo — start / stop / restart via Supervisord |
| 👥 **User management** | CRUD, traffic quotas, expiry dates, auto-enforcement, subscription links |
| 📡 **25+ protocols** | VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC v4/v5, Naive, AnyTLS, XHTTP, Snell, SSR, and more |
| 🔗 **Subscriptions** | Auto-detect, Clash, Sing-box, Isolate formats; QR codes; short URLs |
| 📊 **Real-time dashboard** | WebSocket-powered stats: active connections, traffic, core status — with polling fallback |
| 📈 **Traffic analytics** | 7-day traffic history, top-users chart, hourly/daily aggregation |
| 🔁 **Traffic reset** | Automatic weekly / monthly reset scheduler |
| 🔒 **Certificates** | ACME / Let's Encrypt auto-provisioning, manual upload, renewal, revocation |
| ☁️ **Cloudflare WARP** | WARP integration with route management and presets (gaming, streaming, etc.) |
| 🌍 **GeoIP / GeoSite** | Automatic database updates, country-based and category-based routing rules |
| 💾 **Encrypted backups** | Streaming AES-256-GCM, configurable retention, cron scheduling |
| 🔔 **Notifications** | Telegram bot + Webhook (HMAC-signed), configurable event triggers |
| 📝 **Audit log** | Immutable log of all admin actions (create, delete, start, stop, …) |
| 🛡️ **2FA / TOTP** | Time-based one-time password for admin login |
| 🖥️ **CLI** | Cobra-based CLI: `isolate-panel user list`, `core start xray`, `backup create` |
| 🔐 **Security** | JWT + Argon2id, CSP headers, rate limiting, request validation, SSH-tunnel-only access |

---

## 🧭 Protocol Matrix

| Protocol | Sing-box | Xray | Mihomo | Transport |
|---|:---:|:---:|:---:|---|
| HTTP | ✅ | ✅ | ✅ | — |
| SOCKS5 | ✅ | ✅ | ✅ | — |
| Mixed (HTTP+SOCKS5) | ✅ | — | ✅ | — |
| Shadowsocks | ✅ | ✅ | ✅ | WS, gRPC |
| VMess | ✅ | ✅ | ✅ | WS, gRPC, HTTP, HTTPUpgrade |
| VLESS | ✅ | ✅ | ✅ | WS, gRPC, HTTP, HTTPUpgrade |
| Trojan | ✅ | ✅ | ✅ | WS, gRPC |
| Hysteria2 | ✅ | ✅ | ✅ | QUIC |
| TUIC v4 | ✅ | — | ✅ | QUIC |
| TUIC v5 | ✅ | — | ✅ | QUIC |
| Naive | ✅ | — | — | — |
| AnyTLS | ✅ | — | — | — |
| XHTTP | — | ✅ | — | — |
| Redirect | ✅ | — | ✅ | — |
| Mieru | — | — | ✅ | — |
| Sudoku | — | — | ✅ | — |
| TrustTunnel | — | — | ✅ | — |
| ShadowsocksR | — | — | ✅ | — |
| Snell | — | — | ✅ | — |
| MASQUE (outbound) | — | — | ✅ | — |
| Tor (outbound) | ✅ | — | — | — |

> All protocols support **TLS** and **REALITY** where applicable.

---

## 🏗️ Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.26, Fiber v3, GORM, SQLite (WAL) |
| Frontend | Preact 10, TypeScript 5.9, Vite 6, Tailwind CSS 4, Zustand |
| Auth | JWT (access + refresh), Argon2id, TOTP (pquerna/otp) |
| Process mgmt | Supervisord (XML-RPC) |
| Deployment | Docker, Alpine Linux, multi-stage build |
| Logging | Zerolog (structured JSON) |

---

## 🚀 Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+

### One-liner install (recommended for VPS)

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh)
```

### Manual setup

```bash
mkdir -p /opt/isolate-panel && cd /opt/isolate-panel
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.yml -o docker-compose.yml
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/.env.example -o .env
nano .env   # set JWT_SECRET and ADMIN_PASSWORD
docker compose up -d
```

### Access via SSH tunnel

The panel **only** listens on `localhost:8080` — it never binds to a public interface. Open a tunnel from your local machine:

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

---

## 🔐 SSH Tunnel Access

The panel deliberately **does not support** direct internet access — this is a security feature, not a limitation.

| Scenario | Command |
|---|---|
| Direct tunnel | `ssh -L 8080:localhost:8080 user@host` |
| Background tunnel | `ssh -fNL 8080:localhost:8080 user@host` |
| Via jump host | `ssh -J jump-host -L 8080:localhost:8080 user@host` |

---

## 🛠️ Development

### Full stack with hot reload (recommended)

```bash
cd docker
docker compose -f docker-compose.dev.yml up --build
```

### Or run separately

```bash
# Backend (Go server on :8080)
cd backend && make run

# Frontend (Vite dev server on :5173, proxies /api → :8080)
cd frontend && npm run dev
```

### Running tests

```bash
# Backend
cd backend
make test                                                      # all tests
go test -v -run TestFoo ./internal/api/...                     # single test
go test ./... -coverprofile=coverage.out                       # with coverage

# Frontend
cd frontend
npm test                                                       # Vitest unit tests
npm run test:e2e                                               # Playwright e2e tests
```

### Linting

```bash
cd backend && make lint           # golangci-lint
cd frontend && npm run lint       # ESLint
cd frontend && npm run typecheck  # tsc --noEmit
```

---

## 📁 Project Structure

```
isolate-panel/
├── backend/
│   ├── cmd/
│   │   ├── server/main.go           # Application entry point
│   │   └── migrate/main.go          # Database migration tool
│   ├── internal/
│   │   ├── api/                     # Fiber HTTP handlers (one file per domain)
│   │   ├── services/                # Business logic layer
│   │   ├── models/                  # GORM models (16 domain models)
│   │   ├── middleware/              # Auth, rate limit, audit, security headers
│   │   ├── scheduler/              # Cron jobs (backup, traffic reset)
│   │   ├── app/                     # DI wiring, routes, background workers
│   │   ├── auth/                    # JWT + Argon2id + TOTP
│   │   ├── cache/                   # Ristretto cache manager
│   │   ├── cores/                   # Core adapters (xray/, singbox/, mihomo/)
│   │   ├── protocol/               # Protocol schema registry (25+ protocols)
│   │   ├── database/               # Migrations (39 steps), seeds
│   │   └── config/                 # Viper config loading
│   ├── tests/                       # Integration, e2e, edge-case, leak tests
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── pages/                   # 19 page components (lazy-loaded)
│   │   ├── components/             # UI primitives + feature components
│   │   ├── hooks/                  # Custom hooks (useUsers, useCores, useWebSocket, …)
│   │   ├── stores/                 # Zustand stores (auth, theme, toast)
│   │   └── api/                    # Axios client + typed endpoints
│   └── e2e/                        # Playwright tests
├── cli/                            # Cobra CLI tool (separate go.mod)
├── docker/                         # Dockerfile, Compose, Supervisord configs
└── docs/                           # Architecture, API reference, guides
```

---

## 🤝 Contributing

We welcome contributions! Please read the [Contributing Guide](docs/CONTRIBUTING.md) for:

- **Development setup** — how to run the project locally
- **Code style** — Go formatting, TypeScript/ESLint rules, commit conventions
- **Pull Request process** — branch naming, PR template, review checklist
- **Reporting issues** — bug reports, feature requests

### Quick rules

1. **Fork** the repository and create a feature branch from `develop`
2. **Write tests** for any new functionality
3. **Run linters** before committing: `make lint` (backend), `npm run lint` (frontend)
4. **One feature per PR** — keep changes focused
5. **Conventional commits** — `feat:`, `fix:`, `docs:`, `chore:`

---

## 📚 Documentation

| Document | Description |
|---|---|
| [Master Plan](docs/MASTER_PLAN.md) | Project roadmap, architecture vision, development phases |
| [User Manual](docs/USER_MANUAL.md) | Complete guide for system administrators (RU) |
| [Contributing](docs/CONTRIBUTING.md) | Developer setup, code style, PR process |
| [Architecture](docs/ARCHITECTURE.md) | System architecture deep-dive |
| [API Reference](docs/API.md) | REST API endpoint documentation |
| [CLI Reference](docs/CLI.md) | Cobra CLI command reference |

---

## 🛡️ Security

- Panel accessible **only via SSH tunnel** — never exposed to the internet
- Subscription endpoints served on a dedicated public port with TLS
- **Argon2id** password hashing (industry standard)
- **JWT** access tokens (15 min) + refresh tokens (7 days, hashed in DB)
- **TOTP** two-factor authentication
- Rate limiting: 5/min (login), 60/min (standard), 10/min (heavy operations)
- Content Security Policy + security headers
- Auto-generated API keys for proxy cores
- Swagger UI **disabled** in production
- Immutable audit log for all admin actions
- Automated data retention (expired tokens, old logs)

---

## ⚖️ License

[MIT](LICENSE) © 2026 isolate-project
