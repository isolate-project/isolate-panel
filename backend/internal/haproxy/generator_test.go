package haproxy

import (
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
)

func TestGeneratorGenerate(t *testing.T) {
	generator, err := NewGenerator("templates/haproxy.cfg.tmpl", "test-password")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	assignments := []models.PortAssignment{
		{
			ID:                1,
			InboundID:         1,
			UserListenPort:    443,
			UserListenAddr:    "0.0.0.0",
			BackendPort:       40001,
			CoreType:          "xray",
			UseHAProxy:        true,
			SNIMatch:          "example.com",
			SendProxyProtocol: true,
			IsActive:          true,
		},
		{
			ID:             2,
			InboundID:      2,
			UserListenPort: 443,
			UserListenAddr: "0.0.0.0",
			BackendPort:    40002,
			CoreType:       "singbox",
			UseHAProxy:     true,
			SNIMatch:       "example2.com",
			IsActive:       true,
		},
		{
			ID:             3,
			InboundID:      3,
			UserListenPort: 8443,
			UserListenAddr: "0.0.0.0",
			BackendPort:    40003,
			CoreType:       "mihomo",
			UseHAProxy:     true,
			PathMatch:      "/mihomo",
			IsActive:       true,
		},
	}

	config, err := generator.Generate(assignments)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	expectedSections := []string{
		"global",
		"defaults",
		"frontend ft_user_443",
		"frontend ft_user_8443",
		"backend bk_xray_40001",
		"backend bk_singbox_40002",
		"backend bk_mihomo_40003",
		"backend bk_default_web",
		"stats socket /run/haproxy/admin.sock",
		"maxconn 4096",
		"mode tcp",
		"bind :443",
		"bind :8443",
		"acl is_inbound_1 req.ssl_sni -i example.com",
		"use_backend bk_xray_40001 if is_inbound_1",
		"acl is_inbound_2 req.ssl_sni -i example2.com",
		"use_backend bk_singbox_40002 if is_inbound_2",
		"use_backend bk_mihomo_40003 if { path_beg /mihomo }",
		"server xray_40001 127.0.0.1:40001 send-proxy-v2 check inter 5s",
		"server singbox_40002 127.0.0.1:40002 check inter 5s",
		"server mihomo_40003 127.0.0.1:40003 check inter 5s",
	}

	for _, section := range expectedSections {
		if !contains(config, section) {
			t.Errorf("Generated config missing expected section: %s", section)
		}
	}

	if len(config) == 0 {
		t.Error("Generated config is empty")
	}
}

func TestGeneratorGenerateWithNoAssignments(t *testing.T) {
	generator, err := NewGenerator("templates/haproxy.cfg.tmpl", "test-password")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	config, err := generator.Generate([]models.PortAssignment{})
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	expectedSections := []string{
		"global",
		"defaults",
		"backend bk_default_web",
	}

	for _, section := range expectedSections {
		if !contains(config, section) {
			t.Errorf("Generated config missing expected section: %s", section)
		}
	}
}

func TestGeneratorGenerateWithDisabledHAProxy(t *testing.T) {
	generator, err := NewGenerator("templates/haproxy.cfg.tmpl", "test-password")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	assignments := []models.PortAssignment{
		{
			ID:             1,
			InboundID:      1,
			UserListenPort: 443,
			UserListenAddr: "0.0.0.0",
			BackendPort:    40001,
			CoreType:       "xray",
			UseHAProxy:     false,
			IsActive:       true,
		},
	}

	config, err := generator.Generate(assignments)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	if contains(config, "frontend ft_user_443") {
		t.Error("Generated config should not have frontend for disabled HAProxy")
	}
	if contains(config, "backend bk_xray_40001") {
		t.Error("Generated config should not have backend for disabled HAProxy")
	}
}
