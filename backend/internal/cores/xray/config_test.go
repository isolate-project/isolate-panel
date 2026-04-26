package xray_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/models"
)

var b64Encoding = base64.StdEncoding

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

func TestBuildClients_VLESSWithReality(t *testing.T) {
	db := setupTestDB(t)

	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	user := models.User{
		Username: "reality-user",
		UUID:     "reality-uuid-123",
		IsActive: true,
	}
	db.Create(&user)

	inbound := models.Inbound{
		Name:           "test-vless-reality",
		Protocol:       "vless",
		CoreID:         core.ID,
		Port:           443,
		IsEnabled:      true,
		RealityEnabled: true,
		ConfigJSON:     `{"decryption": "none"}`,
	}
	db.Create(&inbound)

	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	})

	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	if len(config.Inbounds) < 2 {
		t.Fatal("not enough inbounds")
	}

	var vlessInbound *xray.InboundConfig
	for _, inbound := range config.Inbounds {
		if inbound.Protocol == "vless" {
			vlessInbound = &inbound
			break
		}
	}

	if vlessInbound == nil {
		t.Fatal("VLESS inbound not found")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(vlessInbound.Settings, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	clients, ok := settings["clients"].([]interface{})
	if !ok {
		t.Fatal("clients not found in settings")
	}

	if len(clients) == 0 {
		t.Fatal("no clients found")
	}

	firstClient, ok := clients[0].(map[string]interface{})
	if !ok {
		t.Fatal("first client is not a map")
	}

	flow, ok := firstClient["flow"].(string)
	if !ok {
		t.Fatal("flow field not found or not a string")
	}

	if flow != "xtls-rprx-vision" {
		t.Errorf("expected flow 'xtls-rprx-vision', got '%s'", flow)
	}

	encryption, ok := firstClient["encryption"].(string)
	if !ok {
		t.Fatal("encryption field not found or not a string")
	}

	if encryption != "none" {
		t.Errorf("expected encryption 'none', got '%s'", encryption)
	}
}

func TestBuildClients_VLESSWithoutReality(t *testing.T) {
	db := setupTestDB(t)

	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	user := models.User{
		Username: "non-reality-user",
		UUID:     "non-reality-uuid-456",
		IsActive: true,
	}
	db.Create(&user)

	inbound := models.Inbound{
		Name:           "test-vless-no-reality",
		Protocol:       "vless",
		CoreID:         core.ID,
		Port:           443,
		IsEnabled:      true,
		RealityEnabled: false,
		ConfigJSON:     `{"decryption": "none"}`,
	}
	db.Create(&inbound)

	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	})

	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	if len(config.Inbounds) < 2 {
		t.Fatal("not enough inbounds")
	}

	var vlessInbound *xray.InboundConfig
	for _, inbound := range config.Inbounds {
		if inbound.Protocol == "vless" {
			vlessInbound = &inbound
			break
		}
	}

	if vlessInbound == nil {
		t.Fatal("VLESS inbound not found")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(vlessInbound.Settings, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	clients, ok := settings["clients"].([]interface{})
	if !ok {
		t.Fatal("clients not found in settings")
	}

	if len(clients) == 0 {
		t.Fatal("no clients found")
	}

	firstClient, ok := clients[0].(map[string]interface{})
	if !ok {
		t.Fatal("first client is not a map")
	}

	flow, ok := firstClient["flow"].(string)
	if !ok {
		t.Fatal("flow field not found or not a string")
	}

	if flow != "" {
		t.Errorf("expected empty flow, got '%s'", flow)
	}

	encryption, ok := firstClient["encryption"].(string)
	if !ok {
		t.Fatal("encryption field not found or not a string")
	}

	if encryption != "none" {
		t.Errorf("expected encryption 'none', got '%s'", encryption)
	}
}

func TestGenerateConfig_Hysteria2WithQUICParams(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a test user
	user := models.User{
		Username: "testuser",
		UUID:     "test-uuid-12345",
		IsActive: true,
	}
	db.Create(&user)

	// Create a hysteria2 inbound with QUIC params in ConfigJSON
	configJSON := `{"congestion": "bbr", "brutal_up": "100 mbps", "brutal_down": "200 mbps", "force_brutal": true}`
	inbound := models.Inbound{
		Name:       "test-hysteria2",
		Protocol:   "hysteria2",
		CoreID:     core.ID,
		Port:       443,
		IsEnabled:  true,
		ConfigJSON: configJSON,
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

	// Find the hysteria2 inbound
	var hysteria2Inbound *xray.InboundConfig
	for _, ib := range config.Inbounds {
		if ib.Protocol == "hysteria2" {
			hysteria2Inbound = &ib
			break
		}
	}

	if hysteria2Inbound == nil {
		t.Fatal("hysteria2 inbound not found")
	}

	// Parse settings to verify finalmask.quicParams structure
	var settings map[string]interface{}
	if err := json.Unmarshal(hysteria2Inbound.Settings, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	// Verify finalmask exists
	finalmask, ok := settings["finalmask"].(map[string]interface{})
	if !ok {
		t.Error("finalmask not found in settings")
	}

	// Verify quicParams exists
	quicParams, ok := finalmask["quicParams"].(map[string]interface{})
	if !ok {
		t.Error("quicParams not found in finalmask")
	}

	// Verify congestion parameter
	if congestion, ok := quicParams["congestion"].(string); !ok || congestion != "bbr" {
		t.Errorf("expected congestion 'bbr', got %v", quicParams["congestion"])
	}

	// Verify brutal_up parameter
	if brutalUp, ok := quicParams["brutal_up"].(string); !ok || brutalUp != "100 mbps" {
		t.Errorf("expected brutal_up '100 mbps', got %v", quicParams["brutal_up"])
	}

	// Verify brutal_down parameter
	if brutalDown, ok := quicParams["brutal_down"].(string); !ok || brutalDown != "200 mbps" {
		t.Errorf("expected brutal_down '200 mbps', got %v", quicParams["brutal_down"])
	}

	// Verify force_brutal parameter
	if forceBrutal, ok := quicParams["force_brutal"].(bool); !ok || !forceBrutal {
		t.Errorf("expected force_brutal true, got %v", quicParams["force_brutal"])
	}

	// Verify old top-level params are removed
	if _, ok := settings["congestion"]; ok {
		t.Error("congestion should be removed from top-level settings")
	}
	if _, ok := settings["brutal_up"]; ok {
		t.Error("brutal_up should be removed from top-level settings")
	}
	if _, ok := settings["brutal_down"]; ok {
		t.Error("brutal_down should be removed from top-level settings")
	}
	if _, ok := settings["force_brutal"]; ok {
		t.Error("force_brutal should be removed from top-level settings")
	}
}

func TestGenerateConfig_ECHForceQuery(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a test certificate
	cert := models.Certificate{
		Domain:   "example.com",
		CertPath: "/path/to/cert.pem",
		KeyPath:  "/path/to/key.pem",
	}
	db.Create(&cert)

	// Create a test user
	user := models.User{
		Username: "testuser",
		UUID:     "test-uuid-12345",
		IsActive: true,
	}
	db.Create(&user)

	// Test 1: Default echForceQuery should be "off"
	inbound1 := models.Inbound{
		Name:        "test-vmess-default",
		Protocol:    "vmess",
		CoreID:      core.ID,
		Port:        443,
		IsEnabled:   true,
		TLSEnabled:  true,
		TLSCertID:   &cert.ID,
		ConfigJSON:  `{}`,
	}
	db.Create(&inbound1)
	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound1.ID,
	})

	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Find the vmess inbound with TLS
	var vmessInbound *xray.InboundConfig
	for _, ib := range config.Inbounds {
		if ib.Protocol == "vmess" && ib.StreamSettings != nil {
			vmessInbound = &ib
			break
		}
	}

	if vmessInbound == nil {
		t.Fatal("vmess inbound with TLS not found")
	}

	if vmessInbound.StreamSettings.TLSConfig == nil {
		t.Fatal("TLSConfig is nil")
	}

	if vmessInbound.StreamSettings.TLSConfig.ECHForceQuery != "off" {
		t.Errorf("expected echForceQuery 'off', got '%s'", vmessInbound.StreamSettings.TLSConfig.ECHForceQuery)
	}

	// Test 2: Custom echForceQuery should be respected
	inbound2 := models.Inbound{
		Name:        "test-vmess-custom",
		Protocol:    "vmess",
		CoreID:      core.ID,
		Port:        8443,
		IsEnabled:   true,
		TLSEnabled:  true,
		TLSCertID:   &cert.ID,
		ConfigJSON:  `{"ech_force_query": "strict"}`,
	}
	db.Create(&inbound2)
	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound2.ID,
	})

	config, err = xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Find the second vmess inbound
	var vmessInbound2 *xray.InboundConfig
	for _, ib := range config.Inbounds {
		if ib.Protocol == "vmess" && ib.Port == 8443 {
			vmessInbound2 = &ib
			break
		}
	}

	if vmessInbound2 == nil {
		t.Fatal("second vmess inbound with TLS not found")
	}

	if vmessInbound2.StreamSettings.TLSConfig == nil {
		t.Fatal("TLSConfig is nil for second inbound")
	}

	if vmessInbound2.StreamSettings.TLSConfig.ECHForceQuery != "strict" {
		t.Errorf("expected echForceQuery 'strict', got '%s'", vmessInbound2.StreamSettings.TLSConfig.ECHForceQuery)
	}
}

func TestGenerateConfig_XHTTPWithSettings(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a test user
	user := models.User{
		Username: "testuser",
		UUID:     "test-uuid-12345",
		IsActive: true,
	}
	db.Create(&user)

	// Create an XHTTP inbound with custom settings
	configJSON := `{"transport": "xhttp", "xhttp_path": "/custom-path", "xhttp_host": "custom.example.com", "xhttp_mode": "packet-up"}`
	inbound := models.Inbound{
		Name:       "test-xhttp",
		Protocol:   "xhttp",
		CoreID:     core.ID,
		Port:       443,
		IsEnabled:  true,
		ConfigJSON: configJSON,
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

	// Find the XHTTP inbound
	var xhttpInbound *xray.InboundConfig
	for _, ib := range config.Inbounds {
		if ib.Protocol == "xhttp" {
			xhttpInbound = &ib
			break
		}
	}

	if xhttpInbound == nil {
		t.Fatal("XHTTP inbound not found")
	}

	if xhttpInbound.StreamSettings == nil {
		t.Fatal("StreamSettings is nil")
	}

	if xhttpInbound.StreamSettings.Network != "splithttp" {
		t.Errorf("expected network 'splithttp', got '%s'", xhttpInbound.StreamSettings.Network)
	}

	if xhttpInbound.StreamSettings.XHTTPConfig == nil {
		t.Fatal("XHTTPConfig is nil")
	}

	if xhttpInbound.StreamSettings.XHTTPConfig.Path != "/custom-path" {
		t.Errorf("expected path '/custom-path', got '%s'", xhttpInbound.StreamSettings.XHTTPConfig.Path)
	}

	if xhttpInbound.StreamSettings.XHTTPConfig.Host != "custom.example.com" {
		t.Errorf("expected host 'custom.example.com', got '%s'", xhttpInbound.StreamSettings.XHTTPConfig.Host)
	}

	if xhttpInbound.StreamSettings.XHTTPConfig.Mode != "packet-up" {
		t.Errorf("expected mode 'packet-up', got '%s'", xhttpInbound.StreamSettings.XHTTPConfig.Mode)
	}
}

func TestGenerateConfig_TUNInbound(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a TUN inbound with custom settings
	configJSON := `{"interface_name": "tun-custom", "inet4_address": "10.0.0.1/24", "inet6_address": "fd00::1/64", "mtu": 8500, "stack": "gvisor"}`
	inbound := models.Inbound{
		Name:       "test-tun",
		Protocol:   "tun",
		CoreID:     core.ID,
		Port:       0, // TUN doesn't use port
		IsEnabled:  true,
		ConfigJSON: configJSON,
	}
	db.Create(&inbound)

	// Generate config
	config, err := xray.GenerateConfig(testCtx(db), core.ID)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Find the TUN inbound
	var tunInbound *xray.InboundConfig
	for _, ib := range config.Inbounds {
		if ib.Protocol == "tun" {
			tunInbound = &ib
			break
		}
	}

	if tunInbound == nil {
		t.Fatal("TUN inbound not found")
	}

	// Verify protocol is "tun"
	if tunInbound.Protocol != "tun" {
		t.Errorf("expected protocol 'tun', got '%s'", tunInbound.Protocol)
	}

	// Verify StreamSettings is nil (TUN doesn't use stream settings)
	if tunInbound.StreamSettings != nil {
		t.Error("expected StreamSettings to be nil for TUN")
	}

	// Parse settings to verify TUN configuration
	var settings map[string]interface{}
	if err := json.Unmarshal(tunInbound.Settings, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	// Verify name (interface_name maps to name in Xray config)
	if name, ok := settings["name"].(string); !ok || name != "tun-custom" {
		t.Errorf("expected name 'tun-custom', got %v", settings["name"])
	}

	// Verify mtu
	if mtu, ok := settings["mtu"].(float64); !ok || mtu != 8500 {
		t.Errorf("expected mtu 8500, got %v", settings["mtu"])
	}

	// Verify stack
	if stack, ok := settings["stack"].(string); !ok || stack != "gvisor" {
		t.Errorf("expected stack 'gvisor', got %v", settings["stack"])
	}

	// Verify inet4_address
	if inet4, ok := settings["inet4_address"].(string); !ok || inet4 != "10.0.0.1/24" {
		t.Errorf("expected inet4_address '10.0.0.1/24', got %v", settings["inet4_address"])
	}

	// Verify inet6_address
	if inet6, ok := settings["inet6_address"].(string); !ok || inet6 != "fd00::1/64" {
		t.Errorf("expected inet6_address 'fd00::1/64', got %v", settings["inet6_address"])
	}
}

func TestGenerateConfig_FinalmaskTransport(t *testing.T) {
	db := setupTestDB(t)

	// Create a test core
	core := models.Core{
		Name:      "xray",
		IsEnabled: true,
	}
	db.Create(&core)

	// Create a test user
	user := models.User{
		Username: "testuser",
		UUID:     "test-uuid-12345",
		IsActive: true,
	}
	db.Create(&user)

	// Create a VLESS inbound with Finalmask settings
	configJSON := `{"finalmask_enabled": true, "finalmask_congestion": "bbr", "finalmask_brutal_up": "100 mbps", "finalmask_brutal_down": "200 mbps"}`
	inbound := models.Inbound{
		Name:       "test-vless-finalmask",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		IsEnabled:  true,
		ConfigJSON: configJSON,
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

	// Find the VLESS inbound
	var vlessInbound *xray.InboundConfig
	for _, ib := range config.Inbounds {
		if ib.Protocol == "vless" {
			vlessInbound = &ib
			break
		}
	}

	if vlessInbound == nil {
		t.Fatal("VLESS inbound not found")
	}

	// Parse settings to verify finalmask structure
	var settings map[string]interface{}
	if err := json.Unmarshal(vlessInbound.Settings, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	// Verify finalmask exists
	finalmask, ok := settings["finalmask"].(map[string]interface{})
	if !ok {
		t.Error("finalmask not found in settings")
	}

	// Verify quicParams exists
	quicParams, ok := finalmask["quicParams"].(map[string]interface{})
	if !ok {
		t.Error("quicParams not found in finalmask")
	}

	// Verify congestion parameter
	if congestion, ok := quicParams["congestion"].(string); !ok || congestion != "bbr" {
		t.Errorf("expected congestion 'bbr', got %v", quicParams["congestion"])
	}

	// Verify brutal_up parameter
	if brutalUp, ok := quicParams["brutal_up"].(string); !ok || brutalUp != "100 mbps" {
		t.Errorf("expected brutal_up '100 mbps', got %v", quicParams["brutal_up"])
	}

	// Verify brutal_down parameter
	if brutalDown, ok := quicParams["brutal_down"].(string); !ok || brutalDown != "200 mbps" {
		t.Errorf("expected brutal_down '200 mbps', got %v", quicParams["brutal_down"])
	}
}

func TestBuildInboundSettings_SS2022(t *testing.T) {
	db := setupTestDB(t)
	core := models.Core{Name: "xray", IsEnabled: true}
	if err := db.Create(&core).Error; err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	users := []models.User{
		{Username: "ss-user1", UUID: "uuid-ss-001", SubscriptionToken: "sub-ss-001", IsActive: true},
		{Username: "ss-user2", UUID: "uuid-ss-002", SubscriptionToken: "sub-ss-002", IsActive: true},
	}
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
	}

	ciphers := []struct {
		method      string
		expectedLen int
	}{
		{"2022-blake3-aes-128-gcm", 16},
		{"2022-blake3-aes-256-gcm", 32},
		{"2022-blake3-chacha20-poly1305", 32},
	}

	for _, tc := range ciphers {
		t.Run(tc.method, func(t *testing.T) {
			inbound := models.Inbound{
				Name:       "ss2022-test",
				Protocol:   "shadowsocks",
				CoreID:     core.ID,
				Port:       8388,
				IsEnabled:  true,
				ConfigJSON: fmt.Sprintf(`{"method": "%s"}`, tc.method),
			}
			if err := db.Create(&inbound).Error; err != nil {
				t.Fatalf("failed to create inbound: %v", err)
			}
			for _, u := range users {
				if err := db.Create(&models.UserInboundMapping{UserID: u.ID, InboundID: inbound.ID}).Error; err != nil {
					t.Fatalf("failed to assign user: %v", err)
				}
			}

			config, err := xray.GenerateConfig(testCtx(db), core.ID)
			if err != nil {
				t.Fatalf("GenerateConfig failed: %v", err)
			}

			data, err := json.Marshal(config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			inbounds, ok := raw["inbounds"].([]interface{})
			if !ok || len(inbounds) == 0 {
				t.Fatal("no inbounds in config")
			}

			var targetInbound map[string]interface{}
			for _, inb := range inbounds {
				ib := inb.(map[string]interface{})
				if ib["protocol"] == "shadowsocks" {
					targetInbound = ib
					break
				}
			}
			if targetInbound == nil {
				t.Fatal("shadowsocks inbound not found")
			}

			settings, ok := targetInbound["settings"].(map[string]interface{})
			if !ok {
				t.Fatal("settings not found in inbound")
			}

			if method, ok := settings["method"].(string); !ok || method != tc.method {
				t.Errorf("expected method %s, got %v", tc.method, settings["method"])
			}

			password, ok := settings["password"].(string)
			if !ok || password == "" {
				t.Fatal("server password not set for SS2022")
			}

			keyBytes, err := b64Encoding.DecodeString(password)
			if err != nil {
				t.Fatalf("password is not valid base64: %v", err)
			}
			if len(keyBytes) != tc.expectedLen {
				t.Errorf("expected %d-byte key for %s, got %d bytes", tc.expectedLen, tc.method, len(keyBytes))
			}

			clients, ok := settings["clients"].([]interface{})
			if !ok {
				t.Fatal("clients not found in settings")
			}
			for _, c := range clients {
				client := c.(map[string]interface{})
				if _, hasMethod := client["method"]; hasMethod {
					t.Error("SS2022 client should NOT have method field")
				}
				if _, hasLevel := client["level"]; hasLevel {
					t.Error("SS2022 client should NOT have level field")
				}
			}

			db.Exec("DELETE FROM user_inbound_mappings WHERE inbound_id = ?", inbound.ID)
			db.Delete(&inbound)
		})
	}
}
