package xray_test

import (
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/core/xray"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func TestGenerateXrayConfig(t *testing.T) {
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
					Name:          "vless-inbound",
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
					Name:          "vless-inbound",
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
			config, err := xray.GenerateConfig(tt.inbounds, tt.outbounds)
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

func TestXrayConfigStructure(t *testing.T) {
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

	config, err := xray.GenerateConfig(inbounds, []models.Outbound{})
	if err != nil {
		t.Fatalf("GenerateConfig() failed: %v", err)
	}

	// Verify log level
	if config.Log.LogLevel != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Log.LogLevel)
	}

	// Verify inbound structure
	if len(config.Inbounds) != 1 {
		t.Fatalf("Expected 1 inbound, got %d", len(config.Inbounds))
	}

	inbound := config.Inbounds[0]
	if inbound.Protocol != "vless" {
		t.Errorf("Expected protocol 'vless', got '%s'", inbound.Protocol)
	}
	if inbound.Port != 443 {
		t.Errorf("Expected port 443, got %d", inbound.Port)
	}
	if inbound.Listen != "0.0.0.0" {
		t.Errorf("Expected listen '0.0.0.0', got '%s'", inbound.Listen)
	}

	// Verify TLS is configured
	if inbound.StreamSettings == nil {
		t.Error("Expected StreamSettings to be configured")
	} else if inbound.StreamSettings.Security != "tls" {
		t.Errorf("Expected security 'tls', got '%s'", inbound.StreamSettings.Security)
	}

	// Verify default outbound
	if len(config.Outbounds) == 0 {
		t.Error("Expected at least one outbound")
	} else {
		outbound := config.Outbounds[0]
		if outbound.Protocol != "freedom" {
			t.Errorf("Expected default outbound protocol 'freedom', got '%s'", outbound.Protocol)
		}
	}
}
