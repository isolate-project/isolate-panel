package cores_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/cores"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Core{}, &models.Inbound{}, &models.Outbound{},
		&models.User{}, &models.UserInboundMapping{},
		&models.WarpRoute{}, &models.GeoRule{},
	)
	require.NoError(t, err)

	return db
}

func createTestWARPAccount(t *testing.T, dir string) {
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

	data, err := json.MarshalIndent(account, "", "  ")
	require.NoError(t, err)

	err = os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "warp_account.json"), data, 0600)
	require.NoError(t, err)
}

// ============================================================
// WARP Integration Tests
// ============================================================

func TestLoadWARPOutbound_NoAccount(t *testing.T) {
	db := setupTestDB(t)
	tmpDir := t.TempDir()

	ctx := &cores.ConfigContext{
		DB:      db,
		WarpDir: tmpDir,
	}

	// No warp_account.json → should return nil, nil
	data, err := cores.LoadWARPOutbound(ctx, 1)
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestLoadWARPOutbound_NoRoutes(t *testing.T) {
	db := setupTestDB(t)
	tmpDir := t.TempDir()
	createTestWARPAccount(t, tmpDir)

	core := models.Core{Name: "singbox", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)

	ctx := &cores.ConfigContext{
		DB:      db,
		WarpDir: tmpDir,
	}

	// Account exists but no routes → should return nil
	data, err := cores.LoadWARPOutbound(ctx, core.ID)
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestLoadWARPOutbound_WithRoutes(t *testing.T) {
	db := setupTestDB(t)
	tmpDir := t.TempDir()
	createTestWARPAccount(t, tmpDir)

	core := models.Core{Name: "singbox", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)

	// Add routes
	routes := []models.WarpRoute{
		{CoreID: core.ID, ResourceType: "domain", ResourceValue: "openai.com", Priority: 100, IsEnabled: true},
		{CoreID: core.ID, ResourceType: "ip", ResourceValue: "1.2.3.4", Priority: 50, IsEnabled: true},
		{CoreID: core.ID, ResourceType: "cidr", ResourceValue: "10.0.0.0/8", Priority: 30, IsEnabled: true},
		{CoreID: core.ID, ResourceType: "domain", ResourceValue: "disabled.com", Priority: 10, IsEnabled: true},
	}
	for _, r := range routes {
		require.NoError(t, db.Create(&r).Error)
	}
	// Disable one route explicitly (GORM treats false as zero-value with default:true)
	require.NoError(t, db.Model(&models.WarpRoute{}).Where("resource_value = ?", "disabled.com").Update("is_enabled", false).Error)

	ctx := &cores.ConfigContext{
		DB:      db,
		WarpDir: tmpDir,
	}

	data, err := cores.LoadWARPOutbound(ctx, core.ID)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Only 3 enabled routes (disabled one is skipped)
	assert.Len(t, data.Routes, 3)
	assert.Equal(t, "test-account-id", data.Account.AccountID)
	assert.Equal(t, "172.16.0.2", data.Account.IPv4Address)
}

func TestSingboxWARPOutbound(t *testing.T) {
	account := &cores.WARPAccount{
		PrivateKey:  "test-key",
		IPv4Address: "172.16.0.99",
		IPv6Address: "fd01::1",
	}

	outbound := cores.SingboxWARPOutbound(account)

	assert.Equal(t, "wireguard", outbound["type"])
	assert.Equal(t, "warp-out", outbound["tag"])
	assert.Equal(t, 2408, outbound["server_port"])
	assert.Equal(t, "test-key", outbound["private_key"])

	localAddr := outbound["local_address"].([]string)
	assert.Contains(t, localAddr, "172.16.0.99/32")
	assert.Contains(t, localAddr, "fd01::1/128")
}

func TestXrayWARPOutbound(t *testing.T) {
	account := &cores.WARPAccount{
		PrivateKey:  "test-key",
		IPv4Address: "10.0.0.1",
	}

	tag, protocol, settings := cores.XrayWARPOutbound(account)
	assert.Equal(t, "warp-out", tag)
	assert.Equal(t, "wireguard", protocol)

	var s map[string]interface{}
	err := json.Unmarshal(settings, &s)
	require.NoError(t, err)
	assert.Equal(t, "test-key", s["secretKey"])

	addresses := s["address"].([]interface{})
	assert.Contains(t, addresses, "10.0.0.1/32")
}

func TestMihomoWARPProxy(t *testing.T) {
	account := &cores.WARPAccount{
		PrivateKey:  "test-key",
		IPv4Address: "172.16.0.2",
		IPv6Address: "fd01::1",
	}

	proxy := cores.MihomoWARPProxy(account)
	assert.Equal(t, "warp-out", proxy["name"])
	assert.Equal(t, "wireguard", proxy["type"])
	assert.Equal(t, "172.16.0.2", proxy["ip"])
	assert.Equal(t, "fd01::1", proxy["ipv6"])
	assert.Equal(t, true, proxy["udp"])
}

func TestSingboxWARPRouteRules(t *testing.T) {
	routes := []models.WarpRoute{
		{ResourceType: "domain", ResourceValue: "openai.com"},
		{ResourceType: "ip", ResourceValue: "1.2.3.4"},
		{ResourceType: "cidr", ResourceValue: "10.0.0.0/8"},
	}

	rules := cores.SingboxWARPRouteRules(routes)
	assert.Len(t, rules, 3)
	assert.Equal(t, "warp-out", rules[0]["outbound"])
	assert.Equal(t, []string{"openai.com"}, rules[0]["domain_suffix"])
	assert.Equal(t, []string{"1.2.3.4/32"}, rules[1]["ip_cidr"])
	assert.Equal(t, []string{"10.0.0.0/8"}, rules[2]["ip_cidr"])
}

func TestMihomoWARPRules(t *testing.T) {
	routes := []models.WarpRoute{
		{ResourceType: "domain", ResourceValue: "openai.com"},
		{ResourceType: "ip", ResourceValue: "1.2.3.4"},
		{ResourceType: "cidr", ResourceValue: "10.0.0.0/8"},
	}

	rules := cores.MihomoWARPRules(routes)
	assert.Len(t, rules, 3)
	assert.Equal(t, "DOMAIN-SUFFIX,openai.com,warp-out", rules[0])
	assert.Equal(t, "IP-CIDR,1.2.3.4/32,warp-out", rules[1])
	assert.Equal(t, "IP-CIDR,10.0.0.0/8,warp-out", rules[2])
}

// ============================================================
// Geo Integration Tests
// ============================================================

func TestLoadGeoRules_NoRules(t *testing.T) {
	db := setupTestDB(t)

	ctx := &cores.ConfigContext{DB: db}
	data, err := cores.LoadGeoRules(ctx, 1)
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestLoadGeoRules_WithRules(t *testing.T) {
	db := setupTestDB(t)

	core := models.Core{Name: "singbox", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)

	rules := []models.GeoRule{
		{CoreID: core.ID, Type: "geoip", Code: "US", Action: "direct", Priority: 100, IsEnabled: true},
		{CoreID: core.ID, Type: "geosite", Code: "google", Action: "warp", Priority: 50, IsEnabled: true},
		{CoreID: core.ID, Type: "geoip", Code: "CN", Action: "block", Priority: 30, IsEnabled: true},
	}
	for _, r := range rules {
		require.NoError(t, db.Create(&r).Error)
	}
	// Disable one rule explicitly
	require.NoError(t, db.Model(&models.GeoRule{}).Where("code = ?", "CN").Update("is_enabled", false).Error)

	ctx := &cores.ConfigContext{DB: db}
	data, err := cores.LoadGeoRules(ctx, core.ID)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Only 2 enabled rules
	assert.Len(t, data.Rules, 2)
}

func TestSingboxGeoRouteRules(t *testing.T) {
	rules := []models.GeoRule{
		{Type: "geoip", Code: "US", Action: "direct"},
		{Type: "geosite", Code: "google", Action: "warp"},
	}

	routeRules := cores.SingboxGeoRouteRules(rules, "/data/geo")
	assert.Len(t, routeRules, 2)
	assert.Equal(t, "direct", routeRules[0]["outbound"])
	assert.Equal(t, "US", routeRules[0]["geoip"])
	assert.Equal(t, "warp-out", routeRules[1]["outbound"])
	assert.Equal(t, "google", routeRules[1]["geosite"])
}

func TestXrayGeoRoutingRules(t *testing.T) {
	rules := []models.GeoRule{
		{Type: "geoip", Code: "CN", Action: "block"},
		{Type: "geosite", Code: "category-ads", Action: "block"},
	}

	routingRules := cores.XrayGeoRoutingRules(rules)
	assert.Len(t, routingRules, 2)
	assert.Equal(t, "field", routingRules[0]["type"])
	assert.Equal(t, "block", routingRules[0]["outboundTag"])
	assert.Equal(t, []string{"geoip:CN"}, routingRules[0]["ip"])
	assert.Equal(t, []string{"geosite:category-ads"}, routingRules[1]["domain"])
}

func TestMihomoGeoRules(t *testing.T) {
	rules := []models.GeoRule{
		{Type: "geoip", Code: "US", Action: "direct"},
		{Type: "geosite", Code: "google", Action: "warp"},
		{Type: "geoip", Code: "CN", Action: "block"},
	}

	mihomoRules := cores.MihomoGeoRules(rules)
	assert.Len(t, mihomoRules, 3)
	assert.Equal(t, "GEOIP,US,DIRECT", mihomoRules[0])
	assert.Equal(t, "GEOSITE,google,warp-out", mihomoRules[1])
	assert.Equal(t, "GEOIP,CN,REJECT", mihomoRules[2])
}

func TestInjectWARP_GracefulSkip(t *testing.T) {
	db := setupTestDB(t)
	ctx := &cores.ConfigContext{DB: db, WarpDir: ""}

	data, ok := cores.InjectWARP(ctx, 1)
	assert.False(t, ok)
	assert.Nil(t, data)
}

func TestInjectGeo_GracefulSkip(t *testing.T) {
	db := setupTestDB(t)
	ctx := &cores.ConfigContext{DB: db}

	data, ok := cores.InjectGeo(ctx, 999)
	assert.False(t, ok)
	assert.Nil(t, data)
}

func TestSingboxGeoAssets(t *testing.T) {
	assets := cores.SingboxGeoAssets("/data/geo")
	require.NotNil(t, assets)
	assert.Equal(t, "/data/geo/geoip.db", assets["geoip"])
	assert.Equal(t, "/data/geo/geosite.db", assets["geosite"])

	// Empty dir returns nil
	assert.Nil(t, cores.SingboxGeoAssets(""))
}
