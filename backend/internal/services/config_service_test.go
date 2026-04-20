package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	_ "github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	_ "github.com/isolate-project/isolate-panel/internal/cores/singbox"
	_ "github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/models"
)

func setupConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.Core{},
		&models.Inbound{},
		&models.Outbound{},
		&models.User{},
		&models.UserInboundMapping{},
		&models.GeoRule{},
		&models.WarpRoute{},
	))
	return db
}

func seedXrayCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "xray", Version: "1.8.0", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func seedSingboxCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "singbox", Version: "1.5.0", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func seedMihomoCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "mihomo", Version: "1.16.0", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func seedUser(t *testing.T, db *gorm.DB, username, uuid string) models.User {
	t.Helper()
	user := models.User{
		Username:          username,
		UUID:              uuid,
		IsActive:          true,
		SubscriptionToken: username + "-token",
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func seedInbound(t *testing.T, db *gorm.DB, name, protocol string, coreID uint, port int) models.Inbound {
	t.Helper()
	inbound := models.Inbound{
		Name:          name,
		Protocol:      protocol,
		CoreID:        coreID,
		ListenAddress: "0.0.0.0",
		Port:          port,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(&inbound).Error)
	return inbound
}

func seedOutbound(t *testing.T, db *gorm.DB, name, protocol string, coreID uint) models.Outbound {
	t.Helper()
	outbound := models.Outbound{
		Name:       name,
		Protocol:   protocol,
		CoreID:     coreID,
		ConfigJSON: `{"tag":"` + name + `"}`,
		IsEnabled:  true,
	}
	require.NoError(t, db.Create(&outbound).Error)
	return outbound
}

func newSupervisorStubForConfig(t *testing.T, running bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)

		w.WriteHeader(200)
		if strings.Contains(bodyStr, "getProcessInfo") {
			state := 0
			stateName := "STOPPED"
			if running {
				state = 20
				stateName = "RUNNING"
			}
			w.Write([]byte(fmt.Sprintf(`<?xml version="1.0"?><methodResponse><params><param><value><struct>`+
				`<member><name>name</name><value><string>xray</string></value></member>`+
				`<member><name>state</name><value><int>%d</int></value></member>`+
				`<member><name>statename</name><value><string>%s</string></value></member>`+
				`<member><name>pid</name><value><int>1234</int></value></member>`+
				`<member><name>start</name><value><int>1000</int></value></member>`+
				`<member><name>stop</name><value><int>0</int></value></member>`+
				`<member><name>now</name><value><int>2000</int></value></member>`+
				`</struct></value></param></params></methodResponse>`, state, stateName)))
		} else {
			w.Write([]byte(`<?xml version="1.0"?><methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`))
		}
	}))
}

func TestNewConfigService(t *testing.T) {
	db := setupConfigTestDB(t)
	svc := NewConfigService(db, nil, "/tmp/test-config", "secret123")
	assert.NotNil(t, svc)
}

func TestNewConfigService_DefaultConfigDir(t *testing.T) {
	db := setupConfigTestDB(t)
	svc := NewConfigService(db, nil, "", "secret")
	assert.Equal(t, "./data/cores", svc.configDir)
}

func TestConfigService_SetV2RayAPIListenAddr(t *testing.T) {
	db := setupConfigTestDB(t)
	svc := NewConfigService(db, nil, "/tmp", "secret")
	svc.SetV2RayAPIListenAddr("127.0.0.1:10086")
	assert.Equal(t, "127.0.0.1:10086", svc.v2rayAPIListenAddr)
}

func TestConfigService_RegenerateConfig_CoreNotFound(t *testing.T) {
	db := setupConfigTestDB(t)
	svc := NewConfigService(db, nil, t.TempDir(), "secret")
	err := svc.RegenerateConfig("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "core not found")
}

func TestConfigService_RegenerateConfig_UnknownCore(t *testing.T) {
	db := setupConfigTestDB(t)
	core := models.Core{Name: "weirdcore", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	svc := NewConfigService(db, nil, t.TempDir(), "secret")
	err := svc.RegenerateConfig("weirdcore")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get core adapter")
}

func TestConfigService_RegenerateConfig_Xray(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedXrayCore(t, db)
	user := seedUser(t, db, "testuser", "uuid-xray-001")
	inbound := seedInbound(t, db, "test-vmess", "vmess", core.ID, 443)

	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID: user.ID, InboundID: inbound.ID,
	}).Error)

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")

	err := svc.RegenerateConfig("xray")
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "xray", "config.json")
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	assert.NotNil(t, config["api"])
	assert.NotNil(t, config["inbounds"])
}

func TestConfigService_RegenerateConfig_Singbox(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedSingboxCore(t, db)
	user := seedUser(t, db, "sbuser", "uuid-sb-001")
	inbound := seedInbound(t, db, "sb-vmess", "vmess", core.ID, 443)

	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID: user.ID, InboundID: inbound.ID,
	}).Error)

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")

	err := svc.RegenerateConfig("singbox")
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "singbox", "config.json")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	assert.NotNil(t, config["inbounds"])
}

func TestConfigService_RegenerateConfig_Mihomo(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedMihomoCore(t, db)
	user := seedUser(t, db, "mhomouser", "uuid-mh-001")
	inbound := seedInbound(t, db, "mh-in", "mixed", core.ID, 7890)

	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID: user.ID, InboundID: inbound.ID,
	}).Error)

	seedOutbound(t, db, "mh-direct", "direct", core.ID)

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")

	err := svc.RegenerateConfig("mihomo")
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "mihomo", "config.yaml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestConfigService_RegenerateConfig_Xray_WithOutbounds(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedXrayCore(t, db)
	user := seedUser(t, db, "obuser", "uuid-ob-001")
	inbound := seedInbound(t, db, "ob-vmess", "vmess", core.ID, 443)
	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID: user.ID, InboundID: inbound.ID,
	}).Error)
	seedOutbound(t, db, "direct-out", "freedom", core.ID)

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")
	require.NoError(t, svc.RegenerateConfig("xray"))

	data, err := os.ReadFile(filepath.Join(configDir, "xray", "config.json"))
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	assert.NotNil(t, config["outbounds"])
}

func TestConfigService_RegenerateConfig_Singbox_WithMultipleInbounds(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedSingboxCore(t, db)
	user1 := seedUser(t, db, "multi-u1", "uuid-m1")
	user2 := seedUser(t, db, "multi-u2", "uuid-m2")

	in1 := seedInbound(t, db, "multi-vmess", "vmess", core.ID, 443)
	in2 := seedInbound(t, db, "multi-vless", "vless", core.ID, 8443)

	for _, u := range []models.User{user1, user2} {
		for _, in := range []models.Inbound{in1, in2} {
			require.NoError(t, db.Create(&models.UserInboundMapping{
				UserID: u.ID, InboundID: in.ID,
			}).Error)
		}
	}

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")
	require.NoError(t, svc.RegenerateConfig("singbox"))

	data, err := os.ReadFile(filepath.Join(configDir, "singbox", "config.json"))
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	inbounds, ok := config["inbounds"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(inbounds), 2)
}

func TestConfigService_RegenerateAndReload_CoreRunning(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedXrayCore(t, db)
	user := seedUser(t, db, "reloaduser", "uuid-reload-001")
	inbound := seedInbound(t, db, "reload-vmess", "vmess", core.ID, 443)
	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID: user.ID, InboundID: inbound.ID,
	}).Error)

	srv := newSupervisorStubForConfig(t, true)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	configDir := t.TempDir()
	svc := NewConfigService(db, coreMgr, configDir, "test-secret")

	err := svc.RegenerateAndReload("xray")
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "xray", "config.json")
	_, statErr := os.Stat(configPath)
	assert.NoError(t, statErr)
}

func TestConfigService_RegenerateAndReload_CoreNotRunning(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedXrayCore(t, db)
	user := seedUser(t, db, "stoppeduser", "uuid-stopped-001")
	inbound := seedInbound(t, db, "stopped-vmess", "vmess", core.ID, 443)
	require.NoError(t, db.Create(&models.UserInboundMapping{
		UserID: user.ID, InboundID: inbound.ID,
	}).Error)

	srv := newSupervisorStubForConfig(t, false)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	configDir := t.TempDir()
	svc := NewConfigService(db, coreMgr, configDir, "test-secret")

	err := svc.RegenerateAndReload("xray")
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "xray", "config.json")
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestConfigService_RegenerateAndReload_CoreNotFound(t *testing.T) {
	db := setupConfigTestDB(t)
	srv := newSupervisorStubForConfig(t, false)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	svc := NewConfigService(db, coreMgr, t.TempDir(), "secret")

	err := svc.RegenerateAndReload("nonexistent")
	assert.Error(t, err)
}

func TestConfigService_RegenerateConfig_NoInbounds(t *testing.T) {
	db := setupConfigTestDB(t)
	_ = seedXrayCore(t, db)

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")

	err := svc.RegenerateConfig("xray")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(configDir, "xray", "config.json"))
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	inbounds, ok := config["inbounds"].([]interface{})
	require.True(t, ok)
	assert.Len(t, inbounds, 1)
}

func TestConfigService_RegenerateConfig_DisabledInbound(t *testing.T) {
	db := setupConfigTestDB(t)
	core := seedXrayCore(t, db)

	disabledInbound := models.Inbound{
		Name:          "disabled-in",
		Protocol:      "vmess",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          443,
		IsEnabled:     false,
	}
	require.NoError(t, db.Create(&disabledInbound).Error)

	configDir := t.TempDir()
	svc := NewConfigService(db, nil, configDir, "test-secret")

	err := svc.RegenerateConfig("xray")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(configDir, "xray", "config.json"))
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	inbounds, ok := config["inbounds"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(inbounds), 2)
}
