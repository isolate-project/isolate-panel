package xray_test

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/cores"
	"github.com/vovk4morkovk4/isolate-panel/internal/cores/xray"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
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
		Name:      "xray",
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
		Name:          "test-vmess",
		Protocol:      "vmess",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          443,
		IsEnabled:     true,
	}
	db.Create(&inbound)

	// Assign user to inbound
	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	})

	// Generate config
	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify config structure
	if config == nil {
		t.Fatal("config is nil")
	}

	if config.API == nil {
		t.Error("API config is nil")
	}

	if config.Stats == nil {
		t.Error("Stats config is nil")
	}

	if config.Policy == nil {
		t.Error("Policy config is nil")
	}

	// Should have API inbound + user inbound
	if len(config.Inbounds) != 2 {
		t.Errorf("expected 2 inbounds, got %d", len(config.Inbounds))
	}

	// Verify API inbound
	apiInbound := config.Inbounds[0]
	if apiInbound.Tag != "api" {
		t.Errorf("expected API inbound tag 'api', got '%s'", apiInbound.Tag)
	}
	if apiInbound.Port != 10085 {
		t.Errorf("expected API port 10085, got %d", apiInbound.Port)
	}

	// Verify user inbound
	userInbound := config.Inbounds[1]
	if userInbound.Protocol != "vmess" {
		t.Errorf("expected protocol 'vmess', got '%s'", userInbound.Protocol)
	}
	if userInbound.Port != 443 {
		t.Errorf("expected port 443, got %d", userInbound.Port)
	}
}

func TestGenerateConfig_MultipleProtocols(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
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

	// Create inbounds with different protocols
	protocols := []string{"vmess", "vless", "trojan", "shadowsocks", "hysteria2", "xhttp"}

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
	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Should have API inbound + 6 protocol inbounds
	expectedInbounds := 1 + len(protocols)
	if len(config.Inbounds) != expectedInbounds {
		t.Errorf("expected %d inbounds, got %d", expectedInbounds, len(config.Inbounds))
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := xray.ValidateConfig(nil)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("missing API", func(t *testing.T) {
		config := &xray.Config{}
		err := xray.ValidateConfig(config)
		if err == nil {
			t.Error("expected error for missing API config")
		}
	})

	t.Run("missing inbounds", func(t *testing.T) {
		config := &xray.Config{
			API:      &xray.APIConfig{},
			Inbounds: []xray.InboundConfig{},
		}
		err := xray.ValidateConfig(config)
		if err == nil {
			t.Error("expected error for empty inbounds")
		}
	})

	t.Run("duplicate tags", func(t *testing.T) {
		config := &xray.Config{
			API: &xray.APIConfig{},
			Inbounds: []xray.InboundConfig{
				{Tag: "test"},
				{Tag: "test"},
			},
		}
		err := xray.ValidateConfig(config)
		if err == nil {
			t.Error("expected error for duplicate tags")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		config := &xray.Config{
			API: &xray.APIConfig{
				Tag:      "api",
				Services: []string{"HandlerService"},
			},
			Inbounds: []xray.InboundConfig{
				{Tag: "api", Protocol: "dokodemo-door"},
				{Tag: "vmess_1", Protocol: "vmess"},
			},
			Outbounds: []xray.OutboundConfig{
				{Tag: "direct", Protocol: "freedom"},
			},
		}
		err := xray.ValidateConfig(config)
		if err != nil {
			t.Errorf("unexpected error for valid config: %v", err)
		}
	})
}

func TestWriteConfig(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a simple inbound
	inbound := models.Inbound{
		Name:      "test",
		Protocol:  "vmess",
		CoreID:    core.ID,
		Port:      443,
		IsEnabled: true,
	}
	db.Create(&inbound)

	// Generate and write config
	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Write to temp file
	tmpPath := t.TempDir() + "/config.json"
	err = xray.WriteConfig(config, tmpPath)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Read back and verify
	readConfig, err := xray.ReadConfig(tmpPath)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	if len(readConfig.Inbounds) != len(config.Inbounds) {
		t.Errorf("inbound count mismatch: expected %d, got %d", len(config.Inbounds), len(readConfig.Inbounds))
	}
}

func TestBuildClients(t *testing.T) {
	// This tests the internal buildClients function indirectly
	// by generating config and checking client count

	db := setupTestDB(t)

	core := models.Core{Name: "xray", IsEnabled: true}
	db.Create(&core)

	// Create 5 users
	for i := 0; i < 5; i++ {
		db.Create(&models.User{
			Username: fmt.Sprintf("user%d", i),
			UUID:     fmt.Sprintf("uuid-%d", i),
			IsActive: true,
		})
	}

	inbound := models.Inbound{
		Name:     "test-vmess",
		Protocol: "vmess",
		CoreID:   core.ID,
		Port:     443,
	}
	db.Create(&inbound)

	// Assign all users
	var users []models.User
	db.Find(&users)
	for _, user := range users {
		db.Create(&models.UserInboundMapping{
			UserID:    user.ID,
			InboundID: inbound.ID,
		})
	}

	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Find the vmess inbound (should be second after API)
	if len(config.Inbounds) < 2 {
		t.Fatal("not enough inbounds")
	}

	// Note: We can't directly check client count without parsing settings JSON
	// This is a basic sanity check
	if config.Inbounds[1].Protocol != "vmess" {
		t.Errorf("expected vmess protocol, got %s", config.Inbounds[1].Protocol)
	}
}
