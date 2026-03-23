package singbox_test

import (
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/core/singbox"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func TestGenerateSingboxConfig(t *testing.T) {
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
					Protocol:      "vless",
					Port:          443,
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
					Protocol:      "vless",
					Port:          443,
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
			config, err := singbox.GenerateConfig(tt.inbounds, tt.outbounds)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if config == nil {
				t.Error("GenerateConfig() returned nil config")
				return
			}

			// Verify basic structure
			if config.Log == nil {
				t.Error("Config should have log section")
			}
			if config.DNS == nil {
				t.Error("Config should have DNS section")
			}
			if config.Inbounds == nil {
				t.Error("Config should have inbounds array")
			}
			if config.Outbounds == nil {
				t.Error("Config should have outbounds array")
			}

			// Verify default outbound is added when none provided
			if len(tt.outbounds) == 0 && len(config.Outbounds) == 0 {
				t.Error("Config should have at least one default outbound")
			}

			// Verify disabled inbounds are not included
			enabledCount := 0
			for _, inbound := range tt.inbounds {
				if inbound.IsEnabled {
					enabledCount++
				}
			}
			if len(config.Inbounds) != enabledCount {
				t.Errorf("Expected %d enabled inbounds, got %d", enabledCount, len(config.Inbounds))
			}
		})
	}
}

func TestSingboxConfigStructure(t *testing.T) {
	inbounds := []models.Inbound{
		{
			ID:            1,
			Name:          "vless-inbound",
			Protocol:      "vless",
			Port:          443,
			ListenAddress: "0.0.0.0",
			IsEnabled:     true,
			TLSEnabled:    true,
		},
	}

	config, err := singbox.GenerateConfig(inbounds, []models.Outbound{})
	if err != nil {
		t.Fatalf("GenerateConfig() failed: %v", err)
	}

	// Verify log level
	if config.Log.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Log.Level)
	}

	// Verify DNS servers
	if len(config.DNS.Servers) < 2 {
		t.Error("Expected at least 2 DNS servers")
	}

	// Verify inbound structure
	if len(config.Inbounds) != 1 {
		t.Fatalf("Expected 1 inbound, got %d", len(config.Inbounds))
	}

	inbound := config.Inbounds[0]
	if inbound.Type != "vless" {
		t.Errorf("Expected inbound type 'vless', got '%s'", inbound.Type)
	}
	if inbound.Port != 443 {
		t.Errorf("Expected port 443, got %d", inbound.Port)
	}
	if inbound.Listen != "0.0.0.0" {
		t.Errorf("Expected listen '0.0.0.0', got '%s'", inbound.Listen)
	}

	// Verify TLS is configured
	if inbound.TLS == nil {
		t.Error("Expected TLS to be configured")
	} else if !inbound.TLS.Enabled {
		t.Error("Expected TLS to be enabled")
	}
}
