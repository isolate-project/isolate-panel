package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// Config represents Xray configuration
type Config struct {
	Log       *LogConfig       `json:"log,omitempty"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   *RoutingConfig   `json:"routing,omitempty"`
}

type LogConfig struct {
	LogLevel string `json:"loglevel"`
}

type InboundConfig struct {
	Tag            string                 `json:"tag"`
	Port           int                    `json:"port"`
	Protocol       string                 `json:"protocol"`
	Listen         string                 `json:"listen"`
	Settings       map[string]interface{} `json:"settings,omitempty"`
	StreamSettings *StreamSettings        `json:"streamSettings,omitempty"`
}

type StreamSettings struct {
	Network       string                 `json:"network,omitempty"`
	Security      string                 `json:"security,omitempty"`
	TLSSettings   *TLSSettings           `json:"tlsSettings,omitempty"`
	XHTTPSettings map[string]interface{} `json:"xhttpSettings,omitempty"`
}

type TLSSettings struct {
	Certificates []Certificate `json:"certificates,omitempty"`
	ServerName   string        `json:"serverName,omitempty"`
}

type Certificate struct {
	CertificateFile string `json:"certificateFile"`
	KeyFile         string `json:"keyFile"`
}

type OutboundConfig struct {
	Tag      string                 `json:"tag"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

type RoutingConfig struct {
	Rules []RoutingRule `json:"rules,omitempty"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

// GenerateConfig generates Xray configuration from database models
func GenerateConfig(inbounds []models.Inbound, outbounds []models.Outbound) (*Config, error) {
	config := &Config{
		Log: &LogConfig{
			LogLevel: "info",
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
			Tag:      fmt.Sprintf("inbound-%d", inbound.ID),
			Port:     inbound.Port,
			Protocol: inbound.Protocol,
			Listen:   inbound.ListenAddress,
			Settings: make(map[string]interface{}),
		}

		// Parse config_json for additional settings
		if inbound.ConfigJSON != "" {
			var additionalConfig map[string]interface{}
			if err := json.Unmarshal([]byte(inbound.ConfigJSON), &additionalConfig); err == nil {
				if settings, ok := additionalConfig["settings"].(map[string]interface{}); ok {
					inboundConfig.Settings = settings
				}
			}
		}

		// Add TLS if enabled
		if inbound.TLSEnabled {
			inboundConfig.StreamSettings = &StreamSettings{
				Security:    "tls",
				TLSSettings: &TLSSettings{},
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
			Tag:      fmt.Sprintf("outbound-%d", outbound.ID),
			Protocol: outbound.Protocol,
			Settings: make(map[string]interface{}),
		}

		// Parse config_json
		if outbound.ConfigJSON != "" {
			var additionalConfig map[string]interface{}
			if err := json.Unmarshal([]byte(outbound.ConfigJSON), &additionalConfig); err == nil {
				if settings, ok := additionalConfig["settings"].(map[string]interface{}); ok {
					outboundConfig.Settings = settings
				}
			}
		}

		config.Outbounds = append(config.Outbounds, outboundConfig)
	}

	// Add default freedom outbound if none exist
	if len(config.Outbounds) == 0 {
		config.Outbounds = append(config.Outbounds, OutboundConfig{
			Tag:      "direct",
			Protocol: "freedom",
			Settings: make(map[string]interface{}),
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

// ValidateConfig validates Xray configuration
func ValidateConfig(path string) error {
	// Check if xray binary exists
	xrayPath := "/usr/local/bin/xray"
	if _, err := os.Stat(xrayPath); os.IsNotExist(err) {
		// Skip validation if binary not found (development mode)
		return nil
	}

	// Run xray test
	cmd := exec.Command(xrayPath, "test", "-c", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	return nil
}
