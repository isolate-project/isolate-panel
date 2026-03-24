# Phase 7 Implementation Plan: Additional Cores (Xray + Mihomo)

## Overview

**Goal:** Full integration of Xray and Mihomo cores alongside existing Sing-box integration.

**Timeline:** 2 weeks
**Status:** Not started

---

## Current State

### ✅ Completed (Phase 1-6)
- **Sing-box:** Full integration (config generator, lifecycle, stats via Clash API)
- **Stats Clients:** Xray gRPC and Mihomo REST stats clients implemented (Phase 6)
- **Protocol Registry:** 25 protocols defined with core support mapping
- **Port Manager:** Basic port validation exists

### ❌ Missing for Phase 7
- **Xray:** Config generator, Docker integration, Supervisord config, lifecycle
- **Mihomo:** Config generator, Docker integration, Supervisord config, lifecycle
- **HAProxy:** Post-MVP (excluded from Phase 7)

---

## Protocol Support Matrix

### Inbound Protocols (MVP)

| Protocol | Sing-box | Xray | Mihomo | Priority |
|----------|----------|------|--------|----------|
| **HTTP** | ✅ | ✅ | ✅ | P0 |
| **SOCKS5** | ✅ | ✅ | ✅ | P0 |
| **Mixed** | ✅ | ❌ | ✅ | P0 |
| **Shadowsocks** | ✅ | ✅ | ✅ | P0 |
| **VMess** | ✅ | ✅ | ✅ | P0 |
| **VLESS** | ✅ | ✅ | ✅ | P0 |
| **Trojan** | ✅ | ✅ | ✅ | P0 |
| **Hysteria2** | ✅ | ✅ | ✅ | P0 |
| **TUIC v4** | ✅ | ❌ | ✅ | P0 |
| **TUIC v5** | ✅ | ❌ | ✅ | P0 |
| **Naive** | ✅ | ❌ | ❌ | P0 |
| **Redirect** | ✅ | ❌ | ✅ | P0 |
| **XHTTP** | ❌ | ✅ | ❌ | P0 (Xray exclusive) |
| **Mieru** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **Sudoku** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **TrustTunnel** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **ShadowsocksR** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **Snell** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |

### Outbound Protocols (MVP)

| Protocol | Sing-box | Xray | Mihomo | Priority |
|----------|----------|------|--------|----------|
| **Direct** | ✅ | ✅ | ✅ | P0 |
| **Block** | ✅ | ✅ | ✅ | P0 |
| **DNS** | ✅ | ✅ | ✅ | P0 |
| **HTTP** | ✅ | ✅ | ✅ | P0 |
| **SOCKS5** | ✅ | ✅ | ✅ | P0 |
| **Shadowsocks** | ✅ | ✅ | ✅ | P0 |
| **VMess** | ✅ | ✅ | ✅ | P0 |
| **VLESS** | ✅ | ✅ | ✅ | P0 |
| **Trojan** | ✅ | ✅ | ✅ | P0 |
| **Hysteria** | ✅ | ✅ | ✅ | P0 |
| **Hysteria2** | ✅ | ✅ | ✅ | P0 |
| **TUIC** | ✅ | ❌ | ✅ | P0 |
| **Tor** | ✅ | ❌ | ❌ | P0 |
| **XHTTP** | ❌ | ✅ | ❌ | P0 (Xray exclusive) |
| **Mieru** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **Sudoku** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **TrustTunnel** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **ShadowsocksR** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **Snell** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |
| **MASQUE** | ❌ | ❌ | ✅ | P0 (Mihomo exclusive) |

---

## Phase 7.1: Xray Integration (1 week)

### 7.1.1 Docker & Infrastructure

**Files to create/modify:**
- `Dockerfile` - Add Xray binary installation
- `docker-compose.yml` - Add Xray volume mounts
- `configs/supervisord.conf` - Add Xray program configuration

**Xray Installation:**
```dockerfile
# Download Xray core
RUN wget -q https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-linux-64.zip -O /tmp/xray.zip && \
    unzip -q /tmp/xray.zip -d /usr/local/bin/ && \
    rm /tmp/xray.zip && \
    chmod +x /usr/local/bin/xray
```

**Supervisord Configuration:**
```ini
[program:xray]
command=/usr/local/bin/xray run -c /etc/isolate-panel/cores/xray/config.json
autostart=false
autorestart=true
stderr_logfile=/var/log/isolate-panel/xray.err.log
stdout_logfile=/var/log/isolate-panel/xray.out.log
user=root
```

### 7.1.2 Config Generator (`internal/cores/xray/config.go`)

**Structure:**
```go
package xray

type Config struct {
    Log      *LogConfig      `json:"log"`
    API      *APIConfig      `json:"api"`
    Stats    *StatsConfig    `json:"stats"`
    Policy   *PolicyConfig   `json:"policy"`
    Inbounds []Inbound       `json:"inbounds"`
    Outbounds []Outbound     `json:"outbounds"`
    Routing  *RoutingConfig  `json:"routing"`
}

type Inbound struct {
    Tag      string          `json:"tag"`
    Listen   string          `json:"listen"`
    Port     int             `json:"port"`
    Protocol string          `json:"protocol"`
    Settings json.RawMessage `json:"settings"`
    StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
    Sniffing *SniffingConfig `json:"sniffing,omitempty"`
}

// GenerateConfig generates Xray config from database models
func GenerateConfig(db *gorm.DB, coreID uint) (*Config, error)
// ValidateConfig validates Xray config
func ValidateConfig(config *Config) error
// WriteConfig writes config to file
func WriteConfig(config *Config, path string) error
```

**Key Features:**
- gRPC Stats API enabled (port 10085)
- gRPC HandlerService for dynamic user management
- Policy configuration for per-user stats
- Routing rules for inbound/outbound mapping

### 7.1.3 Protocol Implementations

**Inbound Settings Structures:**
```go
// VMess inbound settings
type VMessInboundSettings struct {
    Clients []VMessClient `json:"clients"`
    DefaultSecurity string `json:"defaultSecurity"`
}

type VMessClient struct {
    ID       string `json:"id"`
    Level    int    `json:"level"`
    Email    string `json:"email"`
    AlterID  int    `json:"alterId"`
}

// VLESS inbound settings
type VLESSInboundSettings struct {
    Clients []VLESSClient `json:"clients"`
    Decryption    string  `json:"decryption"`
    Fallback      string  `json:"fallback,omitempty"`
}

// XHTTP inbound settings (Xray exclusive)
type XHTTPInboundSettings struct {
    Clients []XHTTPClient `json:"clients"`
    Host    string        `json:"host"`
    Path    string        `json:"path"`
}
```

### 7.1.4 Port Allocation

**Range:** 20000-29999 (1000 ports)

**Update `internal/services/port_manager.go`:**
```go
const (
    SingboxPortStart = 10000
    SingboxPortEnd   = 19999
    XrayPortStart    = 20000
    XrayPortEnd      = 29999
    MihomoPortStart  = 30000
    MihomoPortEnd    = 39999
)
```

### 7.1.5 Integration with Core Lifecycle

**Update `internal/services/core_lifecycle.go`:**
- Add Xray start/stop/restart methods
- Config regeneration on changes
- Lazy loading (start when first inbound created)

### 7.1.6 Testing

**Unit Tests:**
- `internal/cores/xray/config_test.go` - Config generation tests
- Protocol-specific tests for each inbound/outbound type
- Port allocation tests

**Integration Tests:**
- Xray starts successfully via supervisord
- gRPC Stats API returns correct data
- Dynamic user addition/removal works
- Config reload applies changes

---

## Phase 7.2: Mihomo Integration (1 week)

### 7.2.1 Docker & Infrastructure

**Files to create/modify:**
- `Dockerfile` - Add Mihomo binary installation
- `docker-compose.yml` - Add Mihomo volume mounts
- `configs/supervisord.conf` - Add Mihomo program configuration

**Mihomo Installation:**
```dockerfile
# Download Mihomo (Clash.Meta)
RUN wget -q https://github.com/MetaCubeX/mihomo/releases/download/v1.19.21/mihomo-linux-amd64-v1.19.21.gz -O /tmp/mihomo.gz && \
    gunzip -q /tmp/mihomo.gz && \
    mv mihomo /usr/local/bin/ && \
    chmod +x /usr/local/bin/mihomo
```

**Supervisord Configuration:**
```ini
[program:mihomo]
command=/usr/local/bin/mihomo -d /etc/isolate-panel/cores/mihomo -f /etc/isolate-panel/cores/mihomo/config.yaml
autostart=false
autorestart=true
stderr_logfile=/var/log/isolate-panel/mihomo.err.log
stdout_logfile=/var/log/isolate-panel/mihomo.out.log
user=root
```

### 7.2.2 Config Generator (`internal/cores/mihomo/config.go`)

**Structure:**
```go
package mihomo

type Config struct {
    Port              int              `yaml:"port"`
    SocksPort         int              `yaml:"socks-port"`
    MixedPort         int              `yaml:"mixed-port"`
    AllowLan          bool             `yaml:"allow-lan"`
    Mode              string           `yaml:"mode"`
    LogLevel          string           `yaml:"log-level"`
    ExternalController string          `yaml:"external-controller"`
    Secret            string           `yaml:"secret"`
    Inbounds          []Inbound        `yaml:"-"` // Custom field for our inbounds
    Outbounds         []Outbound       `yaml:"proxies"`
    Rules             []string         `yaml:"rules"`
}

type Inbound struct {
    Name     string      `yaml:"name"`
    Type     string      `yaml:"type"`
    Port     int         `yaml:"port"`
    Listen   string      `yaml:"listen,omitempty"`
    Settings interface{} `yaml:",inline"`
}

// GenerateConfig generates Mihomo config from database models
func GenerateConfig(db *gorm.DB, coreID uint) (*Config, error)
// ValidateConfig validates Mihomo config
func ValidateConfig(config *Config) error
// WriteConfig writes config to YAML file
func WriteConfig(config *Config, path string) error
```

**Key Features:**
- External Controller API enabled (port 9091)
- Secret for API authentication
- Clash-compatible configuration format

### 7.2.3 Protocol Implementations

**Mihomo Exclusive Protocols:**

```go
// Mieru inbound (Mihomo exclusive)
type MieruInbound struct {
    Name     string `yaml:"name"`
    Type     string `yaml:"type"` // "mieru"
    Port     int    `yaml:"port"`
    Listen   string `yaml:"listen"`
    Users    []MieruUser `yaml:"users"`
}

type MieruUser struct {
    Name     string `yaml:"name"`
    Password string `yaml:"password"`
}

// Sudoku inbound (Mihomo exclusive)
type SudokuInbound struct {
    Name     string `yaml:"name"`
    Type     string `yaml:"type"` // "sudoku"
    Port     int    `yaml:"port"`
    Listen   string `yaml:"listen"`
    Password string `yaml:"password"`
}

// ShadowsocksR inbound (Mihomo exclusive)
type ShadowsocksRInbound struct {
    Name       string `yaml:"name"`
    Type       string `yaml:"type"` // "ssr"
    Port       int    `yaml:"port"`
    Listen     string `yaml:"listen"`
    Cipher     string `yaml:"cipher"`
    Password   string `yaml:"password"`
    Protocol   string `yaml:"protocol"`
    Obfs       string `yaml:"obfs"`
}
```

### 7.2.4 Port Allocation

**Range:** 30000-39999 (1000 ports)

### 7.2.5 Integration with Core Lifecycle

Similar to Xray integration.

### 7.2.6 Testing

Similar to Xray testing.

---

## Implementation Order

### Week 1: Xray
1. **Day 1:** Docker + Supervisord setup
2. **Day 2-3:** Config generator (common protocols)
3. **Day 4:** XHTTP exclusive protocol
4. **Day 5:** Testing + bug fixes

### Week 2: Mihomo
1. **Day 1:** Docker + Supervisord setup
2. **Day 2-3:** Config generator (common protocols)
3. **Day 4:** Mihomo exclusive protocols (Mieru, Sudoku, SSR, Snell, etc.)
4. **Day 5:** Testing + bug fixes

---

## Acceptance Criteria

### For Xray:
- ✅ Xray binary installed in Docker image
- ✅ Xray starts/stops via supervisord
- ✅ Config generator produces valid JSON
- ✅ All MVP inbound protocols supported (HTTP, SOCKS5, Shadowsocks, VMess, VLESS, Trojan, Hysteria2, XHTTP)
- ✅ All MVP outbound protocols supported
- ✅ gRPC Stats API returns per-user statistics
- ✅ gRPC HandlerService allows dynamic user management
- ✅ Port allocation in 20000-29999 range
- ✅ Unit tests with >80% coverage
- ✅ Integration tests pass

### For Mihomo:
- ✅ Mihomo binary installed in Docker image
- ✅ Mihomo starts/stops via supervisord
- ✅ Config generator produces valid YAML
- ✅ All MVP inbound protocols supported (HTTP, SOCKS5, Mixed, Shadowsocks, VMess, VLESS, Trojan, Hysteria2, TUIC, Mieru, Sudoku, TrustTunnel, SSR, Snell)
- ✅ All MVP outbound protocols supported
- ✅ External Controller API returns connection data
- ✅ Secret-based API authentication works
- ✅ Port allocation in 30000-39999 range
- ✅ Unit tests with >80% coverage
- ✅ Integration tests pass

### For Both:
- ✅ Core lifecycle management works (lazy loading)
- ✅ Stats collection via unified interface (Phase 6)
- ✅ Connection tracking works (Phase 6)
- ✅ Quota enforcement via graceful reload
- ✅ Frontend shows correct protocols per core
- ✅ Bundle size remains under 200 KB gzipped

---

## Technical Notes

### Xray gRPC Configuration
```json
{
  "api": {
    "tag": "api",
    "services": ["HandlerService", "StatsService"]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserUplink": true,
        "statsUserDownlink": true
      }
    }
  },
  "inbounds": [
    {
      "tag": "api",
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      }
    }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "inboundTag": ["api"],
        "outboundTag": "api"
      }
    ]
  }
}
```

### Mihomo External Controller Configuration
```yaml
external-controller: 127.0.0.1:9091
secret: your-api-secret-here
```

### Config File Locations
```
/etc/isolate-panel/
├── cores/
│   ├── singbox/
│   │   └── config.json
│   ├── xray/
│   │   └── config.json
│   └── mihomo/
│       └── config.yaml
├── certs/
│   └── ...
└── config.yaml (panel config)
```

---

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Xray gRPC API complexity | High | Medium | Use existing xray-core protobuf definitions, start with basic stats |
| Mihomo YAML config validation | Medium | High | Use strict YAML unmarshaling with validation |
| Port conflicts | Medium | Low | Port manager with database-level locking |
| Core startup failures | High | Medium | Comprehensive logging, health checks, auto-restart |
| Performance degradation | High | Low | Load testing with 1000+ concurrent connections |

---

## Next Steps

1. **Review and approve this plan**
2. **Create Phase 7.1 sub-tasks in project tracker**
3. **Start with Xray Docker integration**
4. **Implement incrementally with tests**
5. **Demo at end of each week**
