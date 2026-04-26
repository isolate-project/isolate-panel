package urlvalidator

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateWebhookURL validates a webhook URL to prevent SSRF attacks.
// It ensures scheme is https:// only and hostname does not resolve to private/internal IPs.
func ValidateWebhookURL(webhookURL string) error {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only https:// scheme is allowed, got: %s", parsedURL.Scheme)
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("hostname is empty")
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", hostname, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("hostname %s resolves to private IP %s", hostname, ip)
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is private/internal
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8 (RFC 1918)
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12 (RFC 1918)
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16 (RFC 1918)
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local / cloud metadata)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 127.0.0.0/8 (loopback)
		if ip4[0] == 127 {
			return true
		}
		return false
	}

	if ip.IsLoopback() {
		return true
	}
	// fc00::/7 (IPv6 Unique Local Address - ULA)
	if ip[0] >= 0xfc && ip[0] <= 0xfd {
		return true
	}
	// fe80::/10 (link-local)
	if ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
		return true
	}

	return false
}

// IsPrivateIPString checks if an IP string is private/internal
func IsPrivateIPString(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return isPrivateIP(ip)
}

// ValidateHostname checks if a hostname is safe (doesn't resolve to private IPs)
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is empty")
	}

	lowerHostname := strings.ToLower(hostname)
	if lowerHostname == "localhost" ||
		strings.HasSuffix(lowerHostname, ".localhost") ||
		lowerHostname == "local" ||
		strings.HasSuffix(lowerHostname, ".local") {
		return fmt.Errorf("localhost hostname is not allowed")
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", hostname, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("hostname %s resolves to private IP %s", hostname, ip)
		}
	}

	return nil
}