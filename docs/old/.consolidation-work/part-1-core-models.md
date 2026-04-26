# Core Domain Models

> **Part 1 of Consolidated HAProxy Documentation**  
> **Date**: 2026-04-19  
> **Status**: Ready for Implementation  
> **Purpose**: Complete Go struct definitions with dynamic port configuration

---

## Table of Contents

1. [Port Configuration Types](#1-port-configuration-types)
2. [Core Domain Models](#2-core-domain-models)
3. [HAProxy Configuration Structs](#3-haproxy-configuration-structs)
4. [Port Management Types](#4-port-management-types)
5. [Validation Types](#5-validation-types)
6. [Database Schema](#6-database-schema)

---

## 1. Port Configuration Types

### 1.1 Port Assignment Modes

```go
// PortAssignmentMode defines how ports are allocated to inbounds and services
type PortAssignmentMode string

const (
    // PortAssignmentAuto - System auto-assigns next available port from pool
    // Recommended for new installations
    PortAssignmentAuto PortAssignmentMode = "auto"
    
    // PortAssignmentManual - User specifies exact port number
    // For advanced users who need specific port assignments
    PortAssignmentManual PortAssignmentMode = "manual"
    
    // PortAssignmentRange - Legacy fixed range assignment
    // Used for backward compatibility with existing installations
    PortAssignmentRange PortAssignmentMode = "range"
    
    // PortAssignmentRandom - Random port from available pool
    // Useful for security through obscurity
    PortAssignmentRandom PortAssignmentMode = "random"
)
```

### 1.2 Base Port Configuration

```go
// PortConfig defines how a port is assigned and configured
// This is the foundation for all port-related configuration in the system
type PortConfig struct {
    ID uint `gorm:"primaryKey" json:"id"`
    
    // Name is a human-readable identifier for this port configuration
    Name string `gorm:"not null" json:"name"`
    
    // AssignmentMode determines how the port is allocated
    // Values: "auto", "manual", "range", "random"
    AssignmentMode PortAssignmentMode `gorm:"not null" json:"assignment_mode"`
    
    // Port is the actual port number (meaning depends on assignment mode)
    // For manual mode: the user-specified port
    // For auto/random: populated after assignment
    // For range: starting port of the range
    Port int `json:"port,omitempty"`
    
    // AutoStartPort is the starting port for auto assignment pool
    // System searches for available ports starting from this value
    AutoStartPort int `json:"auto_start_port,omitempty"`
    
    // RangeStart and RangeEnd define boundaries for range/random assignment
    RangeStart int `json:"range_start,omitempty"`
    RangeEnd   int `json:"range_end,omitempty"`
    
    // Protocol constraints - what protocols can use this port
    Protocol  string `json:"protocol,omitempty"`  // "tcp", "udp", "tcp+udp"
    Transport string `json:"transport,omitempty"` // "tcp", "ws", "grpc", "quic"
    
    // CoreType association - which proxy core owns this port
    // Values: "xray", "singbox", "mihomo"
    CoreType string `json:"core_type,omitempty"`
    
    // IsPrivileged indicates if port requires root privileges (< 1024)
    IsPrivileged bool `json:"is_privileged,omitempty"`
    
    // RequiresRoot indicates if running as root is mandatory
    RequiresRoot bool `json:"requires_root,omitempty"`
    
    // HealthCheckPort is an alternative port for health checks (if different from main port)
    HealthCheckPort int `json:"health_check_port,omitempty"`
    
    // HealthCheckPath is the HTTP path for health check probes
    HealthCheckPath string `json:"health_check_path,omitempty"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 1.3 Frontend Port Configuration

```go
// FrontendPortConfig for public-facing HAProxy ports (e.g., 443, 8443)
// These ports receive incoming client connections
type FrontendPortConfig struct {
    PortConfig
    
    // SNIEnabled determines if SNI-based routing is available on this port
    SNIEnabled bool `json:"sni_enabled"`
    
    // TLSTermination determines if HAProxy terminates TLS or passes through
    TLSTermination bool `json:"tls_termination"`
    
    // CertPath and KeyPath for TLS certificates
    CertPath string `json:"cert_path,omitempty"`
    KeyPath  string `json:"key_path,omitempty"`
    
    // ALPN protocols advertised during TLS handshake
    // Examples: ["h2", "http/1.1"]
    ALPN []string `json:"alpn,omitempty"`
    
    // RateLimit configuration for DDoS protection
    RateLimit RateLimitConfig `json:"rate_limit,omitempty"`
    
    // BindAddress is the interface to bind to (default: "*" for all interfaces)
    BindAddress string `json:"bind_address,omitempty"`
}

// RateLimitConfig for connection rate limiting
type RateLimitConfig struct {
    Enabled bool `json:"enabled"`
    
    // Requests per time window
    Requests int `json:"requests"`
    
    // Time window for rate limiting
    Window time.Duration `json:"window"`
    
    // Burst allows temporary exceeding of the rate limit
    Burst int `json:"burst,omitempty"`
}
```

### 1.4 Backend Port Configuration

```go
// BackendPortConfig for internal HAProxy-to-core communication ports
// These ports are not exposed publicly but used for internal routing
type BackendPortConfig struct {
    PortConfig
    
    // UseProxyProtocol enables PROXY protocol for preserving client IP
    // Note: Only Xray supports PROXY protocol v2 for TCP/WS transports
    // Sing-box and Mihomo do NOT support PROXY protocol (removed in Sing-box 1.6.0+)
    UseProxyProtocol bool `json:"use_proxy_protocol"`
    
    // ProxyVersion is the PROXY protocol version (1 or 2)
    // Version 2 is recommended for modern deployments
    ProxyVersion int `json:"proxy_version,omitempty"`
    
    // UnixSocketPath for Unix domain socket communication (alternative to TCP port)
    UnixSocketPath string `json:"unix_socket_path,omitempty"`
    
    // MaxConnections limits concurrent connections to this backend
    MaxConnections int `json:"max_connections,omitempty"`
    
    // ServerAddress is the backend server address (default: "127.0.0.1")
    ServerAddress string `json:"server_address,omitempty"`
}
```

### 1.5 Direct Port Configuration

```go
// DirectPortConfig for UDP/QUIC ports that bypass HAProxy
// These ports are used for protocols that cannot be proxied (Hysteria2, TUIC, KCP)
type DirectPortConfig struct {
    PortConfig
    
    // UDPOnly indicates this port is UDP-only (no TCP support)
    UDPOnly bool `json:"udp_only"`
    
    // QUICEnabled indicates QUIC protocol support
    QUICEnabled bool `json:"quic_enabled"`
    
    // HostNetwork indicates if container requires host networking mode
    // Required for UDP protocols to function correctly
    HostNetwork bool `json:"host_network"`
    
    // MTU setting for UDP protocols (default: 1350 for Hysteria2)
    MTU int `json:"mtu,omitempty"`
}
```

### 1.6 Global Port Configuration

```go
// GlobalPortConfig central configuration for all ports in the system
type GlobalPortConfig struct {
    ID uint `gorm:"primaryKey" json:"id"`
    
    // MainHTTPSPort is the primary HTTPS frontend port (default: 443)
    // All SNI-based routing happens on this port
    MainHTTPSPort FrontendPortConfig `json:"main_https_port" gorm:"embedded;embeddedPrefix:main_"`
    
    // AltHTTPSPort is an alternative HTTPS port (e.g., 8443)
    // Used for additional inbounds or testing
    AltHTTPSPort FrontendPortConfig `json:"alt_https_port,omitempty" gorm:"embedded;embeddedPrefix:alt_"`
    
    // HTTPPort is the HTTP fallback port (default: 8080)
    // Used for redirects to HTTPS or plain HTTP services
    HTTPPort FrontendPortConfig `json:"http_port,omitempty" gorm:"embedded;embeddedPrefix:http_"`
    
    // StatsPort is the HAProxy statistics and Prometheus metrics port (default: 8404)
    StatsPort FrontendPortConfig `json:"stats_port" gorm:"embedded;embeddedPrefix:stats_"`
    
    // PanelPort is the web panel UI port (default: 8080)
    PanelPort FrontendPortConfig `json:"panel_port" gorm:"embedded;embeddedPrefix:panel_"`
    
    // Backend port pools for auto assignment
    XrayPortPool    PortPoolConfig `json:"xray_port_pool" gorm:"embedded;embeddedPrefix:xray_pool_"`
    SingboxPortPool PortPoolConfig `json:"singbox_port_pool" gorm:"embedded;embeddedPrefix:singbox_pool_"`
    MihomoPortPool  PortPoolConfig `json:"mihomo_port_pool" gorm:"embedded;embeddedPrefix:mihomo_pool_"`
    
    // Direct UDP ports (bypass HAProxy)
    Hysteria2Port DirectPortConfig `json:"hysteria2_port,omitempty" gorm:"embedded;embeddedPrefix:hysteria2_"`
    TUICPort      DirectPortConfig `json:"tuic_port,omitempty" gorm:"embedded;embeddedPrefix:tuic_"`
    
    // ReservedPorts are system ports that cannot be used (e.g., 22, 80, 443)
    ReservedPorts []int `json:"reserved_ports" gorm:"type:json"`
    
    // Validation settings
    MinPort         int  `json:"min_port"`         // Default: 1024
    MaxPort         int  `json:"max_port"`         // Default: 65535
    AllowPrivileged bool `json:"allow_privileged"` // Allow ports < 1024
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// PortPoolConfig defines a pool for auto port assignment
type PortPoolConfig struct {
    // StartPort is the beginning of the allocation pool
    StartPort int `json:"start_port"`
    
    // EndPort is the end of the allocation pool
    EndPort int `json:"end_port"`
    
    // BlockSize is the number of ports allocated per core instance
    BlockSize int `json:"block_size,omitempty"`
    
    // NextPort is the next available port pointer (for auto assignment)
    NextPort int `json:"next_port,omitempty"`
}
```

---

## 2. Core Domain Models

### 2.1 Domain Type Enumeration

```go
// DomainType defines the type of domain for SNI routing
type DomainType string

const (
    // DomainTypeXray - Xray core inbounds (VLESS, VMess, Trojan, etc.)
    DomainTypeXray DomainType = "xray"
    
    // DomainTypeSingbox - Sing-box TCP-based inbounds (VMess, VLESS, Trojan, ShadowTLS)
    DomainTypeSingbox DomainType = "singbox"
    
    // DomainTypeSingboxQUIC - Sing-box QUIC-based inbounds (Hysteria2, TUIC)
    // Note: These bypass HAProxy and use direct ports
    DomainTypeSingboxQUIC DomainType = "singbox_quic"
    
    // DomainTypeMihomo - Mihomo (Clash Meta) inbounds
    DomainTypeMihomo DomainType = "mihomo"
    
    // DomainTypeDefault - Fallback domain for panel UI or default routing
    DomainTypeDefault DomainType = "default"
)
```

### 2.2 Domain Model

```go
// Domain represents an SNI domain for HAProxy routing
// Each domain maps to a specific backend based on SNI or path matching
type Domain struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `gorm:"uniqueIndex;not null" json:"name"` // Full domain name (e.g., "vless.example.com")
    
    // Type determines which core handles this domain
    Type DomainType `gorm:"not null" json:"type"`
    
    // CoreID links to the specific Core model instance
    CoreID *uint `gorm:"index" json:"core_id,omitempty"`
    
    // IsActive controls whether this domain is included in HAProxy config
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    // IsCDN indicates if domain is behind Cloudflare/Arvan/etc (affects client IP detection)
    IsCDN bool `gorm:"default:false" json:"is_cdn"`
    
    // PathPrefix enables path-based routing for WebSocket/gRPC/XHTTP
    // Example: "/ws", "/grpc", "/upgrade"
    PathPrefix string `gorm:"default:''" json:"path_prefix,omitempty"`
    
    // EnableProxyProtocol enables PROXY protocol v2 for this domain
    // Only effective for Xray TCP/WS transports
    EnableProxyProtocol bool `gorm:"default:true" json:"enable_proxy_protocol"`
    
    // TLS settings (optional - HAProxy can terminate or passthrough)
    TLSTerminate bool   `gorm:"default:false" json:"tls_terminate"`
    CertPath     string `gorm:"default:''" json:"cert_path,omitempty"`
    KeyPath      string `gorm:"default:''" json:"key_path,omitempty"`
    
    // Port configuration reference
    PortConfigID *uint       `json:"port_config_id,omitempty"`
    PortConfig   *PortConfig `json:"port_config,omitempty" gorm:"foreignKey:PortConfigID"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 2.3 Backend Configuration

```go
// BackendConfig represents HAProxy backend configuration
// Defines how HAProxy connects to proxy core instances
type BackendConfig struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `gorm:"not null" json:"name"` // Unique backend name (e.g., "xray_vless_ws")
    
    // Type determines backend mode (tcp vs http) and routing behavior
    Type DomainType `json:"type"`
    
    // ServerAddress is the backend server IP or hostname
    // Default: "127.0.0.1" for local cores
    ServerAddress string `json:"server_address"`
    
    // BackendPort is the DYNAMIC port where the core listens
    // Replaces fixed ports (10001, 9090, 9091) with configurable values
    // Retrieved from PortConfig or PortAssignment
    BackendPort int `json:"backend_port"`
    
    // PortConfigID links to the port configuration for this backend
    PortConfigID *uint       `json:"port_config_id,omitempty"`
    PortConfig   *PortConfig `json:"port_config,omitempty" gorm:"foreignKey:PortConfigID"`
    
    // UseUnixSocket enables Unix domain socket communication
    UseUnixSocket bool `json:"use_unix_socket"`
    SocketPath    string `json:"socket_path,omitempty"`
    
    // Connection settings
    MaxConnections int           `json:"max_connections"`    // Max concurrent connections
    CheckInterval  time.Duration `json:"check_interval"`     // Health check interval
    
    // PROXY protocol configuration
    SendProxyProtocol    bool `json:"send_proxy_protocol"`    // Enable PROXY protocol
    ProxyProtocolVersion int  `json:"proxy_protocol_version"` // 1 or 2
    
    // Health check configuration
    HealthCheck HealthCheckConfig `json:"health_check" gorm:"embedded"`
    
    // Weight for load balancing (if multiple servers)
    Weight int `json:"weight,omitempty"`
    
    // IsActive includes/excludes this backend from configuration
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 2.4 Frontend Configuration

```go
// FrontendConfig represents HAProxy frontend configuration
// Defines how HAProxy accepts incoming client connections
type FrontendConfig struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `gorm:"not null" json:"name"` // Frontend name (e.g., "https-in", "https-alt")
    
    // BindAddress is the interface to bind (default: "*" for all)
    BindAddress string `json:"bind_address"`
    
    // BindPort is the DYNAMIC public port for this frontend
    // Replaces fixed ports (443, 8443, 8080) with configurable values
    BindPort int `json:"bind_port"`
    
    // PortConfigID links to the frontend port configuration
    PortConfigID *uint               `json:"port_config_id,omitempty"`
    PortConfig   *FrontendPortConfig `json:"port_config,omitempty" gorm:"foreignKey:PortConfigID"`
    
    // Mode is the proxy mode: "tcp" for L4, "http" for L7
    Mode string `json:"mode"` // "tcp" or "http"
    
    // TLS settings
    TLSEnabled  bool   `json:"tls_enabled"`
    TLSCertPath string `json:"tls_cert_path,omitempty"`
    TLSKeyPath  string `json:"tls_key_path,omitempty"`
    
    // ALPN protocols for TLS negotiation
    ALPN []string `json:"alpn,omitempty"` // ["h2", "http/1.1"]
    
    // AllowHTTP3 enables QUIC/HTTP3 support (requires HAProxy 3.1+)
    AllowHTTP3 bool `json:"allow_http3"`
    
    // SNI inspection settings (for TCP mode with TLS passthrough)
    SNIInspectionDelay time.Duration `json:"sni_inspection_delay"`
    
    // DDoS protection settings
    RateLimitEnabled  bool          `json:"rate_limit_enabled"`
    RateLimitRequests int           `json:"rate_limit_requests"`
    RateLimitWindow   time.Duration `json:"rate_limit_window"`
    
    // Routing rules
    Domains        []Domain `json:"domains" gorm:"many2many:frontend_domains;"`
    DefaultBackend string   `json:"default_backend"` // Backend name for unmatched traffic
    
    // IsActive includes/excludes this frontend from configuration
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

---

## 3. HAProxy Configuration Structs

### 3.1 Health Check Configuration

```go
// HealthCheckConfig defines backend health check parameters
type HealthCheckConfig struct {
    // Enabled turns health checks on/off
    Enabled bool `json:"enabled"`
    
    // Type is the check type: "tcp" or "http"
    Type string `json:"type"` // "tcp" or "http"
    
    // Path is the HTTP path for health checks (when Type is "http")
    Path string `json:"path,omitempty"`
    
    // Interval between health checks
    Interval time.Duration `json:"interval"`
    
    // RiseCount is consecutive successful checks to mark as UP
    RiseCount int `json:"rise_count"`
    
    // FallCount is consecutive failed checks to mark as DOWN
    FallCount int `json:"fall_count"`
    
    // Timeout for individual health check attempts
    Timeout time.Duration `json:"timeout,omitempty"`
}
```

### 3.2 Global Configuration

```go
// GlobalConfig represents HAProxy global settings
type GlobalConfig struct {
    ID uint `gorm:"primaryKey" json:"id"`
    
    // Connection limits
    MaxConnections    int `json:"max_connections"`     // Global max connections
    MaxSSLConnections int `json:"max_ssl_connections"` // SSL-specific limit
    
    // Health check spreading (0-100, percentage to spread checks)
    SpreadChecks int `json:"spread_checks"`
    
    // Stats socket for runtime API
    StatsSocket  string        `json:"stats_socket"`
    StatsTimeout time.Duration `json:"stats_timeout"`
    
    // Performance tuning
    Nbthread int `json:"nbthread"` // Number of threads (0 = auto)
    Nbproc   int `json:"nbproc"`   // Number of processes (for L4 only)
    
    // Logging configuration
    LogTarget   string `json:"log_target"`   // "stdout", "syslog", "127.0.0.1"
    LogFacility string `json:"log_facility"` // "local0", "daemon"
    LogLevel    string `json:"log_level"`    // "info", "debug", "warning"
    
    // Chroot directory (for security)
    Chroot string `json:"chroot,omitempty"`
    
    // User/Group to run as (after binding ports)
    User  string `json:"user,omitempty"`
    Group string `json:"group,omitempty"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 3.3 Defaults Configuration

```go
// DefaultsConfig represents HAProxy defaults section
type DefaultsConfig struct {
    ID uint `gorm:"primaryKey" json:"id"`
    
    // Mode is the default proxy mode
    Mode string `json:"mode"` // "tcp" or "http"
    
    // Timeouts
    TimeoutConnect time.Duration `json:"timeout_connect"` // Backend connection timeout
    TimeoutClient  time.Duration `json:"timeout_client"`  // Client inactivity timeout
    TimeoutServer  time.Duration `json:"timeout_server"`  // Server inactivity timeout
    TimeoutTunnel  time.Duration `json:"timeout_tunnel,omitempty"` // WebSocket tunnel timeout
    
    // Default options
    Option []string `json:"option,omitempty" gorm:"type:json"` // ["tcplog", "dontlognull", "http-server-close"]
    
    // Default backend (optional)
    DefaultBackend string `json:"default_backend,omitempty"`
    
    // Retries for failed connections
    Retries int `json:"retries,omitempty"`
    
    // Max connections per backend (optional)
    MaxConn int `json:"max_conn,omitempty"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 3.4 Listen Configuration

```go
// ListenConfig represents HAProxy listen section (for stats, etc.)
type ListenConfig struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `gorm:"not null" json:"name"` // Listen section name
    
    // Bind address and DYNAMIC port
    // Replaces fixed port 8404 with configurable value
    Bind string `json:"bind"` // Format: "*:8404" or "127.0.0.1:8404"
    
    // PortConfigID for dynamic port management
    PortConfigID *uint               `json:"port_config_id,omitempty"`
    PortConfig   *FrontendPortConfig `json:"port_config,omitempty" gorm:"foreignKey:PortConfigID"`
    
    // Mode: "tcp" or "http"
    Mode string `json:"mode"`
    
    // Stats enables HAProxy statistics page
    Stats bool `json:"stats,omitempty"`
    
    // StatsURI is the path for stats page (default: "/stats")
    StatsURI string `json:"stats_uri,omitempty"`
    
    // StatsRefresh is the auto-refresh interval
    StatsRefresh time.Duration `json:"stats_refresh,omitempty"`
    
    // Prometheus enables Prometheus metrics endpoint
    Prometheus bool `json:"prometheus,omitempty"`
    
    // PrometheusPath is the metrics endpoint path (default: "/metrics")
    PrometheusPath string `json:"prometheus_path,omitempty"`
    
    // ACLs and rules for this listen section
    ACLs []ListenACL `json:"acls,omitempty" gorm:"foreignKey:ListenID"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// ListenACL defines an ACL for listen section
type ListenACL struct {
    ID       uint   `gorm:"primaryKey" json:"id"`
    ListenID uint   `json:"listen_id"`
    Name     string `json:"name"`
    Criteria string `json:"criteria"` // e.g., "path /metrics"
    Action   string `json:"action"`   // e.g., "use-service prometheus-exporter"
}
```

### 3.5 Full Configuration

```go
// FullConfig represents complete HAProxy configuration
// This is the root structure used for template generation
type FullConfig struct {
    // Global settings
    Global GlobalConfig `json:"global" gorm:"embedded"`
    
    // Defaults section
    Defaults DefaultsConfig `json:"defaults" gorm:"embedded"`
    
    // Frontend definitions (one per public port)
    Frontends []FrontendConfig `json:"frontends" gorm:"foreignKey:FullConfigID"`
    
    // Backend definitions (one per core/protocol combination)
    Backends []BackendConfig `json:"backends" gorm:"foreignKey:FullConfigID"`
    
    // Listen sections (stats, Prometheus, etc.)
    Listen []ListenConfig `json:"listen,omitempty" gorm:"foreignKey:FullConfigID"`
    
    // Userlists for authentication (optional)
    Userlists []UserlistConfig `json:"userlists,omitempty" gorm:"foreignKey:FullConfigID"`
    
    // Peers for stick table synchronization (optional)
    Peers []PeerConfig `json:"peers,omitempty" gorm:"foreignKey:FullConfigID"`
    
    // Raw config lines to append (for custom configuration)
    RawConfig string `json:"raw_config,omitempty"`
}

// UserlistConfig for HTTP basic authentication
type UserlistConfig struct {
    ID          uint       `gorm:"primaryKey" json:"id"`
    FullConfigID uint      `json:"full_config_id"`
    Name        string     `json:"name"`
    Users       []UserCred `json:"users" gorm:"foreignKey:UserlistID"`
}

type UserCred struct {
    ID         uint   `gorm:"primaryKey" json:"id"`
    UserlistID uint   `json:"userlist_id"`
    Username   string `json:"username"`
    Password   string `json:"password"` // Can be encrypted or plain
    IsEncrypted bool  `json:"is_encrypted"`
}

// PeerConfig for stick table synchronization
type PeerConfig struct {
    ID           uint   `gorm:"primaryKey" json:"id"`
    FullConfigID uint   `json:"full_config_id"`
    Name         string `json:"name"`
    Address      string `json:"address"`
    Port         int    `json:"port"`
}
```

---

## 4. Port Management Types

### 4.1 Port Group

```go
// PortGroup defines a group of ports with shared configuration
// Used for managing multiple related ports (e.g., all HTTPS ports)
type PortGroup struct {
    ID   string `gorm:"primaryKey" json:"id"` // e.g., "main_https", "alt_https"
    Name string `json:"name"`                   // Human-readable name
    
    // Port range definition
    StartPort int `json:"start_port"` // Starting port number
    EndPort   int `json:"end_port"`   // Ending port number (inclusive)
    
    // Mode determines proxy behavior
    Mode string `json:"mode"` // "tcp" or "http"
    
    // TLS settings
    TLSEnabled  bool   `json:"tls_enabled"`
    TLSCertPath string `json:"tls_cert_path,omitempty"`
    
    // PortConfigID for dynamic port management
    PortConfigID *uint       `json:"port_config_id,omitempty"`
    PortConfig   *PortConfig `json:"port_config,omitempty" gorm:"foreignKey:PortConfigID"`
    
    // Feature flags
    SNIRouting     bool `json:"sni_routing"`     // Enable SNI-based routing
    PathRouting    bool `json:"path_routing"`    // Enable path-based routing
    DDoSProtection bool `json:"ddos_protection"` // Enable connection rate limiting
    
    // IsActive includes/excludes this port group
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 4.2 Inbound Port Binding

```go
// InboundPortBinding links an inbound to a port with specific routing rules
// This is the core structure for multi-port, cross-core routing
type InboundPortBinding struct {
    ID uint `gorm:"primaryKey" json:"id"`
    
    // InboundID links to the inbound configuration
    InboundID uint `json:"inbound_id" gorm:"index"`
    
    // PortGroupID links to the port group (optional - can bind directly to port)
    PortGroupID string `json:"port_group_id,omitempty" gorm:"index"`
    PortGroup   *PortGroup `json:"port_group,omitempty" gorm:"foreignKey:PortGroupID"`
    
    // Port configuration reference
    PortConfigID *uint       `json:"port_config_id,omitempty"`
    PortConfig   *PortConfig `json:"port_config,omitempty" gorm:"foreignKey:PortConfigID"`
    
    // ListenPort is the specific public port (within the group or standalone)
    // This is the DYNAMIC port value, replacing fixed ports
    ListenPort int `json:"listen_port"`
    
    // BackendPort is the DYNAMIC internal port for the core
    // Retrieved from PortConfig or auto-assigned from pool
    BackendPort int `json:"backend_port"`
    
    // Routing rules (at least one must be set for routing to work)
    SNIMatch  string `json:"sni_match,omitempty"`  // e.g., "vless.example.com"
    PathMatch string `json:"path_match,omitempty"` // e.g., "/vless-ws"
    
    // Priority for rule ordering (lower = higher priority)
    Priority int `json:"priority"`
    
    // Core assignment
    CoreType string `json:"core_type"` // "xray", "singbox", "mihomo"
    
    // HAProxy settings
    UseProxyProtocol bool   `json:"use_proxy_protocol"` // Enable PROXY protocol
    HealthCheckPath  string `json:"health_check_path,omitempty"`
    
    // IsActive includes/excludes this binding from HAProxy config
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 4.3 Port Assignment

```go
// PortAssignment tracks which ports are assigned to which inbounds
// Used for conflict detection and port lifecycle management
type PortAssignment struct {
    ID uint `gorm:"primaryKey" json:"id"`
    
    // InboundID links to the inbound
    InboundID uint `json:"inbound_id" gorm:"index"`
    
    // PortConfigID links to the port configuration
    PortConfigID uint `json:"port_config_id"`
    PortConfig   PortConfig `json:"port_config" gorm:"foreignKey:PortConfigID"`
    
    // AssignedPort is the actual port number assigned
    AssignedPort int `json:"assigned_port" gorm:"uniqueIndex"`
    
    // IsActive indicates if this assignment is currently active
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    // Assignment metadata
    AssignedAt time.Time  `json:"assigned_at"`
    ExpiresAt  *time.Time `json:"expires_at,omitempty"` // For temporary assignments
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 4.4 Port Pool

```go
// PortPool tracks available ports for auto assignment
type PortPool struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `gorm:"uniqueIndex;not null" json:"name"` // e.g., "xray-pool"
    
    // CoreType this pool serves
    CoreType string `json:"core_type"` // "xray", "singbox", "mihomo"
    
    // Port range
    StartPort int `json:"start_port"`
    EndPort   int `json:"end_port"`
    
    // BlockSize is the number of ports allocated per request
    BlockSize int `json:"block_size"` // Default: 10
    
    // NextPort is the next available port pointer
    NextPort int `json:"next_port"`
    
    // AvailablePorts tracks which ports are free (bitmask or list)
    AvailablePorts []int `json:"available_ports,omitempty" gorm:"type:json"`
    
    // IsActive enables/disables this pool
    IsActive bool `gorm:"default:true" json:"is_active"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

---

## 5. Validation Types

### 5.1 Validation Severity and Action

```go
// ValidationSeverity indicates the severity of a validation result
type ValidationSeverity string

const (
    // SeverityInfo - Port is available or can be shared without issues
    SeverityInfo ValidationSeverity = "info"
    
    // SeverityWarning - Port can be shared but requires confirmation
    SeverityWarning ValidationSeverity = "warning"
    
    // SeverityError - Port cannot be used (conflict or incompatibility)
    SeverityError ValidationSeverity = "error"
)

// ValidationAction indicates the recommended action
type ValidationAction string

const (
    // ActionAllow - Port can be used without restrictions
    ActionAllow ValidationAction = "allow"
    
    // ActionConfirm - User confirmation required before using port
    ActionConfirm ValidationAction = "confirm"
    
    // ActionBlock - Port cannot be used, select different port
    ActionBlock ValidationAction = "block"
)
```

### 5.2 Port Conflict Check

```go
// PortConflictCheck contains the result of port conflict validation
// Returned by the port validator to guide UI decisions
type PortConflictCheck struct {
    // Port being validated
    Port int `json:"port"`
    
    // Listen address (e.g., "0.0.0.0", "127.0.0.1")
    Listen string `json:"listen"`
    
    // Protocol and transport being validated
    Protocol  string `json:"protocol"`
    Transport string `json:"transport,omitempty"`
    
    // Validation result
    IsAvailable bool `json:"is_available"`
    
    // Conflicts with existing inbounds
    Conflicts []PortConflict `json:"conflicts,omitempty"`
    
    // HAProxy compatibility analysis
    HaproxyCompatible bool `json:"haproxy_compatible"`
    
    // CanSharePort indicates if HAProxy can enable port sharing
    CanSharePort bool `json:"can_share_port"`
    
    // SharingMechanism describes how ports can share ("sni", "path", "none")
    SharingMechanism string `json:"sharing_mechanism,omitempty"`
    
    // Severity and action for UI display
    Severity ValidationSeverity `json:"severity"` // "info", "warning", "error"
    Message  string             `json:"message"`   // Human-readable message (Russian)
    Action   ValidationAction   `json:"action"`   // "allow", "confirm", "block"
}

// PortConflict describes a conflict with an existing inbound
type PortConflict struct {
    // Inbound identification
    InboundID   uint   `json:"inbound_id"`
    InboundName string `json:"inbound_name"`
    
    // Protocol details of conflicting inbound
    Protocol  string `json:"protocol"`
    Transport string `json:"transport,omitempty"`
    Port      int    `json:"port"`
    Listen    string `json:"listen"`
    
    // Conflict analysis
    SameProtocol     bool `json:"same_protocol"`     // Same protocol as new inbound
    CanShare         bool `json:"can_share"`         // Can share via HAProxy
    RequiresConfirm  bool `json:"requires_confirm"`  // Needs user confirmation
}
```

### 5.3 Port Validation Request/Response

```go
// PortValidationRequest is sent from frontend to check port availability
type PortValidationRequest struct {
    Port      int    `json:"port" binding:"required,min=1,max=65535"`
    Listen    string `json:"listen,omitempty"`    // Default: "0.0.0.0"
    Protocol  string `json:"protocol" binding:"required"`
    Transport string `json:"transport,omitempty"` // Default: "tcp"
    CoreType  string `json:"core_type" binding:"required"` // "xray", "singbox", "mihomo"
}

// PortValidationResponse is returned by the validation endpoint
type PortValidationResponse struct {
    PortConflictCheck
    
    // Additional metadata
    SuggestedPorts []int `json:"suggested_ports,omitempty"` // Alternative port suggestions
    
    // HAProxy configuration preview
    HAProxyPreview string `json:"haproxy_preview,omitempty"` // Generated config snippet
}
```

---

## 6. Database Schema

### 6.1 Port Configuration Tables

```sql
-- Port configuration table
-- Stores all port configuration definitions
CREATE TABLE port_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    assignment_mode VARCHAR(20) NOT NULL CHECK (assignment_mode IN ('auto', 'manual', 'range', 'random')),
    port INTEGER,
    auto_start_port INTEGER,
    range_start INTEGER,
    range_end INTEGER,
    protocol VARCHAR(20) DEFAULT 'tcp',
    transport VARCHAR(50),
    core_type VARCHAR(20) CHECK (core_type IN ('xray', 'singbox', 'mihomo')),
    is_privileged BOOLEAN DEFAULT FALSE,
    requires_root BOOLEAN DEFAULT FALSE,
    health_check_port INTEGER,
    health_check_path VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Validation constraints based on assignment mode
    CONSTRAINT valid_port_range CHECK (
        (assignment_mode = 'manual' AND port BETWEEN 1 AND 65535) OR
        (assignment_mode = 'auto' AND auto_start_port BETWEEN 1 AND 65535) OR
        (assignment_mode = 'range' AND range_start BETWEEN 1 AND 65535 AND range_end BETWEEN 1 AND 65535 AND range_start <= range_end) OR
        (assignment_mode = 'random' AND range_start BETWEEN 1 AND 65535 AND range_end BETWEEN 1 AND 65535 AND range_start <= range_end)
    )
);

-- Port pool tracking
-- Manages auto-assignment pools for each core type
CREATE TABLE port_pools (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    core_type VARCHAR(20) NOT NULL,
    start_port INTEGER NOT NULL,
    end_port INTEGER NOT NULL,
    block_size INTEGER DEFAULT 10,
    next_port INTEGER NOT NULL,
    available_ports JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_pool_range CHECK (start_port BETWEEN 1 AND 65535 AND end_port BETWEEN 1 AND 65535 AND start_port <= end_port),
    CONSTRAINT valid_next_port CHECK (next_port BETWEEN start_port AND end_port)
);

-- Port assignments (links inbounds to ports)
-- Tracks which ports are currently assigned to which inbounds
CREATE TABLE port_assignments (
    id BIGSERIAL PRIMARY KEY,
    inbound_id INTEGER REFERENCES inbounds(id) ON DELETE CASCADE,
    port_config_id INTEGER REFERENCES port_configs(id),
    assigned_port INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    assigned_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(inbound_id, port_config_id),
    UNIQUE(assigned_port)
);

-- Port conflict cache for performance
-- Caches validation results to speed up repeated checks
CREATE TABLE port_conflict_cache (
    port INTEGER,
    listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    protocol VARCHAR(20),
    transport VARCHAR(50),
    core_type VARCHAR(20),
    is_available BOOLEAN,
    conflict_reason VARCHAR(255),
    checked_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    PRIMARY KEY (port, listen_address, protocol, transport, core_type)
);
```

### 6.2 HAProxy Domain and Configuration Tables

```sql
-- Domains table for SNI routing
-- Stores all domains that HAProxy routes to backends
CREATE TABLE domains (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('xray', 'singbox', 'singbox_quic', 'mihomo', 'default')),
    core_id INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    is_cdn BOOLEAN DEFAULT FALSE,
    path_prefix VARCHAR(255) DEFAULT '',
    enable_proxy_protocol BOOLEAN DEFAULT TRUE,
    tls_terminate BOOLEAN DEFAULT FALSE,
    cert_path TEXT DEFAULT '',
    key_path TEXT DEFAULT '',
    port_config_id INTEGER REFERENCES port_configs(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Backend configurations
-- Defines how HAProxy connects to proxy cores
CREATE TABLE backend_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL,
    server_address VARCHAR(255) DEFAULT '127.0.0.1',
    backend_port INTEGER NOT NULL,
    port_config_id INTEGER REFERENCES port_configs(id),
    use_unix_socket BOOLEAN DEFAULT FALSE,
    socket_path VARCHAR(255),
    max_connections INTEGER DEFAULT 10000,
    check_interval INTEGER DEFAULT 5, -- seconds
    send_proxy_protocol BOOLEAN DEFAULT TRUE,
    proxy_protocol_version INTEGER DEFAULT 2,
    health_check_enabled BOOLEAN DEFAULT TRUE,
    health_check_type VARCHAR(20) DEFAULT 'tcp',
    health_check_path VARCHAR(255),
    health_check_interval INTEGER DEFAULT 5,
    health_check_rise_count INTEGER DEFAULT 2,
    health_check_fall_count INTEGER DEFAULT 3,
    weight INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Frontend configurations
-- Defines how HAProxy accepts incoming connections
CREATE TABLE frontend_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    bind_address VARCHAR(255) DEFAULT '*',
    bind_port INTEGER NOT NULL,
    port_config_id INTEGER REFERENCES port_configs(id),
    mode VARCHAR(10) NOT NULL CHECK (mode IN ('tcp', 'http')),
    tls_enabled BOOLEAN DEFAULT TRUE,
    tls_cert_path TEXT,
    tls_key_path TEXT,
    alpn JSONB DEFAULT '["h2", "http/1.1"]',
    allow_http3 BOOLEAN DEFAULT FALSE,
    sni_inspection_delay INTEGER DEFAULT 5, -- seconds
    rate_limit_enabled BOOLEAN DEFAULT TRUE,
    rate_limit_requests INTEGER DEFAULT 100,
    rate_limit_window INTEGER DEFAULT 10, -- seconds
    default_backend VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Frontend-Domain many-to-many relationship
CREATE TABLE frontend_domains (
    frontend_id INTEGER REFERENCES frontend_configs(id) ON DELETE CASCADE,
    domain_id INTEGER REFERENCES domains(id) ON DELETE CASCADE,
    PRIMARY KEY (frontend_id, domain_id)
);

-- Global HAProxy configuration
CREATE TABLE global_configs (
    id BIGSERIAL PRIMARY KEY,
    max_connections INTEGER DEFAULT 10000,
    max_ssl_connections INTEGER DEFAULT 10000,
    spread_checks INTEGER DEFAULT 5,
    stats_socket VARCHAR(255) DEFAULT '/var/run/haproxy.sock',
    stats_timeout INTEGER DEFAULT 30, -- seconds
    nbthread INTEGER DEFAULT 4,
    nbproc INTEGER DEFAULT 0,
    log_target VARCHAR(255) DEFAULT 'stdout',
    log_facility VARCHAR(50) DEFAULT 'local0',
    log_level VARCHAR(50) DEFAULT 'info',
    chroot VARCHAR(255),
    user VARCHAR(50),
    group_name VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Defaults configuration
CREATE TABLE defaults_configs (
    id BIGSERIAL PRIMARY KEY,
    mode VARCHAR(10) DEFAULT 'tcp',
    timeout_connect INTEGER DEFAULT 5, -- seconds
    timeout_client INTEGER DEFAULT 50, -- seconds
    timeout_server INTEGER DEFAULT 50, -- seconds
    timeout_tunnel INTEGER DEFAULT 3600, -- seconds (1 hour for WebSocket)
    options JSONB DEFAULT '["tcplog", "dontlognull"]',
    default_backend VARCHAR(100),
    retries INTEGER DEFAULT 3,
    max_conn INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Listen configurations (stats, etc.)
CREATE TABLE listen_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    bind VARCHAR(255) NOT NULL,
    port_config_id INTEGER REFERENCES port_configs(id),
    mode VARCHAR(10) NOT NULL,
    stats_enabled BOOLEAN DEFAULT FALSE,
    stats_uri VARCHAR(255) DEFAULT '/stats',
    stats_refresh INTEGER DEFAULT 10, -- seconds
    prometheus_enabled BOOLEAN DEFAULT TRUE,
    prometheus_path VARCHAR(255) DEFAULT '/metrics',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 6.3 Port Group and Binding Tables

```sql
-- Port groups for organizing related ports
CREATE TABLE port_groups (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    start_port INTEGER NOT NULL,
    end_port INTEGER NOT NULL,
    mode VARCHAR(10) NOT NULL CHECK (mode IN ('tcp', 'http')),
    tls_enabled BOOLEAN DEFAULT TRUE,
    tls_cert_path TEXT,
    port_config_id INTEGER REFERENCES port_configs(id),
    sni_routing BOOLEAN DEFAULT TRUE,
    path_routing BOOLEAN DEFAULT TRUE,
    ddos_protection BOOLEAN DEFAULT TRUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Inbound port bindings
-- Links inbounds to ports with routing rules
CREATE TABLE inbound_port_bindings (
    id BIGSERIAL PRIMARY KEY,
    inbound_id INTEGER REFERENCES inbounds(id) ON DELETE CASCADE,
    port_group_id VARCHAR(50) REFERENCES port_groups(id),
    port_config_id INTEGER REFERENCES port_configs(id),
    listen_port INTEGER NOT NULL,
    backend_port INTEGER NOT NULL,
    sni_match VARCHAR(255),
    path_match VARCHAR(255),
    priority INTEGER DEFAULT 100,
    core_type VARCHAR(20) NOT NULL,
    use_proxy_protocol BOOLEAN DEFAULT FALSE,
    health_check_path VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(inbound_id, port_group_id, listen_port)
);

-- Indexes for performance
CREATE INDEX idx_domains_type ON domains(type);
CREATE INDEX idx_domains_core_id ON domains(core_id);
CREATE INDEX idx_domains_sni ON domains(name) WHERE is_active = TRUE;
CREATE INDEX idx_backend_configs_type ON backend_configs(type);
CREATE INDEX idx_backend_configs_port ON backend_configs(backend_port);
CREATE INDEX idx_frontend_configs_port ON frontend_configs(bind_port);
CREATE INDEX idx_port_configs_core_type ON port_configs(core_type);
CREATE INDEX idx_port_configs_assignment_mode ON port_configs(assignment_mode);
CREATE INDEX idx_port_pools_core_type ON port_pools(core_type);
CREATE INDEX idx_port_assignments_port ON port_assignments(assigned_port);
CREATE INDEX idx_port_assignments_inbound ON port_assignments(inbound_id);
CREATE INDEX idx_inbound_bindings_inbound ON inbound_port_bindings(inbound_id);
CREATE INDEX idx_inbound_bindings_port ON inbound_port_bindings(listen_port);
CREATE INDEX idx_inbound_bindings_group ON inbound_port_bindings(port_group_id);
CREATE INDEX idx_port_conflict_cache_port ON port_conflict_cache(port);
CREATE INDEX idx_port_conflict_cache_expires ON port_conflict_cache(expires_at) WHERE expires_at IS NOT NULL;
```

### 6.4 Migration from Fixed Ports

```sql
-- Migration: Convert fixed ports to dynamic configuration
-- This script migrates existing inbounds with fixed ports to the new system

-- Step 1: Create port configs for existing fixed ports
INSERT INTO port_configs (name, assignment_mode, port, core_type, protocol)
SELECT 
    'migrated-' || core_type || '-' || id,
    'manual',
    port,
    core_type,
    'tcp'
FROM inbounds 
WHERE port IS NOT NULL 
  AND port_config_id IS NULL;

-- Step 2: Update inbounds with port_config_id
UPDATE inbounds i
SET 
    port_config_id = pc.id,
    backend_port = i.port
FROM port_configs pc
WHERE pc.name = 'migrated-' || i.core_type || '-' || i.id
  AND i.port_config_id IS NULL;

-- Step 3: Create default port pools for each core type
INSERT INTO port_pools (name, core_type, start_port, end_port, next_port)
VALUES 
    ('xray-pool', 'xray', 10000, 11000, 10000),
    ('singbox-pool', 'singbox', 11000, 12000, 11000),
    ('mihomo-pool', 'mihomo', 12000, 13000, 12000)
ON CONFLICT (name) DO NOTHING;

-- Step 4: Create default global port config
INSERT INTO global_configs (max_connections, max_ssl_connections, nbthread)
VALUES (10000, 10000, 4);

-- Step 5: Create default frontend configs with dynamic ports
INSERT INTO frontend_configs (name, bind_address, bind_port, mode, tls_enabled, default_backend)
VALUES 
    ('https-main', '*', 443, 'tcp', TRUE, 'default_backend'),
    ('https-alt', '*', 8443, 'tcp', TRUE, 'default_backend'),
    ('http-fallback', '*', 8080, 'http', FALSE, 'default_backend')
ON CONFLICT (name) DO NOTHING;

-- Step 6: Create default backend configs with dynamic ports
INSERT INTO backend_configs (name, type, server_address, backend_port, send_proxy_protocol)
VALUES 
    ('xray_backend', 'xray', '127.0.0.1', 10001, TRUE),
    ('singbox_backend', 'singbox', '127.0.0.1', 9090, FALSE),
    ('mihomo_backend', 'mihomo', '127.0.0.1', 9091, FALSE),
    ('default_backend', 'default', '127.0.0.1', 8080, FALSE)
ON CONFLICT (name) DO NOTHING;
```

---

## Summary

This document defines **15+ core Go structs** with dynamic port configuration:

### Port Configuration (6 structs)
- `PortConfig` - Base port configuration
- `FrontendPortConfig` - Public-facing ports
- `BackendPortConfig` - Internal core communication
- `DirectPortConfig` - UDP/QUIC bypass ports
- `GlobalPortConfig` - Central port management
- `PortPoolConfig` - Auto-assignment pools

### Domain Models (3 structs)
- `Domain` - SNI routing definitions
- `BackendConfig` - HAProxy backend with dynamic `BackendPort`
- `FrontendConfig` - HAProxy frontend with dynamic `BindPort`

### HAProxy Configuration (5 structs)
- `HealthCheckConfig` - Backend health checks
- `GlobalConfig` - Global HAProxy settings
- `DefaultsConfig` - Default parameters
- `ListenConfig` - Stats and monitoring with dynamic `Bind`
- `FullConfig` - Complete configuration root

### Port Management (4 structs)
- `PortGroup` - Port organization
- `InboundPortBinding` - Inbound-to-port mapping with dynamic `ListenPort` and `BackendPort`
- `PortAssignment` - Port lifecycle tracking
- `PortPool` - Auto-assignment management

### Validation (3 structs)
- `PortConflictCheck` - Validation results
- `PortConflict` - Individual conflict details
- `PortValidationRequest/Response` - API types

### Key Dynamic Port Replacements

| Fixed Port | Dynamic Field | Location |
|------------|----------------|----------|
| 10001-10010 (Xray) | `BackendPort` | `BackendConfig`, `InboundPortBinding` |
| 9090-9099 (Sing-box) | `BackendPort` | `BackendConfig`, `InboundPortBinding` |
| 9091-9099 (Mihomo) | `BackendPort` | `BackendConfig`, `InboundPortBinding` |
| 443 (Main HTTPS) | `BindPort` | `FrontendConfig`, `FrontendPortConfig` |
| 8443 (Alt HTTPS) | `BindPort` | `FrontendConfig`, `FrontendPortConfig` |
| 8080 (HTTP) | `BindPort` | `FrontendConfig`, `FrontendPortConfig` |
| 8404 (Stats) | `Bind`, `StatsPort` | `ListenConfig`, `GlobalPortConfig` |
| 8444-8447 (Direct UDP) | `Port` | `DirectPortConfig` |

All structs preserve existing fields, comments, and JSON tags while replacing fixed port references with dynamic `PortConfig`-based fields.
