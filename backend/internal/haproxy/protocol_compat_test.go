package haproxy

import (
	"testing"
)

func TestIsUDPTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		want      bool
	}{
		// UDP transports - should return true
		{"quic lowercase", "quic", true},
		{"QUIC uppercase", "QUIC", true},
		{"Quic mixed case", "Quic", true},
		{"kcp lowercase", "kcp", true},
		{"KCP uppercase", "KCP", true},
		{"hysteria lowercase", "hysteria", true},
		{"HYSTERIA uppercase", "HYSTERIA", true},
		{"hysteria2 lowercase", "hysteria2", true},
		{"HYSTERIA2 uppercase", "HYSTERIA2", true},
		{"tuic lowercase", "tuic", true},
		{"TUIC uppercase", "TUIC", true},

		// TCP transports - should return false
		{"tcp", "tcp", false},
		{"TCP uppercase", "TCP", false},
		{"websocket", "websocket", false},
		{"ws", "ws", false},
		{"grpc", "grpc", false},
		{"httpupgrade", "httpupgrade", false},
		{"xhttp", "xhttp", false},

		// Edge cases
		{"empty string", "", false},
		{"unknown transport", "unknown", false},
		{"random string", "random123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUDPTransport(tt.transport)
			if got != tt.want {
				t.Errorf("IsUDPTransport(%q) = %v, want %v", tt.transport, got, tt.want)
			}
		})
	}
}

func TestSupportsSNI(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     bool
	}{
		// SNI-capable protocols - should return true
		{"vless lowercase", "vless", true},
		{"VLESS uppercase", "VLESS", true},
		{"Vless mixed case", "Vless", true},
		{"vmess lowercase", "vmess", true},
		{"VMESS uppercase", "VMESS", true},
		{"trojan lowercase", "trojan", true},
		{"TROJAN uppercase", "TROJAN", true},
		{"shadowtls lowercase", "shadowtls", true},
		{"SHADOWTLS uppercase", "SHADOWTLS", true},
		{"anytls lowercase", "anytls", true},
		{"ANYTLS uppercase", "ANYTLS", true},
		{"naive lowercase", "naive", true},
		{"NAIVE uppercase", "NAIVE", true},

		// Non-SNI protocols - should return false
		{"socks", "socks", false},
		{"SOCKS uppercase", "SOCKS", false},
		{"http", "http", false},
		{"https", "https", false},
		{"shadowsocks", "shadowsocks", false},

		// Edge cases
		{"empty string", "", false},
		{"unknown protocol", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportsSNI(tt.protocol)
			if got != tt.want {
				t.Errorf("SupportsSNI(%q) = %v, want %v", tt.protocol, got, tt.want)
			}
		})
	}
}

func TestSupportsPath(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		want      bool
	}{
		// Path-capable transports - should return true
		{"websocket lowercase", "websocket", true},
		{"WEBSOCKET uppercase", "WEBSOCKET", true},
		{"WebSocket mixed case", "WebSocket", true},
		{"ws lowercase", "ws", true},
		{"WS uppercase", "WS", true},
		{"httpupgrade lowercase", "httpupgrade", true},
		{"HTTPUPGRADE uppercase", "HTTPUPGRADE", true},
		{"xhttp lowercase", "xhttp", true},
		{"XHTTP uppercase", "XHTTP", true},
		{"grpc lowercase", "grpc", true},
		{"GRPC uppercase", "GRPC", true},

		// Non-path transports - should return false
		{"tcp", "tcp", false},
		{"TCP uppercase", "TCP", false},
		{"kcp", "kcp", false},
		{"quic", "quic", false},
		{"hysteria", "hysteria", false},

		// Edge cases
		{"empty string", "", false},
		{"unknown transport", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportsPath(tt.transport)
			if got != tt.want {
				t.Errorf("SupportsPath(%q) = %v, want %v", tt.transport, got, tt.want)
			}
		})
	}
}

func TestIsHaproxyCompatible(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		transport string
		coreType  string
		want      bool
	}{
		// TCP-based protocols - should return true
		{"vless over tcp", "vless", "tcp", "xray", true},
		{"vmess over tcp", "vmess", "tcp", "xray", true},
		{"trojan over tcp", "trojan", "tcp", "xray", true},
		{"vless over websocket", "vless", "websocket", "xray", true},
		{"vmess over ws", "vmess", "ws", "xray", true},
		{"trojan over grpc", "trojan", "grpc", "xray", true},
		{"any protocol over tcp", "", "tcp", "xray", true},
		{"any protocol over ws", "", "ws", "sing-box", true},

		// UDP-based transports - should return false
		{"vless over quic", "vless", "quic", "xray", false},
		{"vmess over kcp", "vmess", "kcp", "xray", false},
		{"trojan over hysteria", "trojan", "hysteria", "sing-box", false},
		{"vless over hysteria2", "vless", "hysteria2", "xray", false},
		{"vmess over tuic", "vmess", "tuic", "sing-box", false},

		// Case insensitivity tests
		{"vless over QUIC uppercase", "vless", "QUIC", "xray", false},
		{"vless over TCP uppercase", "vless", "TCP", "xray", true},

		// Edge cases
		{"empty protocol over tcp", "", "tcp", "xray", true},
		{"empty transport", "vless", "", "xray", true},
		{"both empty", "", "", "xray", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHaproxyCompatible(tt.protocol, tt.transport, tt.coreType)
			if got != tt.want {
				t.Errorf("IsHaproxyCompatible(%q, %q, %q) = %v, want %v",
					tt.protocol, tt.transport, tt.coreType, got, tt.want)
			}
		})
	}
}

func TestGetSharingMechanism(t *testing.T) {
	tests := []struct {
		name       string
		protocol1  string
		transport1 string
		protocol2  string
		transport2 string
		want       string
	}{
		// Both support SNI - should return "sni"
		{"vless and trojan both tcp", "vless", "tcp", "trojan", "tcp", "sni"},
		{"vmess and vless both tcp", "vmess", "tcp", "vless", "tcp", "sni"},
		{"shadowtls and anytls", "shadowtls", "tcp", "anytls", "tcp", "sni"},
		{"naive and trojan", "naive", "tcp", "trojan", "tcp", "sni"},

		// Both support Path (but also both support SNI, so SNI takes precedence)
		{"vless and vmess both ws - sni takes precedence", "vless", "ws", "vmess", "ws", "sni"},
		{"vless and trojan both websocket - sni takes precedence", "vless", "websocket", "trojan", "websocket", "sni"},
		{"vmess and trojan both grpc - sni takes precedence", "vmess", "grpc", "trojan", "grpc", "sni"},
		{"vless and vmess both httpupgrade - sni takes precedence", "vless", "httpupgrade", "vmess", "httpupgrade", "sni"},
		{"vless and vmess both xhttp - sni takes precedence", "vless", "xhttp", "vmess", "xhttp", "sni"},

		// SNI takes precedence over Path
		{"vless tcp vs vmess ws", "vless", "tcp", "vmess", "ws", "sni"},
		{"trojan tcp vs vless websocket", "trojan", "tcp", "vless", "websocket", "sni"},

		// Neither supports SNI or Path - should return ""
		{"socks and http both tcp", "socks", "tcp", "http", "tcp", ""},
		{"shadowsocks and socks", "shadowsocks", "tcp", "socks", "tcp", ""},
		{"unknown protocols", "unknown1", "tcp", "unknown2", "tcp", ""},

		// One supports SNI, other doesn't - should check Path
		{"vless tcp vs socks tcp", "vless", "tcp", "socks", "tcp", ""},

		// Case insensitivity
		{"VLESS and TROJAN uppercase", "VLESS", "TCP", "TROJAN", "TCP", "sni"},
		{"VLESS and VMESS both WS uppercase - sni takes precedence", "VLESS", "WS", "VMESS", "WS", "sni"},

		// Edge cases with empty strings
		{"empty protocol1", "", "tcp", "vless", "tcp", ""},
		{"empty protocol2", "vless", "tcp", "", "tcp", ""},
		{"both protocols empty", "", "tcp", "", "tcp", ""},
		{"empty transport1", "vless", "", "vmess", "ws", "sni"},
		{"empty transport2 - sni takes precedence", "vless", "ws", "vmess", "", "sni"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSharingMechanism(tt.protocol1, tt.transport1, tt.protocol2, tt.transport2)
			if got != tt.want {
				t.Errorf("GetSharingMechanism(%q, %q, %q, %q) = %q, want %q",
					tt.protocol1, tt.transport1, tt.protocol2, tt.transport2, got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkIsUDPTransport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsUDPTransport("quic")
		IsUDPTransport("tcp")
		IsUDPTransport("kcp")
	}
}

func BenchmarkSupportsSNI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SupportsSNI("vless")
		SupportsSNI("socks")
		SupportsSNI("trojan")
	}
}

func BenchmarkSupportsPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SupportsPath("websocket")
		SupportsPath("tcp")
		SupportsPath("grpc")
	}
}

func BenchmarkIsHaproxyCompatible(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsHaproxyCompatible("vless", "tcp", "xray")
		IsHaproxyCompatible("vless", "quic", "xray")
		IsHaproxyCompatible("trojan", "ws", "sing-box")
	}
}

func BenchmarkGetSharingMechanism(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetSharingMechanism("vless", "tcp", "trojan", "tcp")
		GetSharingMechanism("vless", "ws", "vmess", "ws")
		GetSharingMechanism("socks", "tcp", "http", "tcp")
	}
}
