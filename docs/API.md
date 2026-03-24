# API Documentation

## Overview

Isolate Panel API v0.1.0 - RESTful API for managing proxy cores (Xray, Sing-box, Mihomo).

**Base URL:** `http://localhost:8080/api`

**Authentication:** JWT Bearer token (except auth endpoints)

---

## Authentication

### POST /api/auth/login

Login with admin credentials.

**Request:**
```json
{
  "username": "admin",
  "password": "admin"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": "2024-03-24T15:30:00Z",
  "admin": {
    "id": 1,
    "username": "admin",
    "is_super_admin": true
  }
}
```

**Rate Limit:** 5 requests per minute per IP

---

### POST /api/auth/refresh

Refresh access token using refresh token.

**Request:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": "2024-03-24T15:45:00Z"
}
```

---

### POST /api/auth/logout

Revoke refresh token.

**Request:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response (200):**
```json
{
  "message": "Logged out successfully"
}
```

---

### GET /api/me

Get current admin information.

**Headers:** `Authorization: Bearer <access_token>`

**Response (200):**
```json
{
  "id": 1,
  "username": "admin",
  "is_super_admin": true,
  "created_at": "2024-03-24T10:00:00Z"
}
```

---

## Core Management

### GET /api/cores

List all proxy cores.

**Headers:** `Authorization: Bearer <access_token>`

**Response (200):**
```json
[
  {
    "id": 1,
    "name": "singbox",
    "type": "sing-box",
    "version": "1.13.3",
    "is_enabled": true,
    "is_running": false,
    "pid": null,
    "uptime_seconds": 0,
    "restart_count": 0,
    "last_error": ""
  },
  {
    "id": 2,
    "name": "xray",
    "type": "xray",
    "version": "26.2.6",
    "is_enabled": true,
    "is_running": false,
    "pid": null,
    "uptime_seconds": 0,
    "restart_count": 0,
    "last_error": ""
  },
  {
    "id": 3,
    "name": "mihomo",
    "type": "mihomo",
    "version": "1.19.21",
    "is_enabled": true,
    "is_running": false,
    "pid": null,
    "uptime_seconds": 0,
    "restart_count": 0,
    "last_error": ""
  }
]
```

---

### GET /api/cores/:name

Get specific core information.

**Parameters:**
- `name` (path): Core name (singbox, xray, mihomo)

**Response (200):**
```json
{
  "id": 1,
  "name": "singbox",
  "type": "sing-box",
  "version": "1.13.3",
  "is_enabled": true,
  "is_running": true,
  "pid": 12345,
  "uptime_seconds": 3600,
  "restart_count": 2,
  "last_error": ""
}
```

---

### POST /api/cores/:name/start

Start a core.

**Parameters:**
- `name` (path): Core name

**Response (200):**
```json
{
  "message": "Core started successfully",
  "core": "singbox"
}
```

---

### POST /api/cores/:name/stop

Stop a core.

**Response (200):**
```json
{
  "message": "Core stopped successfully",
  "core": "singbox"
}
```

---

### POST /api/cores/:name/restart

Restart a core.

**Response (200):**
```json
{
  "message": "Core restarted successfully",
  "core": "singbox"
}
```

---

### GET /api/cores/:name/status

Get core status.

**Response (200):**
```json
{
  "name": "singbox",
  "is_running": true,
  "is_enabled": true,
  "pid": 12345,
  "uptime": 3600,
  "restarts": 2,
  "last_error": ""
}
```

---

## User Management

### GET /api/users

List all users with pagination.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20)

**Response (200):**
```json
{
  "users": [
    {
      "id": 1,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "username": "user1",
      "email": "user1@example.com",
      "token": "abc123def456",
      "subscription_token": "sub_token_123",
      "is_active": true,
      "traffic_limit_bytes": 107374182400,
      "traffic_used_bytes": 1073741824,
      "expire_at": "2024-12-31T23:59:59Z",
      "created_at": "2024-03-24T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

---

### POST /api/users

Create a new user.

**Request:**
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "traffic_limit_bytes": 107374182400,
  "expire_at": "2024-12-31T23:59:59Z",
  "is_active": true
}
```

**Response (201):**
```json
{
  "id": 2,
  "uuid": "generated-uuid",
  "username": "newuser",
  "email": "newuser@example.com",
  "token": "generated-token",
  "subscription_token": "generated-sub-token",
  "is_active": true,
  "traffic_limit_bytes": 107374182400,
  "traffic_used_bytes": 0,
  "expire_at": "2024-12-31T23:59:59Z",
  "created_at": "2024-03-24T11:00:00Z"
}
```

---

### GET /api/users/:id

Get user details.

**Response (200):**
```json
{
  "id": 1,
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "username": "user1",
  "email": "user1@example.com",
  "token": "abc123def456",
  "subscription_token": "sub_token_123",
  "is_active": true,
  "traffic_limit_bytes": 107374182400,
  "traffic_used_bytes": 1073741824,
  "expire_at": "2024-12-31T23:59:59Z",
  "created_at": "2024-03-24T10:00:00Z",
  "updated_at": "2024-03-24T10:00:00Z"
}
```

---

### PUT /api/users/:id

Update user.

**Request:**
```json
{
  "username": "updated_user",
  "email": "updated@example.com",
  "is_active": false,
  "traffic_limit_bytes": 53687091200
}
```

**Response (200):**
```json
{
  "id": 1,
  "username": "updated_user",
  "email": "updated@example.com",
  "is_active": false,
  "traffic_limit_bytes": 53687091200,
  "updated_at": "2024-03-24T12:00:00Z"
}
```

---

### DELETE /api/users/:id

Delete user.

**Response (200):**
```json
{
  "message": "User deleted successfully"
}
```

---

### POST /api/users/:id/regenerate

Regenerate user credentials (UUID, Token, Subscription Token).

**Response (200):**
```json
{
  "id": 1,
  "uuid": "new-generated-uuid",
  "token": "new-generated-token",
  "subscription_token": "new-generated-sub-token",
  "message": "Credentials regenerated successfully"
}
```

---

### GET /api/users/:id/inbounds

Get inbounds assigned to user.

**Response (200):**
```json
[
  {
    "id": 1,
    "name": "VLESS Inbound",
    "protocol": "vless",
    "port": 443,
    "is_enabled": true
  }
]
```

---

## Inbound Management

### GET /api/inbounds

List all inbounds.

**Query Parameters:**
- `core_id` (optional): Filter by core ID
- `is_enabled` (optional): Filter by enabled status (true/false)

**Response (200):**
```json
[
  {
    "id": 1,
    "name": "VLESS Inbound",
    "protocol": "vless",
    "core_id": 1,
    "listen_address": "0.0.0.0",
    "port": 443,
    "config_json": "{}",
    "tls_enabled": true,
    "tls_cert_id": null,
    "reality_enabled": false,
    "is_enabled": true,
    "created_at": "2024-03-24T10:00:00Z",
    "core": {
      "id": 1,
      "name": "singbox",
      "type": "sing-box"
    }
  }
]
```

---

### POST /api/inbounds

Create a new inbound.

**Request:**
```json
{
  "name": "VLESS Inbound",
  "protocol": "vless",
  "core_id": 1,
  "listen_address": "0.0.0.0",
  "port": 443,
  "config_json": "{}",
  "tls_enabled": true,
  "is_enabled": true
}
```

**Response (201):**
```json
{
  "id": 1,
  "name": "VLESS Inbound",
  "protocol": "vless",
  "core_id": 1,
  "port": 443,
  "is_enabled": true,
  "created_at": "2024-03-24T10:00:00Z"
}
```

**Note:** Creating an inbound automatically starts the associated core if not running.

---

### GET /api/inbounds/:id

Get inbound details.

**Response (200):**
```json
{
  "id": 1,
  "name": "VLESS Inbound",
  "protocol": "vless",
  "core_id": 1,
  "listen_address": "0.0.0.0",
  "port": 443,
  "config_json": "{}",
  "tls_enabled": true,
  "is_enabled": true,
  "created_at": "2024-03-24T10:00:00Z",
  "updated_at": "2024-03-24T10:00:00Z"
}
```

---

### PUT /api/inbounds/:id

Update inbound.

**Request:**
```json
{
  "name": "Updated VLESS",
  "port": 8443,
  "is_enabled": false
}
```

**Response (200):**
```json
{
  "id": 1,
  "name": "Updated VLESS",
  "port": 8443,
  "is_enabled": false,
  "updated_at": "2024-03-24T12:00:00Z"
}
```

**Note:** Updating an inbound triggers config regeneration and core reload.

---

### DELETE /api/inbounds/:id

Delete inbound.

**Response (200):**
```json
{
  "message": "Inbound deleted successfully"
}
```

**Note:** Deleting the last inbound automatically stops the associated core.

---

### GET /api/inbounds/core/:core_id

Get all inbounds for a specific core.

**Response (200):**
```json
[
  {
    "id": 1,
    "name": "VLESS Inbound",
    "protocol": "vless",
    "port": 443,
    "is_enabled": true
  }
]
```

---

### POST /api/inbounds/assign

Assign inbound to user.

**Request:**
```json
{
  "user_id": 1,
  "inbound_id": 1
}
```

**Response (200):**
```json
{
  "message": "Inbound assigned to user successfully"
}
```

---

### POST /api/inbounds/unassign

Unassign inbound from user.

**Request:**
```json
{
  "user_id": 1,
  "inbound_id": 1
}
```

**Response (200):**
```json
{
  "message": "Inbound unassigned from user successfully"
}
```

---

## Protocols (Phase 3)

Protocol Schema Registry - dynamic protocol-aware forms.

### GET /api/protocols

List all available protocols.

**Query Parameters:**
- `core` (optional): Filter by core name (singbox, xray, mihomo)
- `direction` (optional): Filter by direction (inbound, outbound)

**Response (200):**
```json
{
  "protocols": [
    {
      "protocol": "vless",
      "label": "VLESS",
      "description": "Next-generation V2Ray protocol",
      "core": ["xray"],
      "direction": "inbound",
      "requires_tls": true,
      "category": "shadow"
    }
  ],
  "total": 25
}
```

---

### GET /api/protocols/:name

Get full schema for a specific protocol.

**Response (200):**
```json
{
  "protocol": "vless",
  "label": "VLESS",
  "description": "Next-generation V2Ray protocol",
  "core": ["xray"],
  "direction": "inbound",
  "requires_tls": true,
  "parameters": {
    "uuid": {
      "name": "uuid",
      "label": "UUID",
      "type": "uuid",
      "required": true,
      "auto_generate": true,
      "auto_gen_func": "GenerateUUIDv4"
    }
  },
  "transport": ["tcp", "ws", "grpc"],
  "category": "shadow"
}
```

---

### GET /api/protocols/:name/defaults

Get auto-generated default values for a protocol.

**Response (200):**
```json
{
  "protocol": "vless",
  "defaults": {
    "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "flow": "",
    "encryption": "none"
  }
}
```

---

## Outbounds (Phase 3)

CRUD operations for outbound connections.

### GET /api/outbounds

List all outbounds.

**Query Parameters:**
- `core_id` (optional): Filter by core ID
- `protocol` (optional): Filter by protocol

**Response (200):**
```json
[
  {
    "id": 1,
    "name": "Direct Outbound",
    "protocol": "direct",
    "core_id": 1,
    "config_json": "{}",
    "priority": 1,
    "is_enabled": true,
    "created_at": "2024-03-24T10:00:00Z",
    "updated_at": "2024-03-24T10:00:00Z",
    "core": {
      "id": 1,
      "name": "singbox",
      "type": "singbox"
    }
  }
]
```

---

### POST /api/outbounds

Create a new outbound.

**Request:**
```json
{
  "name": "Direct Outbound",
  "protocol": "direct",
  "core_id": 1,
  "config_json": "{}",
  "priority": 1,
  "is_enabled": true
}
```

**Response (201):**
```json
{
  "id": 1,
  "name": "Direct Outbound",
  "protocol": "direct",
  "core_id": 1,
  "config_json": "{}",
  "priority": 1,
  "is_enabled": true,
  "created_at": "2024-03-24T10:00:00Z",
  "updated_at": "2024-03-24T10:00:00Z"
}
```

**Behavior:** Auto-regenerates core config on success.

---

### GET /api/outbounds/:id

Get a specific outbound by ID.

**Response (200):**
```json
{
  "id": 1,
  "name": "Direct Outbound",
  "protocol": "direct",
  "core_id": 1,
  "config_json": "{}",
  "priority": 1,
  "is_enabled": true,
  "created_at": "2024-03-24T10:00:00Z",
  "updated_at": "2024-03-24T10:00:00Z"
}
```

---

### PUT /api/outbounds/:id

Update an existing outbound.

**Request:**
```json
{
  "name": "Direct Outbound Updated",
  "priority": 2,
  "is_enabled": false
}
```

**Response (200):**
```json
{
  "id": 1,
  "name": "Direct Outbound Updated",
  "protocol": "direct",
  "core_id": 1,
  "config_json": "{}",
  "priority": 2,
  "is_enabled": false,
  "updated_at": "2024-03-24T11:00:00Z"
}
```

**Behavior:** Auto-regenerates core config on success.

---

### DELETE /api/outbounds/:id

Delete an outbound.

**Response (200):**
```json
{
  "message": "Outbound deleted successfully"
}
```

**Behavior:** Auto-regenerates core config on success.

---

## Inbound User Management (Phase 3)

Bulk user assignment operations.

### GET /api/inbounds/:id/users

List users assigned to a specific inbound.

**Response (200):**
```json
{
  "users": [
    {
      "id": 1,
      "username": "user1",
      "uuid": "a1b2c3d4-...",
      "is_active": true
    }
  ],
  "total": 1
}
```

---

### POST /api/inbounds/:id/users/bulk

Bulk add/remove users from an inbound.

**Request:**
```json
{
  "add_user_ids": [2, 3],
  "remove_user_ids": [1]
}
```

**Response (200):**
```json
{
  "message": "Users updated successfully",
  "added": 2,
  "removed": 1
}
```

---

## Subscriptions (Phase 3)

Public subscription endpoints (no JWT auth, token-based).

### GET /sub/:token

V2Ray subscription format (base64-encoded links).

**Authentication:** Token from user's `subscription_token` field

**Response (200):**
```
Content-Type: text/plain
vmess://ey...
vless://abc...
trojan://xyz...
```

**Rate Limit:** 10 requests/hour per token, 30 requests/hour per IP

---

### GET /sub/:token/clash

Clash subscription format (YAML).

**Response (200):**
```yaml
Content-Type: text/yaml
proxies:
  - name: "VLESS-443"
    type: vless
    server: example.com
    port: 443
    uuid: "..."
proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - "VLESS-443"
```

---

### GET /sub/:token/singbox

Sing-box subscription format (JSON).

**Response (200):**
```json
Content-Type: application/json
{
  "outbounds": [
    {
      "type": "vless",
      "tag": "VLESS-443",
      "server": "example.com",
      "server_port": 443,
      "uuid": "..."
    }
  ]
}
```

---

### GET /s/:short_code

Short URL redirect to full subscription URL.

**Response (302):** Redirects to `/sub/:token`

---

### GET /api/subscriptions/:user_id/short-url

(Admin, requires JWT) Generate or retrieve short URL for a user.

**Query Parameters:**
- `token` (required): User's subscription token

**Response (200):**
```json
{
  "short_url": "http://localhost:8080/s/abc12345",
  "short_code": "abc12345"
}
```

---

## Error Responses

All endpoints may return the following error responses:

### 400 Bad Request
```json
{
  "error": "Invalid request body"
}
```

### 401 Unauthorized
```json
{
  "error": "Missing authorization header"
}
```

### 404 Not Found
```json
{
  "error": "Resource not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "Internal server error"
}
```

---

## Features

### Lazy Loading
Cores are not started automatically at system startup. They start only when:
- First inbound is created for that core
- Manual start command is issued

Cores stop automatically when:
- Last inbound is deleted
- Manual stop command is issued

This saves 80-100MB RAM when cores are not in use.

### Config Regeneration
Configuration files are automatically regenerated when:
- Inbound is created/updated/deleted
- User is assigned/unassigned to inbound

### Rate Limiting
- Login endpoint: 5 attempts per minute per IP
- Other endpoints: No rate limiting (protected by authentication)

---

## Total Endpoints: 46

- **Auth:** 4 endpoints
- **Admin:** 1 endpoint
- **Cores:** 6 endpoints
- **Users:** 7 endpoints
- **Inbounds:** 8 endpoints
- **Protocols:** 3 endpoints (Phase 3)
- **Outbounds:** 5 endpoints (Phase 3)
- **Inbound Users:** 2 endpoints (Phase 3)
- **Subscriptions:** 5 endpoints (Phase 3)
- **Health:** 1 endpoint
- **API Info:** 1 endpoint
- **Docs:** 1 endpoint (this page)
