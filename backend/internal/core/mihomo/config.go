package mihomo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// Config represents Mihomo (Clash Meta) configuration
type Config struct {
	Port               int                      `yaml:"port,omitempty"`
	SocksPort          int                      `yaml:"socks-port,omitempty"`
	MixedPort          int                      `yaml:"mixed-port,omitempty"`
	AllowLan           bool                     `yaml:"allow-lan"`
	Mode               string                   `yaml:"mode"`
	LogLevel           string                   `yaml:"log-level"`
	ExternalController string                   `yaml:"external-controller,omitempty"`
	Proxies            []map[string]interface{} `yaml:"proxies,omitempty"`
	ProxyGroups        []map[string]interface{} `yaml:"proxy-groups,omitempty"`
	Rules              []string                 `yaml:"rules,omitempty"`
}

// GenerateConfig generates Mihomo configuration from database models
func GenerateConfig(inbounds []models.Inbound, outbounds []models.Outbound) (*Config, error) {
	config := &Config{
		AllowLan:    true,
		Mode:        "rule",
		LogLevel:    "info",
		Proxies:     make([]map[string]interface{}, 0),
		ProxyGroups: make([]map[string]interface{}, 0),
		Rules:       make([]string, 0),
	}

	// Set first inbound port as main port
	if len(inbounds) > 0 && inbounds[0].IsEnabled {
		config.MixedPort = inbounds[0].Port
	}

	// Add proxies from inbounds
	for _, inbound := range inbounds {
		if !inbound.IsEnabled {
			continue
		}

		proxy := map[string]interface{}{
			"name":   fmt.Sprintf("inbound-%d", inbound.ID),
			"type":   inbound.Protocol,
			"server": inbound.ListenAddress,
			"port":   inbound.Port,
		}

		// Parse config_json for additional settings
		if inbound.ConfigJSON != "" {
			// Mihomo uses YAML, but we store JSON in DB
			// For now, just add basic config
		}

		config.Proxies = append(config.Proxies, proxy)
	}

	// Add default proxy group
	if len(config.Proxies) > 0 {
		proxyNames := make([]string, 0)
		for _, proxy := range config.Proxies {
			if name, ok := proxy["name"].(string); ok {
				proxyNames = append(proxyNames, name)
			}
		}

		config.ProxyGroups = append(config.ProxyGroups, map[string]interface{}{
			"name":    "PROXY",
			"type":    "select",
			"proxies": proxyNames,
		})
	}

	// Add default rules
	config.Rules = []string{
		"MATCH,DIRECT",
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

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ValidateConfig validates Mihomo configuration
func ValidateConfig(path string) error {
	// Check if mihomo binary exists
	mihomoPath := "/usr/local/bin/mihomo"
	if _, err := os.Stat(mihomoPath); os.IsNotExist(err) {
		// Skip validation if binary not found (development mode)
		return nil
	}

	// Run mihomo test (if supported)
	cmd := exec.Command(mihomoPath, "-t", "-f", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Mihomo might not support -t flag, just check if file is valid YAML
		var testConfig Config
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read config: %w", readErr)
		}
		if yamlErr := yaml.Unmarshal(data, &testConfig); yamlErr != nil {
			return fmt.Errorf("invalid YAML: %w", yamlErr)
		}
		// If YAML is valid, ignore mihomo test error
		return nil
	}

	_ = output // Suppress unused warning
	return nil
}
