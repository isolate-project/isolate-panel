// Package haproxy provides protocol transport compatibility utilities for HAProxy.
//
// This package determines which protocols can share ports via HAProxy vs need direct ports.
// It supports Wave 1 of HAProxy implementation with the following capabilities:
//
//   - UDP Transport Detection: Identifies protocols that cannot go through HAProxy
//   - SNI Support Detection: Identifies protocols that support SNI-based routing
//   - Path Support Detection: Identifies transports that support Path-based routing
//   - HAProxy Compatibility: Determines if a protocol/transport combination works with HAProxy
//   - Sharing Mechanism: Determines how two protocols can share a port
//
// Protocol Matrix:
//   - UDP transports (cannot use HAProxy): quic, kcp, hysteria, hysteria2, tuic
//   - SNI protocols: vless, vmess, trojan, shadowtls, anytls, naive
//   - Path transports: websocket, ws, httpupgrade, xhttp, grpc
//
// PROXY Protocol v2 Support:
//   - Only Xray core with TCP-based protocols supports PROXY v2
//   - UDP transports and other cores do not support PROXY v2
package haproxy

import "strings"

// udpTransports contains all UDP-based transports that cannot go through HAProxy.
var udpTransports = map[string]bool{
	"quic":      true,
	"kcp":       true,
	"hysteria":  true,
	"hysteria2": true,
	"tuic":      true,
}

// sniProtocols contains all protocols that support SNI-based routing.
var sniProtocols = map[string]bool{
	"vless":     true,
	"vmess":     true,
	"trojan":    true,
	"shadowtls": true,
	"anytls":    true,
	"naive":     true,
}

// pathTransports contains all transports that support Path-based routing.
var pathTransports = map[string]bool{
	"websocket":   true,
	"ws":          true,
	"httpupgrade": true,
	"xhttp":       true,
	"grpc":        true,
}

// IsUDPTransport returns true if the transport is UDP-based and cannot go through HAProxy.
//
// UDP transports identified: "quic", "kcp", "hysteria", "hysteria2", "tuic"
// These protocols require direct port access and cannot be proxied through HAProxy.
//
// Examples:
//   - IsUDPTransport("quic") returns true
//   - IsUDPTransport("tcp") returns false
//   - IsUDPTransport("") returns false
func IsUDPTransport(transport string) bool {
	if transport == "" {
		return false
	}
	return udpTransports[strings.ToLower(transport)]
}

// SupportsSNI returns true if the protocol supports SNI-based routing.
//
// SNI-capable protocols: "vless", "vmess", "trojan", "shadowtls", "anytls", "naive"
// These protocols can be routed based on the Server Name Indication in TLS handshake.
//
// Examples:
//   - SupportsSNI("vless") returns true
//   - SupportsSNI("socks") returns false
//   - SupportsSNI("") returns false
func SupportsSNI(protocol string) bool {
	if protocol == "" {
		return false
	}
	return sniProtocols[strings.ToLower(protocol)]
}

// SupportsPath returns true if the transport supports Path-based routing.
//
// Path-capable transports: "websocket", "ws", "httpupgrade", "xhttp", "grpc"
// These transports can be routed based on HTTP path or WebSocket path.
//
// Examples:
//   - SupportsPath("websocket") returns true
//   - SupportsPath("ws") returns true
//   - SupportsPath("tcp") returns false
//   - SupportsPath("") returns false
func SupportsPath(transport string) bool {
	if transport == "" {
		return false
	}
	return pathTransports[strings.ToLower(transport)]
}

// IsHaproxyCompatible returns true if the protocol/transport combination can use HAProxy.
//
// A combination is compatible if:
//   - The transport is not UDP-based (IsUDPTransport returns false)
//
// All TCP-based protocols work with HAProxy regardless of core type.
// UDP transports require direct port access.
//
// The coreType parameter is reserved for future use (e.g., PROXY v2 support detection).
// Currently, only Xray core with TCP-based protocols supports PROXY v2.
//
// Examples:
//   - IsHaproxyCompatible("vless", "tcp", "xray") returns true
//   - IsHaproxyCompatible("vless", "quic", "xray") returns false
//   - IsHaproxyCompatible("", "tcp", "xray") returns true
func IsHaproxyCompatible(protocol, transport, coreType string) bool {
	// UDP transports cannot go through HAProxy
	if IsUDPTransport(transport) {
		return false
	}

	// All TCP-based protocols are compatible with HAProxy
	// coreType is reserved for future PROXY v2 support detection
	_ = coreType // explicitly mark as used for documentation purposes

	return true
}

// GetSharingMechanism determines how two protocols can share a port.
//
// Returns:
//   - "sni" if both protocols support SNI-based routing
//   - "path" if both transports support Path-based routing
//   - "" (empty string) if neither mechanism is available
//
// Priority: SNI takes precedence over Path when both are available.
//
// Examples:
//   - GetSharingMechanism("vless", "tcp", "trojan", "tcp") returns "sni"
//   - GetSharingMechanism("vless", "ws", "vmess", "ws") returns "path"
//   - GetSharingMechanism("vless", "tcp", "vmess", "ws") returns "sni"
//   - GetSharingMechanism("socks", "tcp", "http", "tcp") returns ""
func GetSharingMechanism(protocol1, transport1, protocol2, transport2 string) string {
	// Check SNI support first (higher priority)
	if SupportsSNI(protocol1) && SupportsSNI(protocol2) {
		return "sni"
	}

	// Check Path support
	if SupportsPath(transport1) && SupportsPath(transport2) {
		return "path"
	}

	// No sharing mechanism available
	return ""
}
