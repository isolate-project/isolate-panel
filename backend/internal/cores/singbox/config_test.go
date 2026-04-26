package singbox_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/cores/singbox"
	"github.com/isolate-project/isolate-panel/internal/models"
)

func testCtx(db *gorm.DB) *cores.ConfigContext {
	cc := &cores.CoreConfig{}
	cc.ApplyDefaults()
	return &cores.ConfigContext{DB: db, CoreConfig: cc}
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open database")

	err = db.AutoMigrate(&models.Core{}, &models.Inbound{}, &models.Outbound{}, &models.User{}, &models.UserInboundMapping{}, &models.WarpRoute{}, &models.GeoRule{})
	require.NoError(t, err, "failed to migrate database")

	return db
}

func createCoreAndUsers(t *testing.T, db *gorm.DB, userCount int) (models.Core, []models.User) {
	core := models.Core{Name: "singbox", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)

	users := make([]models.User, userCount)
	for i := 0; i < userCount; i++ {
		users[i] = models.User{
			Username:          fmt.Sprintf("user%d", i),
			UUID:              fmt.Sprintf("uuid-test-%d", i),
			SubscriptionToken: fmt.Sprintf("sub-token-test-%d", i),
			IsActive:          true,
		}
		require.NoError(t, db.Create(&users[i]).Error)
	}

	return core, users
}

func assignUsers(t *testing.T, db *gorm.DB, inboundID uint, users []models.User) {
	for _, user := range users {
		require.NoError(t, db.Create(&models.UserInboundMapping{
			UserID:    user.ID,
			InboundID: inboundID,
		}).Error)
	}
}

func TestGenerateConfig_Basic(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 2)

	inbound := models.Inbound{
		Name:          "test-vmess",
		Protocol:      "vmess",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          443,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check base structure
	assert.NotNil(t, config.Log)
	assert.Equal(t, "warning", config.Log.Level)
	assert.NotNil(t, config.Experimental)
	assert.NotNil(t, config.Experimental.ClashAPI)
	assert.Equal(t, "127.0.0.1:9090", config.Experimental.ClashAPI.ExternalController)
	assert.NotNil(t, config.Route)
	assert.Equal(t, "direct", config.Route.Final)

	// Check DNS servers (v1.12+ format)
	assert.NotNil(t, config.DNS)
	assert.Len(t, config.DNS.Servers, 2)
	assert.Equal(t, "https", config.DNS.Servers[0].Type)
	assert.Equal(t, "1.1.1.1", config.DNS.Servers[0].Server)
	assert.Equal(t, "local", config.DNS.Servers[1].Type)

	// Check inbound
	require.Len(t, config.Inbounds, 1)
	assert.Equal(t, "vmess", config.Inbounds[0].Type)
	assert.Equal(t, 443, config.Inbounds[0].ListenPort)

	// Check users were included
	assert.NotNil(t, config.Inbounds[0].Users)
	var vmessUsers []singbox.VMessUser
	err = json.Unmarshal(config.Inbounds[0].Users, &vmessUsers)
	require.NoError(t, err)
	assert.Len(t, vmessUsers, 2)
	assert.Equal(t, "uuid-test-0", vmessUsers[0].UUID)
	assert.Equal(t, "uuid-test-1", vmessUsers[1].UUID)

	// Check default outbound
	require.Len(t, config.Outbounds, 1)
	assert.Equal(t, "direct", config.Outbounds[0].Type)
}

func TestGenerateConfig_MultipleProtocols(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 2)

	// Sing-box supported inbound protocols
	protocols := []struct {
		name     string
		protocol string
		port     int
	}{
		{"vmess-in", "vmess", 10001},
		{"vless-in", "vless", 10002},
		{"trojan-in", "trojan", 10003},
		{"ss-in", "shadowsocks", 10004},
		{"hy2-in", "hysteria2", 10005},
		{"tuic-v5-in", "tuic_v5", 10006},
		{"naive-in", "naive", 10007},
		{"http-in", "http", 10008},
		{"socks5-in", "socks5", 10009},
		{"mixed-in", "mixed", 10010},
	}

	for _, p := range protocols {
		inbound := models.Inbound{
			Name:      p.name,
			Protocol:  p.protocol,
			CoreID:    core.ID,
			Port:      p.port,
			IsEnabled: true,
		}
		require.NoError(t, db.Create(&inbound).Error)
		assignUsers(t, db, inbound.ID, users)
	}

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)

	assert.Len(t, config.Inbounds, len(protocols))

	// Verify each protocol was mapped correctly
	expectedTypes := map[string]string{
		"vmess": "vmess", "vless": "vless", "trojan": "trojan",
		"shadowsocks": "shadowsocks", "hysteria2": "hysteria2",
		"tuic_v5": "tuic", "naive": "naive",
		"http": "http", "socks5": "socks", "mixed": "mixed",
	}

	for _, inb := range config.Inbounds {
		// Extract original protocol from tag (format: protocol_ID)
		for proto, expectedType := range expectedTypes {
			tag := fmt.Sprintf("%s_", proto)
			if len(inb.Tag) > len(tag) && inb.Tag[:len(tag)] == tag {
				assert.Equal(t, expectedType, inb.Type, "protocol %s should map to type %s", proto, expectedType)
			}
		}
	}
}

func TestGenerateConfig_VLESSUsers(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 3)

	inbound := models.Inbound{
		Name:     "vless-test",
		Protocol: "vless",
		CoreID:   core.ID,
		Port:     443,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	var vlessUsers []singbox.VLESSUser
	err = json.Unmarshal(config.Inbounds[0].Users, &vlessUsers)
	require.NoError(t, err)
	assert.Len(t, vlessUsers, 3)
	for i, u := range vlessUsers {
		assert.Equal(t, fmt.Sprintf("uuid-test-%d", i), u.UUID)
		assert.Empty(t, u.Flow, "Flow should be empty when Reality is disabled")
	}
}

func TestGenerateConfig_VLESSUsersWithReality(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 3)

	inbound := models.Inbound{
		Name:            "vless-reality-test",
		Protocol:        "vless",
		CoreID:          core.ID,
		Port:            443,
		RealityEnabled:  true,
		RealityConfigJSON: `{
			"dest": "www.microsoft.com",
			"serverPort": 443,
			"privateKey": "test-private-key",
			"shortIds": ["test-short-id"],
			"serverNames": ["www.microsoft.com"]
		}`,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	var vlessUsers []singbox.VLESSUser
	err = json.Unmarshal(config.Inbounds[0].Users, &vlessUsers)
	require.NoError(t, err)
	assert.Len(t, vlessUsers, 3)
	for i, u := range vlessUsers {
		assert.Equal(t, fmt.Sprintf("uuid-test-%d", i), u.UUID)
		assert.Equal(t, "xtls-rprx-vision", u.Flow, "Flow should be set to xtls-rprx-vision when Reality is enabled")
	}

	// Verify Reality settings are also configured
	assert.NotNil(t, config.Inbounds[0].TLS)
	assert.True(t, config.Inbounds[0].TLS.Enabled)
	assert.NotNil(t, config.Inbounds[0].TLS.Reality)
	assert.True(t, config.Inbounds[0].TLS.Reality.Enabled)
}

func TestGenerateConfig_TrojanUsers(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 2)

	inbound := models.Inbound{
		Name:       "trojan-test",
		Protocol:   "trojan",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Verify TLS is enabled
	assert.NotNil(t, config.Inbounds[0].TLS)
	assert.True(t, config.Inbounds[0].TLS.Enabled)

	// Verify Trojan users
	var trojanUsers []singbox.TrojanUser
	err = json.Unmarshal(config.Inbounds[0].Users, &trojanUsers)
	require.NoError(t, err)
	assert.Len(t, trojanUsers, 2)
	assert.Equal(t, "uuid-test-0", trojanUsers[0].Password)
}

func TestGenerateConfig_ShadowsocksMultiUser(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 3)

	inbound := models.Inbound{
		Name:       "ss-test",
		Protocol:   "shadowsocks",
		CoreID:     core.ID,
		Port:       8388,
		ConfigJSON: `{"method": "2022-blake3-aes-256-gcm", "password": "server-password-base64"}`,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Verify users
	var ssUsers []singbox.ShadowsocksUser
	err = json.Unmarshal(config.Inbounds[0].Users, &ssUsers)
	require.NoError(t, err)
	assert.Len(t, ssUsers, 3)

	// Verify the config marshals correctly with method in Extra
	data, err := json.Marshal(config.Inbounds[0])
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)
	assert.Equal(t, "2022-blake3-aes-256-gcm", raw["method"])
}

func TestGenerateConfig_Hysteria2WithSettings(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 1)

	inbound := models.Inbound{
		Name:       "hy2-test",
		Protocol:   "hysteria2",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		ConfigJSON: `{"up_mbps": 200, "down_mbps": 500, "obfs_type": "salamander", "obfs_password": "test-obfs-pass"}`,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Marshal and check extra fields
	data, err := json.Marshal(config.Inbounds[0])
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, float64(200), raw["up_mbps"])
	assert.Equal(t, float64(500), raw["down_mbps"])
	assert.NotNil(t, raw["obfs"])
}

func TestGenerateConfig_Hysteria2CongestionControl(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 1)

	inbound := models.Inbound{
		Name:       "hy2-cc-test",
		Protocol:   "hysteria2",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		ConfigJSON: `{"congestion_control": "cubic"}`,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Marshal and check extra fields
	data, err := json.Marshal(config.Inbounds[0])
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "cubic", raw["congestion_control"])
}

func TestGenerateConfig_Hysteria2BrutalMode(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 1)

	inbound := models.Inbound{
		Name:       "hy2-brutal-test",
		Protocol:   "hysteria2",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		ConfigJSON: `{"brutal_mode": true, "up_mbps": 200, "down_mbps": 500}`,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Marshal and check extra fields
	data, err := json.Marshal(config.Inbounds[0])
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Check brutal object exists and has correct structure
	brutal, ok := raw["brutal"].(map[string]interface{})
	require.True(t, ok, "brutal should be a map")
	assert.Equal(t, true, brutal["enabled"])
	assert.Equal(t, float64(200), brutal["send_mbps"])
	assert.Equal(t, float64(500), brutal["receive_mbps"])
}

func TestGenerateConfig_Hysteria2CongestionControlDefault(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 1)

	inbound := models.Inbound{
		Name:       "hy2-default-test",
		Protocol:   "hysteria2",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		ConfigJSON: `{}`,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Marshal and check extra fields
	data, err := json.Marshal(config.Inbounds[0])
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// congestion_control should NOT be auto-set if not in ConfigJSON
	_, exists := raw["congestion_control"]
	assert.False(t, exists, "congestion_control should not be auto-set when not in ConfigJSON")
}

func TestGenerateConfig_DefaultOutbound(t *testing.T) {
	db := setupTestDB(t)
	core, _ := createCoreAndUsers(t, db, 0)

	inbound := models.Inbound{
		Name:     "test",
		Protocol: "http",
		CoreID:   core.ID,
		Port:     8080,
	}
	require.NoError(t, db.Create(&inbound).Error)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)

	// Should have default direct outbound
	require.Len(t, config.Outbounds, 1)
	assert.Equal(t, "direct", config.Outbounds[0].Type)
	assert.Equal(t, "direct", config.Outbounds[0].Tag)
}

func TestGenerateConfig_WithOutbounds(t *testing.T) {
	db := setupTestDB(t)
	core, _ := createCoreAndUsers(t, db, 0)

	inbound := models.Inbound{
		Name:     "test",
		Protocol: "http",
		CoreID:   core.ID,
		Port:     8080,
	}
	require.NoError(t, db.Create(&inbound).Error)

	outbound := models.Outbound{
		Name:       "block-ads",
		Protocol:   "block",
		CoreID:     core.ID,
		ConfigJSON: `{"domain_suffix": ["ads.com", "tracker.com"]}`,
		IsEnabled:  true,
	}
	require.NoError(t, db.Create(&outbound).Error)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)

	// Block protocol should NOT create an outbound entry
	require.Len(t, config.Outbounds, 1)
	assert.Equal(t, "direct", config.Outbounds[0].Type)

	// Instead, it should create a route rule with action="block"
	require.Len(t, config.Route.Rules, 1)
	assert.Equal(t, "block", config.Route.Rules[0].Action)
	assert.Equal(t, []string{"ads.com", "tracker.com"}, config.Route.Rules[0].DomainSuffix)
}

func TestGenerateConfig_WIREndpoint(t *testing.T) {
	db := setupTestDB(t)
	core, _ := createCoreAndUsers(t, db, 0)

	inbound := models.Inbound{
		Name:     "test",
		Protocol: "http",
		CoreID:   core.ID,
		Port:     8080,
	}
	require.NoError(t, db.Create(&inbound).Error)

	// Create WARP account file
	tmpDir := t.TempDir()
	account := cores.WARPAccount{
		AccountID:   "test-account-id",
		DeviceID:    "test-device-id",
		PrivateKey:  "eCtXsJZ27+4PbhDkHnB923tkUn2Gj59wZw5wFA75MnU=",
		PublicKey:   "Cr8hWlKvtDt7nrvf+f0brNQQzabAqrjfBvas9pmowjo=",
		Token:       "test-token",
		IPv4Address: "172.16.0.2",
		IPv6Address: "2606:4700:110:8f77::1",
		ClientID:    "dGVzdA==",
	}
	accountData, err := json.MarshalIndent(account, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(tmpDir+"/warp_account.json", accountData, 0600)
	require.NoError(t, err)

	// Add WARP route
	route := models.WarpRoute{
		CoreID:        core.ID,
		ResourceType:  "domain",
		ResourceValue: "openai.com",
		Priority:      100,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(&route).Error)

	ctx := &cores.ConfigContext{
		DB:      db,
		WarpDir: tmpDir,
	}

	config, err := singbox.GenerateConfig(ctx, core.ID)
	require.NoError(t, err)

	// WARP should be in endpoints, not outbounds
	require.Len(t, config.Endpoints, 1)
	assert.Equal(t, "wireguard", config.Endpoints[0].Type)
	assert.Equal(t, "warp-out", config.Endpoints[0].Tag)

	// Verify route rule references warp-out
	require.Len(t, config.Route.Rules, 1)
	assert.Equal(t, "warp-out", config.Route.Rules[0].Outbound)
	assert.Equal(t, []string{"openai.com"}, config.Route.Rules[0].DomainSuffix)
}

func TestValidateConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := singbox.ValidateConfig(nil)
		assert.Error(t, err)
	})

	t.Run("empty inbounds", func(t *testing.T) {
		config := &singbox.Config{
			Inbounds: []singbox.InboundConfig{},
		}
		err := singbox.ValidateConfig(config)
		assert.Error(t, err)
	})

	t.Run("duplicate tags", func(t *testing.T) {
		config := &singbox.Config{
			Inbounds: []singbox.InboundConfig{
				{Tag: "dup", Type: "http"},
				{Tag: "dup", Type: "socks"},
			},
		}
		err := singbox.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("valid config", func(t *testing.T) {
		config := &singbox.Config{
			Inbounds: []singbox.InboundConfig{
				{Tag: "http_1", Type: "http"},
				{Tag: "vmess_2", Type: "vmess"},
			},
			Outbounds: []singbox.OutboundConfig{
				{Tag: "direct", Type: "direct"},
			},
		}
		err := singbox.ValidateConfig(config)
		assert.NoError(t, err)
	})
}

func TestWriteAndReadConfig(t *testing.T) {
	db := setupTestDB(t)
	core, users := createCoreAndUsers(t, db, 1)

	inbound := models.Inbound{
		Name:     "test-vless",
		Protocol: "vless",
		CoreID:   core.ID,
		Port:     443,
	}
	require.NoError(t, db.Create(&inbound).Error)
	assignUsers(t, db, inbound.ID, users)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)

	// Write to temp file
	tmpPath := t.TempDir() + "/config.json"
	err = singbox.WriteConfig(config, tmpPath)
	require.NoError(t, err)

	// Read back
	readConfig, err := singbox.ReadConfig(tmpPath)
	require.NoError(t, err)

	assert.Len(t, readConfig.Inbounds, len(config.Inbounds))
	assert.Len(t, readConfig.Outbounds, len(config.Outbounds))
	assert.Equal(t, config.Log.Level, readConfig.Log.Level)
}

func TestGenerateConfig_NoUsersAssigned(t *testing.T) {
	db := setupTestDB(t)
	core, _ := createCoreAndUsers(t, db, 0) // No users

	inbound := models.Inbound{
		Name:     "empty-vmess",
		Protocol: "vmess",
		CoreID:   core.ID,
		Port:     443,
	}
	require.NoError(t, db.Create(&inbound).Error)
	// No users assigned

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)
	require.Len(t, config.Inbounds, 1)

	// Users should be nil/empty
	assert.Nil(t, config.Inbounds[0].Users)
}

func TestGenerateConfig_CoreNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := singbox.GenerateConfig(testCtx(db), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get core")
}

func TestInboundMarshalJSON_ExtraFields(t *testing.T) {
	inbound := singbox.InboundConfig{
		Type:      "shadowsocks",
		Tag:       "ss_1",
		Listen:    "0.0.0.0",
		ListenPort: 8388,
		Extra: map[string]interface{}{
			"method":   "2022-blake3-aes-128-gcm",
			"password": "server-key",
		},
	}

	data, err := json.Marshal(inbound)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "shadowsocks", raw["type"])
	assert.Equal(t, "ss_1", raw["tag"])
	assert.Equal(t, "2022-blake3-aes-128-gcm", raw["method"])
	assert.Equal(t, "server-key", raw["password"])
	assert.Equal(t, float64(8388), raw["listen_port"])
}

func TestDNSServerMarshalJSON(t *testing.T) {
	t.Run("new format with type and server", func(t *testing.T) {
		server := singbox.DNSServer{
			Tag:    "google",
			Type:   "https",
			Server: "dns.google",
		}

		data, err := json.Marshal(server)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.Equal(t, "google", raw["tag"])
		assert.Equal(t, "https", raw["type"])
		assert.Equal(t, "dns.google", raw["server"])
	})

	t.Run("local type without server", func(t *testing.T) {
		server := singbox.DNSServer{
			Tag:  "local",
			Type: "local",
		}

		data, err := json.Marshal(server)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.Equal(t, "local", raw["tag"])
		assert.Equal(t, "local", raw["type"])
		_, hasServer := raw["server"]
		assert.False(t, hasServer)
	})

	t.Run("backward compatibility with address field", func(t *testing.T) {
		server := singbox.DNSServer{
			Tag:     "google",
			Address: "https://dns.google/dns-query",
		}

		data, err := json.Marshal(server)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		// Should be converted to new format
		assert.Equal(t, "google", raw["tag"])
		assert.Equal(t, "https", raw["type"])
		assert.Equal(t, "dns.google", raw["server"])
	})

	t.Run("backward compatibility with local address", func(t *testing.T) {
		server := singbox.DNSServer{
			Tag:     "local",
			Address: "local",
		}

		data, err := json.Marshal(server)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.Equal(t, "local", raw["tag"])
		assert.Equal(t, "local", raw["type"])
	})

	t.Run("with extra fields", func(t *testing.T) {
		server := singbox.DNSServer{
			Tag:    "custom",
			Type:   "https",
			Server: "custom.dns",
			Extra: map[string]interface{}{
				"detour": "warp-out",
			},
		}

		data, err := json.Marshal(server)
		require.NoError(t, err)

		var raw map[string]interface{}
		err = json.Unmarshal(data, &raw)
		require.NoError(t, err)

		assert.Equal(t, "custom", raw["tag"])
		assert.Equal(t, "https", raw["type"])
		assert.Equal(t, "custom.dns", raw["server"])
		assert.Equal(t, "warp-out", raw["detour"])
	})
}

func TestOutboundConfig_DomainResolver(t *testing.T) {
	outbound := singbox.OutboundConfig{
		Type:           "direct",
		Tag:            "direct-out",
		DomainResolver: "google",
		Extra: map[string]interface{}{
			"detour": "warp-out",
		},
	}

	data, err := json.Marshal(outbound)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "direct", raw["type"])
	assert.Equal(t, "direct-out", raw["tag"])
	assert.Equal(t, "google", raw["domain_resolver"])
	assert.Equal(t, "warp-out", raw["detour"])
}

func TestOutboundConfig_DomainResolverEmpty(t *testing.T) {
	outbound := singbox.OutboundConfig{
		Type:  "direct",
		Tag:   "direct-out",
		Extra: map[string]interface{}{
			"detour": "warp-out",
		},
	}

	data, err := json.Marshal(outbound)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "direct", raw["type"])
	assert.Equal(t, "direct-out", raw["tag"])
	_, hasDomainResolver := raw["domain_resolver"]
	assert.False(t, hasDomainResolver, "domain_resolver should be omitted when empty")
	assert.Equal(t, "warp-out", raw["detour"])
}

func TestGenerateConfig_OutboundWithDomainResolver(t *testing.T) {
	db := setupTestDB(t)
	core, _ := createCoreAndUsers(t, db, 0)

	inbound := models.Inbound{
		Name:     "test",
		Protocol: "http",
		CoreID:   core.ID,
		Port:     8080,
	}
	require.NoError(t, db.Create(&inbound).Error)

	outbound := models.Outbound{
		Name:       "proxy-out",
		Protocol:   "direct",
		CoreID:     core.ID,
		ConfigJSON: `{"domain_resolver": "google"}`,
		IsEnabled:  true,
	}
	require.NoError(t, db.Create(&outbound).Error)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)

	require.Len(t, config.Outbounds, 1)
	assert.Equal(t, "direct", config.Outbounds[0].Type)
	assert.Equal(t, "google", config.Outbounds[0].DomainResolver)

	data, err := json.Marshal(config.Outbounds[0])
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "google", raw["domain_resolver"])
}

func TestGenerateConfig_DNSOutboundAsAction(t *testing.T) {
	db := setupTestDB(t)
	core, _ := createCoreAndUsers(t, db, 0)

	inbound := models.Inbound{
		Name:     "test",
		Protocol: "http",
		CoreID:   core.ID,
		Port:     8080,
	}
	require.NoError(t, db.Create(&inbound).Error)

	outbound := models.Outbound{
		Name:       "dns-outbound",
		Protocol:   "dns",
		CoreID:     core.ID,
		ConfigJSON: `{"ip_cidr": ["1.1.1.0/24", "8.8.8.0/24"]}`,
		IsEnabled:  true,
	}
	require.NoError(t, db.Create(&outbound).Error)

	config, err := singbox.GenerateConfig(testCtx(db), core.ID)
	require.NoError(t, err)

	// DNS protocol should NOT create an outbound entry
	require.Len(t, config.Outbounds, 1)
	assert.Equal(t, "direct", config.Outbounds[0].Type)

	// Instead, it should create a route rule with action="dns"
	require.Len(t, config.Route.Rules, 1)
	assert.Equal(t, "dns", config.Route.Rules[0].Action)
	assert.Equal(t, []string{"1.1.1.0/24", "8.8.8.0/24"}, config.Route.Rules[0].IPCIDR)
}

func TestMapSingboxOutboundProtocol_BlockRejected(t *testing.T) {
	_, err := singbox.MapSingboxOutboundProtocol("block")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must use route rule action instead of outbound type")
}

func TestMapSingboxOutboundProtocol_DNSRejected(t *testing.T) {
	_, err := singbox.MapSingboxOutboundProtocol("dns")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must use route rule action instead of outbound type")
}
