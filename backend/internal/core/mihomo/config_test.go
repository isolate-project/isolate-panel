package mihomo_test

import (
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/core/mihomo"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func TestGenerateMihomoConfig(t *testing.T) {
	tests := []struct {
		name      string
		inbounds  []models.Inbound
		outbounds []models.Outbound
		wantErr   bool
	}{
		{
			name:      "empty config",
			inbounds:  []models.Inbound{},
			outbounds: []models.Outbound{},
			wantErr:   false,
		},
		{
			name: "single inbound",
			inbounds: []models.Inbound{
				{
					ID:            1,
					Name:          "test-inbound",
					Protocol:      "socks",
					Port:          1080,
					ListenAddress: "0.0.0.0",
					IsEnabled:     true,
				},
			},
			outbounds: []models.Outbound{},
			wantErr:   false,
		},
		{
			name: "disabled inbound should be skipped",
			inbounds: []models.Inbound{
				{
					ID:            1,
					Name:          "test-inbound",
					Protocol:      "socks",
					Port:          1080,
					ListenAddress: "0.0.0.0",
					IsEnabled:     false,
				},
			},
			outbounds: []models.Outbound{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := mihomo.GenerateConfig(tt.inbounds, tt.outbounds)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if config == nil {
				t.Error("GenerateConfig() returned nil config")
				return
			}

			// Verify basic structure
			if config.Mode == "" {
				t.Error("Config should have mode set")
			}
			if config.LogLevel == "" {
				t.Error("Config should have log level set")
			}
			if config.Proxies == nil {
				t.Error("Config should have proxies array")
			}

			// Verify disabled inbounds are not included
			enabledCount := 0
			for _, inbound := range tt.inbounds {
				if inbound.IsEnabled {
					enabledCount++
				}
			}
			if len(config.Proxies) != enabledCount {
				t.Errorf("Expected %d enabled inbounds, got %d", enabledCount, len(config.Proxies))
			}
		})
	}
}

func TestMihomoConfigStructure(t *testing.T) {
	inbounds := []models.Inbound{
		{
			ID:            1,
			Name:          "socks-inbound",
			Protocol:      "socks",
			Port:          1080,
			ListenAddress: "0.0.0.0",
			IsEnabled:     true,
		},
	}

	config, err := mihomo.GenerateConfig(inbounds, []models.Outbound{})
	if err != nil {
		t.Fatalf("GenerateConfig() failed: %v", err)
	}

	// Verify mode
	if config.Mode != "rule" {
		t.Errorf("Expected mode 'rule', got '%s'", config.Mode)
	}

	// Verify log level
	if config.LogLevel != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.LogLevel)
	}

	// Verify allow LAN
	if !config.AllowLan {
		t.Error("Expected AllowLan to be true")
	}

	// Verify proxy structure
	if len(config.Proxies) != 1 {
		t.Fatalf("Expected 1 proxy, got %d", len(config.Proxies))
	}

	proxy := config.Proxies[0]
	if proxy["type"] != "socks" {
		t.Errorf("Expected proxy type 'socks', got '%v'", proxy["type"])
	}
	if proxy["port"] != 1080 {
		t.Errorf("Expected port 1080, got %v", proxy["port"])
	}

	// Verify proxy groups are created
	if len(config.ProxyGroups) == 0 {
		t.Error("Expected at least one proxy group")
	}

	// Verify default rules
	if len(config.Rules) == 0 {
		t.Error("Expected at least one rule")
	}
}
