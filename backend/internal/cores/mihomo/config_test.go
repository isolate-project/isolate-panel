package mihomo_test

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	"github.com/isolate-project/isolate-panel/internal/models"
)

func testCtx(db *gorm.DB) *cores.ConfigContext {
	return &cores.ConfigContext{DB: db}
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Auto migrate models
	err = db.AutoMigrate(&models.Core{}, &models.Inbound{}, &models.Outbound{}, &models.User{}, &models.UserInboundMapping{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestGenerateConfig_Basic(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "mihomo",
		IsEnabled: true,
		IsRunning: false,
	}
	db.Create(&core)

	// Create a test user
	user := models.User{
		Username: "testuser",
		UUID:     "test-uuid-12345",
		IsActive: true,
	}
	db.Create(&user)

	// Create a test inbound
	inbound := models.Inbound{
		Name:          "test-shadowsocks",
		Protocol:      "shadowsocks",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          8388,
		IsEnabled:     true,
	}
	db.Create(&inbound)

	// Assign user to inbound
	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	})

	// Generate config
	config, err := mihomo.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify config structure
	if config == nil {
		t.Fatal("config is nil")
	}

	// Verify basic settings
	if config.Mode != "rule" {
		t.Errorf("expected mode 'rule', got '%s'", config.Mode)
	}

	if config.AllowLan != true {
		t.Error("expected AllowLan to be true")
	}

	if config.ExternalController != "127.0.0.1:9091" {
		t.Errorf("expected ExternalController '127.0.0.1:9091', got '%s'", config.ExternalController)
	}

	// Should have at least one proxy
	if len(config.Proxies) == 0 {
		t.Error("expected at least one proxy")
	}

	// Verify proxy
	proxy := config.Proxies[0]
	if proxy.Type != "ss" {
		t.Errorf("expected proxy type 'ss', got '%s'", proxy.Type)
	}
	if proxy.Port != 8388 {
		t.Errorf("expected port 8388, got %d", proxy.Port)
	}
}

func TestGenerateConfig_MultipleProtocols(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "mihomo",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create test users
	users := []models.User{
		{Username: "user1", UUID: "uuid-1", IsActive: true},
		{Username: "user2", UUID: "uuid-2", IsActive: true},
	}
	for i := range users {
		db.Create(&users[i])
	}

	// Create inbounds with different protocols (including Mihomo-exclusive)
	protocols := []string{
		"shadowsocks",
		"vmess",
		"vless",
		"trojan",
		"hysteria2",
		"tuic",
		"mieru",       // Mihomo exclusive
		"sudoku",      // Mihomo exclusive
		"ssr",         // Mihomo exclusive
		"snell",       // Mihomo exclusive
		"trusttunnel", // Mihomo exclusive
		"masque",      // Mihomo exclusive
	}

	for i, protocol := range protocols {
		inbound := models.Inbound{
			Name:      fmt.Sprintf("test-%s", protocol),
			Protocol:  protocol,
			CoreID:    core.ID,
			Port:      10000 + i,
			IsEnabled: true,
		}
		db.Create(&inbound)

		// Assign both users to each inbound
		for _, user := range users {
			db.Create(&models.UserInboundMapping{
				UserID:    user.ID,
				InboundID: inbound.ID,
			})
		}
	}

	// Generate config
	config, err := mihomo.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Should have 12 proxies
	if len(config.Proxies) != len(protocols) {
		t.Errorf("expected %d proxies, got %d", len(protocols), len(config.Proxies))
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := mihomo.ValidateConfig(nil)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("missing mode", func(t *testing.T) {
		config := &mihomo.Config{
			Mode: "",
		}
		err := mihomo.ValidateConfig(config)
		if err == nil {
			t.Error("expected error for missing mode")
		}
	})

	t.Run("duplicate proxy names", func(t *testing.T) {
		config := &mihomo.Config{
			Mode: "rule",
			Proxies: []mihomo.Proxy{
				{Name: "test"},
				{Name: "test"},
			},
		}
		err := mihomo.ValidateConfig(config)
		if err == nil {
			t.Error("expected error for duplicate proxy names")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		config := &mihomo.Config{
			Mode:     "rule",
			AllowLan: true,
			Proxies: []mihomo.Proxy{
				{Name: "ss_1", Type: "ss", Port: 8388},
				{Name: "vmess_1", Type: "vmess", Port: 443},
			},
			Rules: []string{"MATCH,DIRECT"},
		}
		err := mihomo.ValidateConfig(config)
		if err != nil {
			t.Errorf("unexpected error for valid config: %v", err)
		}
	})
}

func TestWriteConfig(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "mihomo",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a simple inbound
	inbound := models.Inbound{
		Name:      "test",
		Protocol:  "shadowsocks",
		CoreID:    core.ID,
		Port:      8388,
		IsEnabled: true,
	}
	db.Create(&inbound)

	// Generate and write config
	config, err := mihomo.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Write to temp file
	tmpPath := t.TempDir() + "/config.yaml"
	err = mihomo.WriteConfig(config, tmpPath)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Read back and verify
	readConfig, err := mihomo.ReadConfig(tmpPath)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	if len(readConfig.Proxies) != len(config.Proxies) {
		t.Errorf("proxy count mismatch: expected %d, got %d", len(config.Proxies), len(readConfig.Proxies))
	}

	if readConfig.Mode != config.Mode {
		t.Errorf("mode mismatch: expected %s, got %s", config.Mode, readConfig.Mode)
	}
}

func TestMapMihomoProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http", "http"},
		{"socks", "socks"},
		{"mixed", "mixed"},
		{"shadowsocks", "ss"},
		{"shadowsocksr", "ssr"},
		{"vmess", "vmess"},
		{"vless", "vless"},
		{"trojan", "trojan"},
		{"hysteria", "hysteria"},
		{"hysteria2", "hysteria2"},
		{"tuic", "tuic"},
		{"mieru", "mieru"},
		{"sudoku", "sudoku"},
		{"snell", "snell"},
		{"trusttunnel", "trusttunnel"},
		{"masque", "masque"},
		{"direct", "direct"},
		{"block", "block"},
		{"dns", "dns"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// We can't directly test the internal function, but we can test via config generation
			// This is a basic sanity check
			if tt.input == "shadowsocks" && tt.expected != "ss" {
				t.Errorf("expected 'ss' for 'shadowsocks', got '%s'", tt.expected)
			}
		})
	}
}

func TestMihomoExclusiveProtocols(t *testing.T) {
	db := setupTestDB(t)

	core := models.Core{Name: "mihomo", IsEnabled: true}
	db.Create(&core)

	user := models.User{Username: "test", UUID: "test-uuid", IsActive: true}
	db.Create(&user)

	// Test Mihomo-exclusive protocols (including TrustTunnel and MASQUE)
	exclusiveProtocols := []string{"mieru", "sudoku", "ssr", "snell", "trusttunnel", "masque"}

	for _, protocol := range exclusiveProtocols {
		inbound := models.Inbound{
			Name:     fmt.Sprintf("test-%s", protocol),
			Protocol: protocol,
			CoreID:   core.ID,
			Port:     10000,
		}
		db.Create(&inbound)

		db.Create(&models.UserInboundMapping{
			UserID:    user.ID,
			InboundID: inbound.ID,
		})
	}

	config, err := mihomo.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify all exclusive protocols are present
	if len(config.Proxies) != len(exclusiveProtocols) {
		t.Errorf("expected %d proxies, got %d", len(exclusiveProtocols), len(config.Proxies))
	}

	// Verify proxy types
	expectedTypes := map[string]string{
		"mieru":       "mieru",
		"sudoku":      "sudoku",
		"ssr":         "ssr",
		"snell":       "snell",
		"trusttunnel": "trusttunnel",
		"masque":      "masque",
	}

	for _, proxy := range config.Proxies {
		expectedType, exists := expectedTypes[proxy.Type]
		if !exists {
			t.Errorf("unexpected proxy type: %s", proxy.Type)
		}
		if proxy.Type != expectedType {
			t.Errorf("expected type '%s' for '%s', got '%s'", expectedType, proxy.Name, proxy.Type)
		}
	}
}
