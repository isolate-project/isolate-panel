# Isolate Panel API Reference

Complete API documentation for Isolate Panel v0.1.0

---

## Base URL

```
Production: http://localhost:8080/api
Development: http://localhost:8080/api
```

---

## Authentication

Most API endpoints require authentication using JWT Bearer tokens.

### Login

**POST** `/api/auth/login`

Authenticate and receive access/refresh tokens.

**Request:**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

**Response (401 Unauthorized):**
```json
{
  "success": false,
  "error": "Invalid credentials"
}
```

---

### Refresh Token

**POST** `/api/auth/refresh`

Refresh an expired access token.

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

---

### Logout

**POST** `/api/auth/logout`

Invalidate the current session.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

---

### Get Current User

**GET** `/api/me`

Get information about the currently authenticated admin.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "is_super_admin": true,
    "created_at": "2026-03-25T10:00:00Z",
    "last_login_at": "2026-03-25T12:00:00Z"
  }
}
```

---

## Users

### List Users

**GET** `/api/users`

Get a list of all users with pagination.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20)
- `search` (optional): Search by username or email

**Response (200 OK):**
```json
{
  "success": true,
  "users": [
    {
      "id": 1,
      "username": "user1",
      "email": "user1@example.com",
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "is_active": true,
      "traffic_limit_bytes": 107374182400,
      "traffic_used_bytes": 10737418240,
      "expiry_date": "2026-12-31T23:59:59Z",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 20
}
```

---

### Create User

**POST** `/api/users`

Create a new user with auto-generated credentials.

**Headers:**
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request:**
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "securepassword123",
  "traffic_limit_bytes": 107374182400,
  "expiry_days": 30,
  "inbound_ids": [1, 2, 3]
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "user": {
    "id": 2,
    "username": "newuser",
    "email": "newuser@example.com",
    "uuid": "550e8400-e29b-41d4-a716-446655440001",
    "password": "securepassword123",
    "subscription_token": "sub_token_xyz123",
    "is_active": true,
    "traffic_limit_bytes": 107374182400,
    "traffic_used_bytes": 0,
    "expiry_date": "2026-04-24T23:59:59Z",
    "created_at": "2026-03-25T12:00:00Z",
    "inbound_ids": [1, 2, 3]
  }
}
```

---

### Get User

**GET** `/api/users/:id`

Get details of a specific user.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "user1",
    "email": "user1@example.com",
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "subscription_token": "sub_token_abc123",
    "is_active": true,
    "is_online": false,
    "traffic_limit_bytes": 107374182400,
    "traffic_used_bytes": 10737418240,
    "expiry_date": "2026-12-31T23:59:59Z",
    "last_connected_at": "2026-03-24T18:00:00Z",
    "created_at": "2026-01-01T00:00:00Z",
    "inbound_ids": [1, 2]
  }
}
```

---

### Update User

**PUT** `/api/users/:id`

Update user information.

**Headers:**
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request:**
```json
{
  "username": "updateduser",
  "email": "updated@example.com",
  "traffic_limit_bytes": 214748364800,
  "is_active": true,
  "inbound_ids": [1, 3]
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "updateduser",
    "email": "updated@example.com",
    "is_active": true,
    "traffic_limit_bytes": 214748364800,
    "updated_at": "2026-03-25T12:30:00Z"
  }
}
```

---

### Delete User

**DELETE** `/api/users/:id`

Delete a user and all associated data.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (204 No Content):**
```
No content
```

---

### Regenerate User Credentials

**POST** `/api/users/:id/regenerate`

Regenerate user's UUID and subscription token.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "uuid": "new-uuid-550e8400-e29b-41d4-a716-446655440099",
    "subscription_token": "new_sub_token_xyz789",
    "updated_at": "2026-03-25T12:45:00Z"
  }
}
```

---

## Inbounds

### List Inbounds

**GET** `/api/inbounds`

Get a list of all inbounds.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `core_id` (optional): Filter by core ID
- `protocol` (optional): Filter by protocol
- `is_enabled` (optional): Filter by enabled status

**Response (200 OK):**
```json
{
  "success": true,
  "inbounds": [
    {
      "id": 1,
      "name": "VMess-443",
      "protocol": "vmess",
      "core_id": 1,
      "listen_address": "0.0.0.0",
      "port": 443,
      "is_enabled": true,
      "tls_enabled": true,
      "config_json": "{\"clients\":[]}",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-25T10:00:00Z"
    }
  ]
}
```

---

### Create Inbound

**POST** `/api/inbounds`

Create a new inbound.

**Headers:**
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request:**
```json
{
  "name": "VLESS-8443",
  "protocol": "vless",
  "core_id": 1,
  "listen_address": "0.0.0.0",
  "port": 8443,
  "config_json": "{\"clients\":[]}",
  "tls_enabled": true,
  "is_enabled": true
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "inbound": {
    "id": 2,
    "name": "VLESS-8443",
    "protocol": "vless",
    "core_id": 1,
    "port": 8443,
    "is_enabled": true,
    "created_at": "2026-03-25T12:00:00Z"
  }
}
```

---

### Get Inbound

**GET** `/api/inbounds/:id`

Get details of a specific inbound.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "inbound": {
    "id": 1,
    "name": "VMess-443",
    "protocol": "vmess",
    "core_id": 1,
    "listen_address": "0.0.0.0",
    "port": 443,
    "config_json": "{\"clients\":[]}",
    "tls_enabled": true,
    "is_enabled": true,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-03-25T10:00:00Z"
  }
}
```

---

### Update Inbound

**PUT** `/api/inbounds/:id`

Update an inbound configuration.

**Headers:**
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request:**
```json
{
  "name": "VMess-443-Updated",
  "port": 10443,
  "is_enabled": false
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "inbound": {
    "id": 1,
    "name": "VMess-443-Updated",
    "port": 10443,
    "is_enabled": false,
    "updated_at": "2026-03-25T13:00:00Z"
  }
}
```

---

### Delete Inbound

**DELETE** `/api/inbounds/:id`

Delete an inbound.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (204 No Content):**
```
No content
```

---

## Settings

### Get Monitoring Settings

**GET** `/api/settings/monitoring`

Get current monitoring mode configuration.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "mode": "lite",
  "interval": 60
}
```

---

### Update Monitoring Settings

**PUT** `/api/settings/monitoring`

Update monitoring mode (lite or full).

**Headers:**
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request:**
```json
{
  "mode": "full"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "mode": "full",
  "message": "Monitoring mode updated successfully"
}
```

---

## Cores

### List Cores

**GET** `/api/cores`

Get a list of all proxy cores.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "cores": [
    {
      "id": 1,
      "name": "singbox",
      "version": "1.13.8",
      "is_enabled": true,
      "is_running": false,
      "uptime_seconds": 0,
      "restart_count": 0
    },
    {
      "id": 2,
      "name": "xray",
      "version": "26.3.27",
      "is_enabled": true,
      "is_running": true,
      "uptime_seconds": 3600,
      "restart_count": 2
    },
    {
      "id": 3,
      "name": "mihomo",
      "version": "1.19.23",
      "is_enabled": true,
      "is_running": false,
      "uptime_seconds": 0,
      "restart_count": 0
    }
  ]
}
```

---

### Get Core Status

**GET** `/api/cores/:name/status`

Get detailed status of a specific core.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "core": {
    "name": "singbox",
    "version": "1.13.8",
    "is_running": false,
    "pid": null,
    "uptime_seconds": 0,
    "memory_usage_bytes": 0,
    "cpu_usage_percent": 0.0
  }
}
```

---

### Start Core

**POST** `/api/cores/:name/start`

Start a proxy core.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Core singbox started successfully"
}
```

---

### Stop Core

**POST** `/api/cores/:name/stop`

Stop a proxy core.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Core singbox stopped successfully"
}
```

---

### Restart Core

**POST** `/api/cores/:name/restart`

Restart a proxy core.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Core singbox restarted successfully"
}
```

---

## Stats

### Get Dashboard Stats

**GET** `/api/stats/dashboard`

Get aggregated statistics for the dashboard.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "stats": {
    "total_users": 150,
    "active_users": 120,
    "online_users": 45,
    "total_inbounds": 25,
    "total_traffic_used_bytes": 1099511627776,
    "total_traffic_limit_bytes": 10995116277760,
    "cores_running": 2,
    "cores_total": 3
  }
}
```

---

### Get Active Connections

**GET** `/api/stats/connections`

Get list of active connections.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `user_id` (optional): Filter by user ID

**Response (200 OK):**
```json
{
  "success": true,
  "connections": [
    {
      "user_id": 1,
      "username": "user1",
      "inbound_id": 1,
      "inbound_name": "VMess-443",
      "protocol": "vmess",
      "ip_address": "192.168.1.100",
      "connected_at": "2026-03-25T11:00:00Z",
      "traffic_upload_bytes": 1048576,
      "traffic_download_bytes": 10485760
    }
  ]
}
```

---

### Disconnect User

**POST** `/api/stats/user/:user_id/disconnect`

Force disconnect a user from all active connections.

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "User disconnected successfully",
  "disconnected_count": 3
}
```

---

## Error Responses

All endpoints may return the following error responses:

### 400 Bad Request
```json
{
  "success": false,
  "error": "Validation failed",
  "details": {
    "username": "Username is required",
    "email": "Invalid email format"
  }
}
```

### 401 Unauthorized
```json
{
  "success": false,
  "error": "Unauthorized",
  "message": "Invalid or expired token"
}
```

### 403 Forbidden
```json
{
  "success": false,
  "error": "Forbidden",
  "message": "Insufficient permissions"
}
```

### 404 Not Found
```json
{
  "success": false,
  "error": "Not found",
  "message": "Resource not found"
}
```

### 500 Internal Server Error
```json
{
  "success": false,
  "error": "Internal server error",
  "message": "An unexpected error occurred"
}
```

---

## Rate Limiting

API endpoints are rate limited to prevent abuse:

- **Login endpoint**: 5 requests per minute per IP
- **Subscription endpoints**: 30 requests per minute per token
- **All other endpoints**: 100 requests per minute per user

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1648234567
```

**429 Too Many Requests:**
```json
{
  "success": false,
  "error": "Rate limit exceeded",
  "retry_after": 60
}
```

---

## Health Check

**GET** `/health`

Public health check endpoint (no authentication required).

**Response (200 OK):**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "2h30m15s",
  "database": "connected",
  "timestamp": "2026-03-25T12:00:00Z"
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "unhealthy",
  "version": "0.1.0",
  "uptime": "2h30m15s",
  "database": "disconnected",
  "timestamp": "2026-03-25T12:00:00Z"
}
```

---

## Subscriptions (Public)

These endpoints use token-based authentication instead of JWT.

### Get V2Ray Subscription

**GET** `/sub/:token`

**Response (200 OK):**
```
vmess://eyJhZGQiOiJleGFtcGxlLmNvbSIsInBvcnQiOiI0NDMiLC...
```

---

### Get Clash Subscription

**GET** `/sub/:token/clash`

**Response (200 OK):**
```yaml
port: 7890
socks-port: 7891
allow-lan: false
mode: rule
proxies:
  - name: "VMess-443"
    type: vmess
    server: example.com
    port: 443
    ...
proxy-groups:
  - name: "PROXY"
    type: select
    proxies:
      - "VMess-443"
rules:
  - MATCH,PROXY
```

---

### Get Sing-box Subscription

**GET** `/sub/:token/singbox`

**Response (200 OK):**
```json
{
  "log": {
    "level": "info"
  },
  "inbounds": [],
  "outbounds": [
    {
      "type": "vmess",
      "tag": "VMess-443",
      "server": "example.com",
      "server_port": 443,
      ...
    }
  ]
}
```

---

### Get QR Code

**GET** `/sub/:token/qr`

Get QR code for subscription URL.

**Response (200 OK):**
```
PNG image data
```

---

**API Version:** 0.1.0  
**Last Updated:** March 2026
