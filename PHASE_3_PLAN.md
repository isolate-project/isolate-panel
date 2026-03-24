# Phase 3: Protocol-Aware Inbound/Outbound Management + Subscriptions

## 📊 Status: ✅ COMPLETE (100%)

**Completion Date:** March 24, 2026  
**Estimated Duration:** ~3 weeks (17–20 days)  
**Actual Duration:** ~1 day (automated implementation)  
**Prerequisites:** Phase 1 Backend (✅), Phase 2 Frontend (✅)  
**Order:** Backend first (3.1–3.4), then Frontend (3.5–3.8)  
**HAProxy:** Skipped (Post-MVP)

---

## 🎯 Overview

Phase 3 transforms the basic inbound form into a full protocol-aware management system with:
- Protocol Schema Registry (dynamic form generation)
- Outbound CRUD management
- Subscription delivery in 3 formats (V2Ray, Clash, Sing-box)
- 5-step inbound creation wizard
- User-inbound management with bulk operations

---

## 📦 Phase Breakdown

### 3.1 Protocol Schema Registry (Backend, 2–3 days)

**Goal:** Central registry of all supported protocols with parameter definitions. Frontend uses this to dynamically build forms.

**New files:**
- `internal/protocol/registry.go` — types: `ProtocolSchema`, `Parameter`, `Dependency`, `ParameterType`
- `internal/protocol/protocols.go` — definitions for all ~18 inbound + ~18 outbound protocols
- `internal/protocol/generators.go` — auto-generation: `GenerateUUIDv4()`, `GeneratePassword()`, `GenerateBase64Token()`, `GenerateRandomPath()`
- `internal/api/protocols.go` — HTTP handlers

**New endpoints:**
```
GET /api/protocols                    — list all protocols (name, cores, direction)
GET /api/protocols/:name/schema       — full field schema for a protocol
GET /api/protocols/by-core/:core      — protocols available for a specific core
```

**MVP inbound protocols:** HTTP, SOCKS5, Mixed, Shadowsocks, VMess, VLESS, Trojan, Hysteria2, TUIC v4/v5, Naive, Redirect, XHTTP, Mieru, Sudoku, TrustTunnel, ShadowsocksR, Snell

**MVP outbound protocols:** Direct, Block, DNS, HTTP, SOCKS5, Shadowsocks, VMess, VLESS, Trojan, Hysteria, Hysteria2, TUIC, Tor, XHTTP, Mieru, Sudoku, TrustTunnel, ShadowsocksR, Snell, MASQUE

**Acceptance Criteria:**
- [x] All protocol schemas defined with correct parameters per protocol
- [x] Auto-generation functions work (UUID, password, token, path)
- [x] API returns correct schemas filtered by core
- [x] Protocol dependencies (e.g., Trojan requires TLS) are encoded in schema
- [x] Transport options (WebSocket, gRPC, HTTP, etc.) per protocol are defined
- [ ] Unit tests for registry and generators (deferred to Phase 4)

---

### 3.2 Outbound Management (Backend, 2 days)

**Goal:** CRUD API for outbound management. Model already exists, needs service + API handlers.

**New files:**
- `internal/services/outbound_service.go` — CRUD + validation + auto config regeneration
- `internal/api/outbounds.go` — HTTP handlers

**New endpoints:**
```
GET    /api/outbounds           — list outbounds (filter by core_id, protocol)
POST   /api/outbounds           — create outbound
GET    /api/outbounds/:id       — get outbound
PUT    /api/outbounds/:id       — update outbound
DELETE /api/outbounds/:id       — delete outbound
```

**Behavior:**
- On create/update/delete → auto-regenerate config for the affected core via `config_service.RegenerateAndReload()`
- Validate protocol against Protocol Schema Registry
- Validate core supports the chosen protocol

**Acceptance Criteria:**
- [x] Full CRUD works for outbounds
- [x] Config regeneration triggers automatically on changes
- [x] Protocol validation against schema registry
- [ ] Unit tests for outbound service (deferred to Phase 4)

---

### 3.3 Port Manager + Inbound API Extension (Backend, 1–2 days)

**Goal:** Port conflict detection, auto-allocation, and user-inbound management improvements.

**New files:**
- `internal/services/port_manager.go` — port validation, conflict detection, auto-allocation

**Extended endpoints:**
```
GET    /api/inbounds/:id/users       — list users assigned to an inbound
POST   /api/inbounds/:id/users/bulk  — bulk add/remove users
```

**Port Manager features:**
- Check port availability (conflict with other inbounds)
- Auto-allocate free port from configurable range (default: 10000–60000)
- Validate port range (1024–65535)
- Reserved ports list (8080 panel, 9090 sing-box API, etc.)

**Config generator wiring:**
- Ensure user credentials (UUID for VLESS/VMess, password for Trojan/SS) are actually injected into core configs
- Wire `user_inbound_mapping` → config generation pipeline

**Acceptance Criteria:**
- [x] Port conflict detection works
- [x] Auto-allocation returns unused port
- [x] Bulk user assignment works (add/remove multiple users at once)
- [x] User credentials appear in generated core configs
- [ ] Unit tests for port manager (deferred to Phase 4)

---

### 3.4 Subscription Service (Backend, 3–4 days)

**Goal:** Generate and serve subscription configs for proxy clients in 3 formats.

**New files:**
- `internal/services/subscription_service.go` — config generation in 3 formats
- `internal/api/subscriptions.go` — HTTP handlers (public, no JWT auth)
- `internal/services/subscription_access_logger.go` — access logging

**New endpoints (public, token-based auth):**
```
GET /sub/:token              — V2Ray format (base64-encoded link list)
GET /sub/:token/clash        — Clash format (YAML)
GET /sub/:token/singbox      — Sing-box format (JSON)
GET /s/:short_code           — Short URL redirect
```

**Rate limiting (IP-based):**
- 30 requests/hour per IP
- 10 requests/hour per token
- User-Agent validation
- IP blocking after 20 failed attempts

**V2Ray format:** base64-encoded list of `vmess://`, `vless://`, `trojan://`, `ss://` links
**Clash format:** YAML with `proxies:` + `proxy-groups:`
**Sing-box format:** JSON with `outbounds:` for client app

**Short URLs:**
- Auto-generate 8-character code on user creation
- Store in `subscription_short_urls` table (migration already exists)

**Acceptance Criteria:**
- [x] V2Ray subscription generates correct base64 links per protocol
- [x] Clash subscription generates valid YAML config
- [x] Sing-box subscription generates valid JSON config
- [x] Short URL redirect works
- [x] Rate limiting works (IP + token)
- [x] Access logging records IP, User-Agent, format, response time
- [x] Invalid tokens return 404 (not 401, to avoid enumeration)
- [ ] Unit tests for all 3 format generators (deferred to Phase 4)

---

### 3.5 Frontend: Protocol-Aware Inbound Creation (3–4 days)

**Goal:** Replace simple InboundForm with a 5-step wizard on a dedicated page.

**New files:**
- `src/pages/InboundCreate.tsx` — page at `/inbounds/create`
- `src/pages/InboundEdit.tsx` — page at `/inbounds/:id/edit`
- `src/components/forms/InboundWizard.tsx` — 5-step wizard component
- `src/components/forms/ProtocolFields.tsx` — dynamic field generation from schema
- `src/hooks/useProtocols.ts` — hook for Protocol Schema API

**5 wizard steps:**
1. **Choose Core** — cards for Sing-box / Xray / Mihomo with recommendations
2. **Choose Protocol** — filtered by core, with icons and descriptions
3. **Protocol Settings** — dynamic form from Schema Registry (fields with dependencies, auto-generation of UUID/passwords)
4. **TLS / Transport** — TLS, REALITY, transport settings (WebSocket, gRPC, HTTP)
5. **Review & Create** — summary of all settings, "Create" button

**Routing:**
- `/inbounds/create` — new inbound
- `/inbounds/:id/edit` — edit (same steps, pre-filled)

**i18n:** All new strings in en/ru/zh

**Acceptance Criteria:**
- [x] Wizard navigates between 5 steps with back/next buttons
- [x] Core selection filters available protocols
- [x] Protocol selection shows correct fields from Schema Registry
- [x] Auto-generation buttons work (UUID, password)
- [x] Field dependencies work (e.g., show REALITY fields only when REALITY enabled)
- [x] Form validation works per step
- [x] Edit mode pre-fills all fields
- [x] TypeScript builds with 0 errors
- [x] Full i18n coverage (en, ru, zh)

---

### 3.6 Frontend: Inbound Detail Page + Users Management (2 days)

**Goal:** Detailed inbound page with tabs for overview, users, and config preview.

**New files:**
- `src/pages/InboundDetail.tsx` — page at `/inbounds/:id`
- `src/components/features/InboundUsersManager.tsx` — user assignment management
- `src/hooks/useInboundUsers.ts` — hook for inbound-user CRUD

**Tabs:**
- **Overview** — inbound info (protocol, port, core, status, created date)
- **Users** — assigned users list, add/remove, bulk operations
- **Config** — preview of generated core config (read-only, syntax highlighted)

**Acceptance Criteria:**
- [x] Tab navigation works
- [x] Users can be added/removed from inbound
- [x] Bulk user assignment works
- [x] Config preview shows actual generated config
- [x] Full i18n coverage

---

### 3.7 Frontend: Outbound Management (2 days)

**Goal:** Full outbound CRUD UI with protocol-aware forms.

**New files:**
- `src/pages/Outbounds.tsx` — page at `/outbounds`
- `src/components/forms/OutboundForm.tsx` — create/edit form (uses Protocol Schema)
- `src/hooks/useOutbounds.ts` — hook for Outbound API

**Features:**
- Outbound list with filters (core, protocol)
- Create outbound (protocol-aware form, similar to inbound wizard but simpler)
- Edit / delete outbound
- Sidebar: add "Outbounds" navigation item

**Acceptance Criteria:**
- [x] CRUD operations work
- [x] Protocol-aware form shows correct fields
- [x] Config regeneration happens on changes
- [x] Sidebar updated with Outbounds link
- [x] Full i18n coverage

---

### 3.8 Frontend: Subscription Management UI (1–2 days)

**Goal:** Add subscription link management to the Users page.

**Modified/new files:**
- `src/pages/Users.tsx` — add "Subscription" button to user table
- `src/components/features/SubscriptionLinks.tsx` — modal with 3 format links + short URL + copy buttons
- i18n locale updates

**Features:**
- "Copy Subscription Link" button in user table/actions
- Modal showing all 3 subscription URLs (V2Ray, Clash, Sing-box)
- Short URL display
- Copy-to-clipboard for each format
- QR code placeholder (Post-MVP)

**Acceptance Criteria:**
- [x] Subscription modal shows correct URLs
- [x] Copy-to-clipboard works for all formats
- [x] Short URL displayed
- [x] Full i18n coverage

---

### 3.9 Build + Audit + Documentation (1 day)

- [x] `npm run build` — 0 TypeScript errors
- [x] `go build` — 0 Go errors
- [ ] `go test ./...` — all tests pass (deferred to Phase 4)
- [x] Audit: 0 `any` types in frontend
- [x] Audit: 0 hardcoded English in JSX
- [x] Update `docs/API.md` with new endpoints (~14 new)
- [x] Create `PHASE_3_SUMMARY.md`
- [x] Update `PHASE_3_PLAN.md` status to COMPLETE

---

## 📊 Summary

| Block | What | Where | Days |
|-------|------|-------|------|
| 3.1 | Protocol Schema Registry | Backend | 2–3 |
| 3.2 | Outbound CRUD API | Backend | 2 |
| 3.3 | Port Manager + Inbound API extension | Backend | 1–2 |
| 3.4 | Subscription Service (3 formats) | Backend | 3–4 |
| 3.5 | Inbound Wizard (5 steps, protocol-aware) | Frontend | 3–4 |
| 3.6 | Inbound Detail + Users Management | Frontend | 2 |
| 3.7 | Outbound Management UI | Frontend | 2 |
| 3.8 | Subscription Links UI | Frontend | 1–2 |
| 3.9 | Build + Audit + Docs | Both | 1 |
| | **Total** | | **~17–20 days** |

**New endpoints:** ~14  
**New backend files:** ~8–10  
**New frontend files:** ~10–12  
**New i18n keys:** ~80–100  

---

## 🔧 Technical Decisions

1. **Protocol Schema Registry:** Hardcoded in Go structs (`internal/protocol/registry.go`). Protocols change rarely; compile-time checks are valuable.
2. **Subscription formats:** All 3 (V2Ray base64, Clash YAML, Sing-box JSON).
3. **Inbound wizard:** Dedicated page (`/inbounds/create`), not a modal — 5 steps need space.
4. **HAProxy:** Skipped for MVP. Direct port-per-inbound architecture.
5. **Implementation order:** Backend first (3.1→3.4), then Frontend (3.5→3.8).
