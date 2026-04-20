package protocol

// init registers all protocol schemas on package load
func init() {
	registerInboundProtocols()
	registerOutboundProtocols()
}

// ============================================================
// Inbound Protocols
// ============================================================

func registerInboundProtocols() {
	// --- Common protocols (multi-core) ---

	Register(&ProtocolSchema{
		Protocol:    "http",
		Label:       "HTTP Proxy",
		Description: "Basic HTTP/HTTPS proxy",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"username": {
				Name:  "username",
				Label: "Username",
				Type:  TypeString,
				Group: "basic",
			},
			"password": {
				Name:  "password",
				Label: "Password",
				Type:  TypeString,
				Group: "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "socks5",
		Label:       "SOCKS5",
		Description: "SOCKS5 proxy with optional authentication",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"username": {
				Name:  "username",
				Label: "Username",
				Type:  TypeString,
				Group: "basic",
			},
			"password": {
				Name:  "password",
				Label: "Password",
				Type:  TypeString,
				Group: "basic",
			},
			"udp": {
				Name:    "udp",
				Label:   "Enable UDP",
				Type:    TypeBoolean,
				Default: true,
				Group:   "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "mixed",
		Label:       "Mixed (HTTP+SOCKS5)",
		Description: "Combined HTTP and SOCKS5 proxy on a single port",
		Core:        []string{"sing-box", "mihomo"},
		Direction:   "inbound",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"username": {
				Name:  "username",
				Label: "Username",
				Type:  TypeString,
				Group: "basic",
			},
			"password": {
				Name:  "password",
				Label: "Password",
				Type:  TypeString,
				Group: "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "shadowsocks",
		Label:       "Shadowsocks",
		Description: "Shadowsocks proxy protocol",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"method": {
				Name:     "method",
				Label:    "Encryption Method",
				Type:     TypeSelect,
				Required: true,
				Default:  "2022-blake3-aes-128-gcm",
				Options: []string{
					"2022-blake3-aes-128-gcm",
					"2022-blake3-aes-256-gcm",
					"2022-blake3-chacha20-poly1305",
					"aes-128-gcm",
					"aes-256-gcm",
					"chacha20-ietf-poly1305",
					"xchacha20-ietf-poly1305",
					"none",
					"plain",
				},
				Group: "basic",
			},
			"password": {
				Name:         "password",
				Label:        "Password / Key",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Description:  "Password or base64 key for 2022 ciphers",
				Group:        "basic",
			},
		},
		Transport: []string{"websocket", "grpc"},
	})

	Register(&ProtocolSchema{
		Protocol:    "vmess",
		Label:       "VMess",
		Description: "V2Ray VMess protocol",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"uuid": {
				Name:         "uuid",
				Label:        "User UUID",
				Type:         TypeUUID,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_uuid_v4",
				Description:  "User UUID for VMess authentication",
				Group:        "basic",
			},
			"alter_id": {
				Name:        "alter_id",
				Label:       "Alter ID",
				Type:        TypeInteger,
				Default:     0,
				Description: "Alter ID (recommended: 0 for VMess AEAD)",
				Min:         intPtr(0),
				Max:         intPtr(65535),
				Group:       "basic",
			},
			"cipher": {
				Name:    "cipher",
				Label:   "Encryption",
				Type:    TypeSelect,
				Default: "auto",
				Options: []string{"auto", "aes-128-gcm", "chacha20-poly1305", "none"},
				Group:   "advanced",
			},
		},
		Transport: []string{"websocket", "grpc", "http", "httpupgrade"},
	})

	Register(&ProtocolSchema{
		Protocol:    "vless",
		Label:       "VLESS",
		Description: "Lightweight V2Ray protocol without encryption overhead",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"uuid": {
				Name:         "uuid",
				Label:        "User UUID",
				Type:         TypeUUID,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_uuid_v4",
				Description:  "User UUID for VLESS authentication",
				Group:        "basic",
			},
			"flow": {
				Name:        "flow",
				Label:       "Flow Control",
				Type:        TypeSelect,
				Default:     "",
				Options:     []string{"", "xtls-rprx-vision"},
				Description: "XTLS flow control (use with REALITY or TLS)",
				Group:       "advanced",
			},
		},
		Transport: []string{"websocket", "grpc", "http", "httpupgrade"},
	})

	Register(&ProtocolSchema{
		Protocol:    "trojan",
		Label:       "Trojan",
		Description: "Trojan protocol (requires TLS)",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: true,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Description:  "Trojan password",
				Group:        "basic",
			},
		},
		Transport: []string{"websocket", "grpc"},
	})

	Register(&ProtocolSchema{
		Protocol:    "hysteria2",
		Label:       "Hysteria 2",
		Description: "QUIC-based protocol optimized for high-latency networks",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "both",
		RequiresTLS: true,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
			"up_mbps": {
				Name:        "up_mbps",
				Label:       "Upload Speed (Mbps)",
				Type:        TypeInteger,
				Default:     100,
				Description: "Upload bandwidth limit in Mbps",
				Min:         intPtr(1),
				Group:       "basic",
			},
			"down_mbps": {
				Name:        "down_mbps",
				Label:       "Download Speed (Mbps)",
				Type:        TypeInteger,
				Default:     100,
				Description: "Download bandwidth limit in Mbps",
				Min:         intPtr(1),
				Group:       "basic",
			},
			"obfs_type": {
				Name:    "obfs_type",
				Label:   "Obfuscation Type",
				Type:    TypeSelect,
				Default: "",
				Options: []string{"", "salamander"},
				Group:   "advanced",
			},
			"obfs_password": {
				Name:         "obfs_password",
				Label:        "Obfuscation Password",
				Type:         TypeString,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				DependsOn: []Dependency{
					{Field: "obfs_type", Value: "salamander", Condition: "equals"},
				},
				Group: "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:          "hysteria",
		Label:             "Hysteria (v1)",
		Description:       "QUIC-based protocol (LEGACY — use Hysteria 2 instead)",
		Core:              []string{"singbox", "mihomo"},
		Direction:         "both",
		RequiresTLS:       true,
		Category:          "tunnel",
		Deprecated:        true,
		DeprecationNotice: "This protocol is legacy. Use Hysteria 2 for better performance and security.",
		Parameters: map[string]Parameter{
			"auth_str": {
				Name:         "auth_str",
				Label:        "Auth String",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
			"up_mbps": {
				Name:    "up_mbps",
				Label:   "Upload Speed (Mbps)",
				Type:    TypeInteger,
				Default: 100,
				Min:     intPtr(1),
				Group:   "basic",
			},
			"down_mbps": {
				Name:    "down_mbps",
				Label:   "Download Speed (Mbps)",
				Type:    TypeInteger,
				Default: 100,
				Min:     intPtr(1),
				Group:   "basic",
			},
			"obfs": {
				Name:    "obfs",
				Label:   "Obfuscation Password",
				Type:    TypeString,
				Group:   "advanced",
			},
			"recv_window_conn": {
				Name:    "recv_window_conn",
				Label:   "QUIC Stream Window",
				Type:    TypeInteger,
				Group:   "advanced",
			},
			"recv_window_client": {
				Name:    "recv_window_conn",
				Label:   "QUIC Connection Window",
				Type:    TypeInteger,
				Group:   "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "tuic_v4",
		Label:       "TUIC v4",
		Description: "TUIC v4 protocol (token-based auth)",
		Core:        []string{"sing-box", "mihomo"},
		Direction:   "both",
		RequiresTLS: true,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"token": {
				Name:         "token",
				Label:        "Token",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_base64_token_32",
				Group:        "basic",
			},
			"congestion_control": {
				Name:    "congestion_control",
				Label:   "Congestion Control",
				Type:    TypeSelect,
				Default: "bbr",
				Options: []string{"cubic", "new_reno", "bbr"},
				Group:   "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "tuic_v5",
		Label:       "TUIC v5",
		Description: "TUIC v5 protocol (UUID + password auth)",
		Core:        []string{"sing-box", "mihomo"},
		Direction:   "both",
		RequiresTLS: true,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"uuid": {
				Name:         "uuid",
				Label:        "UUID",
				Type:         TypeUUID,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_uuid_v4",
				Group:        "basic",
			},
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
			"congestion_control": {
				Name:    "congestion_control",
				Label:   "Congestion Control",
				Type:    TypeSelect,
				Default: "bbr",
				Options: []string{"cubic", "new_reno", "bbr"},
				Group:   "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "naive",
		Label:       "NaiveProxy",
		Description: "NaiveProxy protocol (Sing-box exclusive)",
		Core:        []string{"sing-box"},
		Direction:   "inbound",
		RequiresTLS: true,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"username": {
				Name:         "username",
				Label:        "Username",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_8",
				Group:        "basic",
			},
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "anytls",
		Label:       "AnyTLS",
		Description: "TLS-based proxy protocol with flexible padding (Sing-box exclusive)",
		Core:        []string{"singbox"},
		Direction:   "both",
		RequiresTLS: true,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_base64_token_32",
				Group:        "basic",
			},
			"padding_scheme": {
				Name:        "padding_scheme",
				Label:       "Padding Scheme",
				Type:        TypeArray,
				Description: "Optional custom padding scheme lines",
				Group:       "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "redirect",
		Label:       "Redirect",
		Description: "Transparent redirect proxy",
		Core:        []string{"sing-box", "mihomo"},
		Direction:   "inbound",
		RequiresTLS: false,
		Category:    "utility",
		Parameters:  map[string]Parameter{},
	})

	// --- Xray exclusive ---

	Register(&ProtocolSchema{
		Protocol:    "xhttp",
		Label:       "XHTTP",
		Description: "HTTP-based transport protocol (Xray exclusive)",
		Core:        []string{"xray"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"uuid": {
				Name:         "uuid",
				Label:        "User UUID",
				Type:         TypeUUID,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_uuid_v4",
				Group:        "basic",
			},
			"path": {
				Name:         "path",
				Label:        "Path",
				Type:         TypeString,
				Default:      "/xhttp",
				AutoGenerate: true,
				AutoGenFunc:  "generate_random_path",
				Group:        "basic",
			},
			"mode": {
				Name:    "mode",
				Label:   "Mode",
				Type:    TypeSelect,
				Default: "auto",
				Options: []string{"auto", "packet-up", "stream-up"},
				Group:   "advanced",
			},
		},
	})

	// --- Mihomo exclusive ---

	Register(&ProtocolSchema{
		Protocol:    "mieru",
		Label:       "Mieru",
		Description: "Mieru protocol (Mihomo exclusive)",
		Core:        []string{"mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
			"transport": {
				Name:    "transport",
				Label:   "Transport",
				Type:    TypeSelect,
				Default: "TCP",
				Options: []string{"TCP", "UDP"},
				Group:   "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "sudoku",
		Label:       "Sudoku",
		Description: "Sudoku protocol (Mihomo exclusive)",
		Core:        []string{"mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "trusttunnel",
		Label:       "TrustTunnel",
		Description: "TrustTunnel protocol (Mihomo exclusive)",
		Core:        []string{"mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "shadowsocksr",
		Label:       "ShadowsocksR",
		Description: "ShadowsocksR legacy protocol (Mihomo exclusive)",
		Core:        []string{"mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"cipher": {
				Name:     "cipher",
				Label:    "Cipher",
				Type:     TypeSelect,
				Required: true,
				Default:  "aes-256-cfb",
				Options:  []string{"aes-128-cfb", "aes-256-cfb", "chacha20", "chacha20-ietf", "rc4-md5"},
				Group:    "basic",
			},
			"password": {
				Name:         "password",
				Label:        "Password",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
			"obfs": {
				Name:    "obfs",
				Label:   "Obfuscation",
				Type:    TypeSelect,
				Default: "plain",
				Options: []string{"plain", "http_simple", "http_post", "tls1.2_ticket_auth"},
				Group:   "advanced",
			},
			"protocol": {
				Name:    "protocol",
				Label:   "Protocol",
				Type:    TypeSelect,
				Default: "origin",
				Options: []string{"origin", "auth_sha1_v4", "auth_aes128_md5", "auth_aes128_sha1", "auth_chain_a", "auth_chain_b"},
				Group:   "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "snell",
		Label:       "Snell",
		Description: "Snell protocol (Mihomo exclusive)",
		Core:        []string{"mihomo"},
		Direction:   "both",
		RequiresTLS: false,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"psk": {
				Name:         "psk",
				Label:        "Pre-Shared Key",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_base64_token_32",
				Group:        "basic",
			},
			"version": {
				Name:    "version",
				Label:   "Version",
				Type:    TypeSelect,
				Default: "3",
				Options: []string{"2", "3"},
				Group:   "basic",
			},
			"obfs": {
				Name:    "obfs",
				Label:   "Obfuscation",
				Type:    TypeSelect,
				Default: "",
				Options: []string{"", "tls", "http"},
				Group:   "advanced",
			},
		},
	})
}

// ============================================================
// Outbound-only Protocols
// ============================================================

func registerOutboundProtocols() {
	Register(&ProtocolSchema{
		Protocol:    "direct",
		Label:       "Direct",
		Description: "Direct outbound connection (no proxy)",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "outbound",
		RequiresTLS: false,
		Category:    "utility",
		Parameters:  map[string]Parameter{},
	})

	Register(&ProtocolSchema{
		Protocol:    "block",
		Label:       "Block",
		Description: "Block all traffic (blackhole)",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "outbound",
		RequiresTLS: false,
		Category:    "utility",
		Parameters:  map[string]Parameter{},
	})

	Register(&ProtocolSchema{
		Protocol:    "dns",
		Label:       "DNS",
		Description: "DNS outbound for DNS queries",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "outbound",
		RequiresTLS: false,
		Category:    "utility",
		Parameters:  map[string]Parameter{},
	})

	Register(&ProtocolSchema{
		Protocol:    "hysteria",
		Label:       "Hysteria",
		Description: "Hysteria v1 QUIC-based protocol",
		Core:        []string{"sing-box", "xray", "mihomo"},
		Direction:   "outbound",
		RequiresTLS: true,
		Category:    "tunnel",
		Parameters: map[string]Parameter{
			"auth_str": {
				Name:         "auth_str",
				Label:        "Auth String",
				Type:         TypeString,
				Required:     true,
				AutoGenerate: true,
				AutoGenFunc:  "generate_password_16",
				Group:        "basic",
			},
			"up_mbps": {
				Name:    "up_mbps",
				Label:   "Upload Speed (Mbps)",
				Type:    TypeInteger,
				Default: 100,
				Min:     intPtr(1),
				Group:   "basic",
			},
			"down_mbps": {
				Name:    "down_mbps",
				Label:   "Download Speed (Mbps)",
				Type:    TypeInteger,
				Default: 100,
				Min:     intPtr(1),
				Group:   "basic",
			},
			"obfs": {
				Name:  "obfs",
				Label: "Obfuscation Password",
				Type:  TypeString,
				Group: "advanced",
			},
		},
	})

	Register(&ProtocolSchema{
		Protocol:    "tor",
		Label:       "Tor",
		Description: "Tor network outbound (Sing-box exclusive)",
		Core:        []string{"sing-box"},
		Direction:   "outbound",
		RequiresTLS: false,
		Category:    "tunnel",
		Parameters:  map[string]Parameter{},
	})

	Register(&ProtocolSchema{
		Protocol:    "masque",
		Label:       "MASQUE",
		Description: "MASQUE HTTP/3 proxy (Mihomo exclusive)",
		Core:        []string{"mihomo"},
		Direction:   "outbound",
		RequiresTLS: true,
		Category:    "proxy",
		Parameters: map[string]Parameter{
			"url": {
				Name:     "url",
				Label:    "Server URL",
				Type:     TypeString,
				Required: true,
				Example:  "https://example.com/.well-known/masque/udp/{target_host}/{target_port}/",
				Group:    "basic",
			},
			"username": {
				Name:  "username",
				Label: "Username",
				Type:  TypeString,
				Group: "basic",
			},
			"password": {
				Name:  "password",
				Label: "Password",
				Type:  TypeString,
				Group: "basic",
			},
		},
	})
}
