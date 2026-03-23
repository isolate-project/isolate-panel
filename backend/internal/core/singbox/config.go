package singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// Config represents Sing-box configuration
type Config struct {
	Log       *LogConfig       `json:"log,omitempty"`
	DNS       *DNSConfig       `json:"dns,omitempty"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Route     *RouteConfig     `json:"route,omitempty"`
}

type LogConfig struct {
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

type DNSConfig struct {
	Servers []DNSServer `json:"servers"`
}

type DNSServer struct {
	Tag     string `json:"tag"`
	Address string `json:"address"`
}

type InboundConfig struct {
	Type   string                   `json:"type"`
	Tag    string                   `json:"tag"`
	Listen string                   `json:"listen"`
	Port   int                      `json:"listen_port"`
	Users  []map[string]interface{} `json:"users,omitempty"`
	TLS    *TLSConfig               `json:"tls,omitempty"`
}

type TLSConfig struct {
	Enabled    bool   `json:"enabled"`
	ServerName string `json:"server_name,omitempty"`
	CertPath   string `json:"certificate_path,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
}

type OutboundConfig struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
}

type RouteConfig struct {
	Rules []RouteRule `json:"rules,omitempty"`
}

type RouteRule struct {
	Inbound  []string `json:"inbound,omitempty"`
	Outbound string   `json:"outbound"`
}

// GenerateConfig generates Sing-box configuration from database models
func GenerateConfig(inbounds []models.Inbound, outbounds []models.Outbound) (*Config, error) {
	config := &Config{
		Log: &LogConfig{
			Level:     "info",
			Timestamp: true,
		},
		DNS: &DNSConfig{
			Servers: []DNSServer{
				{Tag: "google", Address: "8.8.8.8"},
				{Tag: "cloudflare", Address: "1.1.1.1"},
			},
		},
		Inbounds:  make([]InboundConfig, 0),
		Outbounds: make([]OutboundConfig, 0),
	}

	// Add inbounds
	for _, inbound := range inbounds {
		if !inbound.IsEnabled {
			continue
		}

		inboundConfig := InboundConfig{
			Type:   inbound.Protocol,
			Tag:    fmt.Sprintf("inbound-%d", inbound.ID),
			Listen: inbound.ListenAddress,
			Port:   inbound.Port,
		}

		// Parse config_json for additional settings
		if inbound.ConfigJSON != "" {
			var additionalConfig map[string]interface{}
			if err := json.Unmarshal([]byte(inbound.ConfigJSON), &additionalConfig); err == nil {
				// Merge additional config (simplified)
				if users, ok := additionalConfig["users"].([]interface{}); ok {
					inboundConfig.Users = make([]map[string]interface{}, len(users))
					for i, u := range users {
						if userMap, ok := u.(map[string]interface{}); ok {
							inboundConfig.Users[i] = userMap
						}
					}
				}
			}
		}

		// Add TLS if enabled
		if inbound.TLSEnabled {
			inboundConfig.TLS = &TLSConfig{
				Enabled: true,
			}
		}

		config.Inbounds = append(config.Inbounds, inboundConfig)
	}

	// Add outbounds
	for _, outbound := range outbounds {
		if !outbound.IsEnabled {
			continue
		}

		outboundConfig := OutboundConfig{
			Type: outbound.Protocol,
			Tag:  fmt.Sprintf("outbound-%d", outbound.ID),
		}

		config.Outbounds = append(config.Outbounds, outboundConfig)
	}

	// Add default direct outbound if none exist
	if len(config.Outbounds) == 0 {
		config.Outbounds = append(config.Outbounds, OutboundConfig{
			Type: "direct",
			Tag:  "direct",
		})
	}

	return config, nil
}

// WriteConfig writes configuration to file
func WriteConfig(config *Config, path string) error {
	// Create directory if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ValidateConfig validates Sing-box configuration
func ValidateConfig(path string) error {
	// Check if sing-box binary exists
	singboxPath := "/usr/local/bin/sing-box"
	if _, err := os.Stat(singboxPath); os.IsNotExist(err) {
		// Skip validation if binary not found (development mode)
		return nil
	}

	// Run sing-box check
	cmd := exec.Command(singboxPath, "check", "-c", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	return nil
}
