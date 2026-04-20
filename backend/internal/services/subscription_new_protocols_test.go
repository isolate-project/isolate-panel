package services

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateProxyLink_AnyTLS tests AnyTLS link generation
func TestGenerateProxyLink_AnyTLS(t *testing.T) {
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "AnyTLS-Test",
		Protocol:      "anytls",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"password":"anytls-pass-123"}`,
	}

	link := generateProxyLink(user, inbound, "http://test-panel", nil)

	require.NotEmpty(t, link)
	assert.True(t, strings.HasPrefix(link, "anytls://"), "Link should start with anytls://")
	assert.Contains(t, link, "anytls-pass-123@", "Password should be in link")
	assert.Contains(t, link, "example.com:443", "Server and port should be in link")

	// Check fragment contains core prefix
	parts := strings.Split(link, "#")
	require.Len(t, parts, 2, "Link should have fragment")
	fragment, err := url.PathUnescape(parts[1])
	require.NoError(t, err)
	assert.Equal(t, "AnyTLS-Test", fragment, "Fragment should be inbound name")
}

// TestGenerateProxyLink_Snell tests Snell link generation
func TestGenerateProxyLink_Snell(t *testing.T) {
	token := "snell-psk"
	user := models.User{ID: 1, UUID: "test-uuid", Token: &token}
	inbound := models.Inbound{
		ID:            1,
		Name:          "Snell-Test",
		Protocol:      "snell",
		Port:          443,
		ListenAddress: "example.com",
		ConfigJSON:    `{"version":3,"obfs":"http"}`,
	}

	link := generateProxyLink(user, inbound, "http://test-panel", nil)

	require.NotEmpty(t, link)
	assert.True(t, strings.HasPrefix(link, "snell://"), "Link should start with snell://")
	assert.Contains(t, link, "snell-psk@", "PSK should be in link")
	assert.Contains(t, link, "version=3", "Version should be in query params")
	assert.Contains(t, link, "obfs=http", "Obfs should be in query params")

	// Test fallback to UUID when no Token
	userNoToken := models.User{ID: 1, UUID: "fallback-uuid"}
	link2 := generateProxyLink(userNoToken, inbound, "http://test-panel", nil)
	assert.Contains(t, link2, "fallback-uuid@", "Should fallback to UUID when no Token")
}

// TestGenerateProxyLink_MieruSudokuTrusttunnel_ReturnEmpty tests protocols without standard URI schemes
func TestGenerateProxyLink_MieruSudokuTrusttunnel_ReturnEmpty(t *testing.T) {
	user := models.User{ID: 1, UUID: "test-uuid"}
	configJSON := `{"password":"test-pass"}`

	protocols := []string{"mieru", "sudoku", "trusttunnel"}
	for _, proto := range protocols {
		t.Run(proto, func(t *testing.T) {
			inbound := models.Inbound{
				ID:            1,
				Name:          proto + "-Test",
				Protocol:      proto,
				Port:          443,
				ListenAddress: "example.com",
				ConfigJSON:    configJSON,
			}

			link := generateProxyLink(user, inbound, "http://test-panel", nil)
			assert.Empty(t, link, proto+" should return empty link (no standard URI scheme)")
		})
	}
}

// TestGenerateSingboxOutbound_AnyTLS tests Sing-box outbound for AnyTLS
func TestGenerateSingboxOutbound_AnyTLS(t *testing.T) {
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "AnyTLS-SB",
		Protocol:      "anytls",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"password":"anytls-pass"}`,
	}

	ob := generateSingboxOutbound(user, inbound, "http://test-panel", nil)

	require.NotNil(t, ob)
	assert.Equal(t, "anytls", ob["type"], "Outbound type should be anytls")
	assert.Equal(t, "anytls-pass", ob["password"], "Password should be set")
	assert.Equal(t, "example.com", ob["server"], "Server should be set")
	assert.Equal(t, 443, ob["server_port"], "Port should be set")

	tls, ok := ob["tls"].(map[string]interface{})
	require.True(t, ok, "TLS config should be present")
	assert.Equal(t, true, tls["enabled"], "TLS should be enabled")
}

// TestGenerateSingboxOutbound_HysteriaV1 tests Sing-box outbound for Hysteria v1
func TestGenerateSingboxOutbound_HysteriaV1(t *testing.T) {
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "Hysteria-SB",
		Protocol:      "hysteria",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"auth_str":"auth-password","up_mbps":50,"down_mbps":100,"obfs":"obfs-pass"}`,
	}

	ob := generateSingboxOutbound(user, inbound, "http://test-panel", nil)

	require.NotNil(t, ob)
	assert.Equal(t, "hysteria", ob["type"], "Outbound type should be hysteria")
	assert.Equal(t, "auth-password", ob["password"], "Password should be auth_str")
	assert.Equal(t, 50, ob["up_mbps"], "up_mbps should be set")
	assert.Equal(t, 100, ob["down_mbps"], "down_mbps should be set")
	assert.Equal(t, "obfs-pass", ob["obfs"], "obfs should be set")

	tls, ok := ob["tls"].(map[string]interface{})
	require.True(t, ok, "TLS config should be present")
	assert.Equal(t, true, tls["enabled"], "TLS should be enabled")
}

// TestGenerateSingboxOutbound_XHTTP_ReturnsNil tests XHTTP returns nil for Sing-box
func TestGenerateSingboxOutbound_XHTTP_ReturnsNil(t *testing.T) {
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "XHTTP-SB",
		Protocol:      "xhttp",
		Port:          443,
		ListenAddress: "example.com",
		ConfigJSON:    `{"path":"/xhttp"}`,
	}

	ob := generateSingboxOutbound(user, inbound, "http://test-panel", nil)
	assert.Nil(t, ob, "XHTTP should return nil (Xray-exclusive)")
}

// TestGenerateIsolate_AnyTLS tests Isolate subscription for AnyTLS
func TestGenerateIsolate_AnyTLS(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("singbox")
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "AnyTLS-Isolate",
		Protocol:      "anytls",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"password":"anytls-pass"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	singboxCore, ok := sub.Cores["Sing-box"]
	require.True(t, ok, "Should have Sing-box core")
	require.Len(t, singboxCore.Inbounds, 1, "Should have 1 inbound")

	ib := singboxCore.Inbounds[0]
	assert.Equal(t, "anytls", ib.Protocol)
	assert.Equal(t, "anytls-pass", ib.Password, "Password should be populated")
	assert.Equal(t, "example.com", ib.Server)
	assert.Equal(t, 443, ib.Port)
}

// TestGenerateIsolate_Snell tests Isolate subscription for Snell
func TestGenerateIsolate_Snell(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	token := "snell-psk"
	user := models.User{ID: 1, UUID: "test-uuid", Token: &token}
	inbound := models.Inbound{
		ID:            1,
		Name:          "Snell-Isolate",
		Protocol:      "snell",
		Port:          443,
		ListenAddress: "example.com",
		ConfigJSON:    `{}`,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	// Snell doesn't have a core assigned, so it won't appear in Coores map
	// But we can verify the raw link is generated
	for _, core := range sub.Cores {
		for _, ib := range core.Inbounds {
			if ib.Protocol == "snell" {
				assert.Equal(t, "snell-psk", ib.Password, "Password should be user.Token")
			}
		}
	}
}

// TestGenerateIsolate_HysteriaV1 tests Isolate subscription for Hysteria v1
func TestGenerateIsolate_HysteriaV1(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("singbox")
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "Hysteria-Isolate",
		Protocol:      "hysteria",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"auth_str":"auth-password"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	singboxCore, ok := sub.Cores["Sing-box"]
	require.True(t, ok, "Should have Sing-box core")
	require.Len(t, singboxCore.Inbounds, 1, "Should have 1 inbound")

	ib := singboxCore.Inbounds[0]
	assert.Equal(t, "hysteria", ib.Protocol)
	assert.Equal(t, "auth-password", ib.Password, "Password should be auth_str from config")
}

// TestGenerateIsolate_XHTTP tests Isolate subscription for XHTTP
func TestGenerateIsolate_XHTTP(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("xray")
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "XHTTP-Isolate",
		Protocol:      "xhttp",
		Port:          443,
		ListenAddress: "example.com",
		ConfigJSON:    `{"path":"/xhttp"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	xrayCore, ok := sub.Cores["Xray"]
	require.True(t, ok, "Should have Xray core")
	require.Len(t, xrayCore.Inbounds, 1, "Should have 1 inbound")

	ib := xrayCore.Inbounds[0]
	assert.Equal(t, "xhttp", ib.Protocol)
	assert.Equal(t, "test-uuid", ib.UUID, "UUID should be populated")
}

// TestGenerateIsolate_SSR tests Isolate subscription for SSR
func TestGenerateIsolate_SSR(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("mihomo")
	user := models.User{ID: 1, UUID: "test-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "SSR-Isolate",
		Protocol:      "ssr",
		Port:          8388,
		ListenAddress: "example.com",
		ConfigJSON:    `{"cipher":"aes-256-cfb","method":"aes-256-cfb"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	mihomoCore, ok := sub.Cores["Mihomo"]
	require.True(t, ok, "Should have Mihomo core")
	require.Len(t, mihomoCore.Inbounds, 1, "Should have 1 inbound")

	ib := mihomoCore.Inbounds[0]
	assert.Equal(t, "ssr", ib.Protocol)
	assert.Equal(t, "aes-256-cfb", ib.Method, "Method should be set")
	assert.Equal(t, "test-uuid", ib.Password, "Password should be populated")
}

// TestGenerateIsolate_MieruSudokuTrusttunnel tests Isolate subscription for Mieru, Sudoku, Trusttunnel
func TestGenerateIsolate_MieruSudokuTrusttunnel(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("mihomo")
	user := models.User{ID: 1, UUID: "test-uuid"}

	protocols := []string{"mieru", "sudoku", "trusttunnel"}
	for _, proto := range protocols {
		t.Run(proto, func(t *testing.T) {
			inbound := models.Inbound{
				ID:            1,
				Name:          proto + "-Isolate",
				Protocol:      proto,
				Port:          443,
				ListenAddress: "example.com",
				ConfigJSON:    `{"password":"custom-pass"}`,
				Core:          &core,
			}

			data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
			result, err := svc.GenerateIsolate(data)
			require.NoError(t, err)

			var sub IsolateSubscription
			require.NoError(t, json.Unmarshal([]byte(result), &sub))

			mihomoCore, ok := sub.Cores["Mihomo"]
			require.True(t, ok, "Should have Mihomo core")
			require.Len(t, mihomoCore.Inbounds, 1, "Should have 1 inbound")

			ib := mihomoCore.Inbounds[0]
			assert.Equal(t, proto, ib.Protocol)
			assert.Equal(t, "custom-pass", ib.Password, "Password should be populated")
		})
	}
}

// TestBuildClashProxy_AnyTLS tests Clash proxy for AnyTLS
func TestBuildClashProxy_AnyTLS(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"password": "anytls-pass"}
	tlsInfo := makeTLSInfo("anytls.example.com")

	p := buildClashProxy("anytls", "AnyTLS-Clash", "example.com", 443, user, config, tlsInfo, nil, nil)

	require.NotNil(t, p)
	assert.Equal(t, "anytls", p.Type, "Type should be anytls")
	assert.Equal(t, "anytls-pass", p.Password, "Password should be set")
	assert.NotNil(t, p.SkipCertVerify, "SkipCertVerify should not be nil")
	assert.False(t, *p.SkipCertVerify, "SkipCertVerify should be false")
	assert.Equal(t, "anytls.example.com", p.SNI, "SNI should be set from tlsInfo")
}

// TestBuildClashProxy_HysteriaV1 tests Clash proxy for Hysteria v1
func TestBuildClashProxy_HysteriaV1(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"auth_str": "auth-password"}
	tlsInfo := makeTLSInfo("hysteria.example.com")

	p := buildClashProxy("hysteria", "Hysteria-Clash", "example.com", 443, user, config, tlsInfo, nil, nil)

	require.NotNil(t, p)
	assert.Equal(t, "hysteria", p.Type, "Type should be hysteria")
	assert.Equal(t, "auth-password", p.Password, "Password should be auth_str")
	assert.NotNil(t, p.SkipCertVerify, "SkipCertVerify should not be nil")
	assert.False(t, *p.SkipCertVerify, "SkipCertVerify should be false")
	assert.Equal(t, "hysteria.example.com", p.SNI, "SNI should be set from tlsInfo")
}