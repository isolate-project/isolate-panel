package services

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeCore(name string) models.Core {
	return models.Core{ID: 1, Name: name, Version: "1.0"}
}

func TestGetInboundRealityInfo_Disabled(t *testing.T) {
	ib := models.Inbound{RealityEnabled: false}
	assert.Nil(t, getInboundRealityInfo(ib))
}

func TestGetInboundRealityInfo_EnabledWithConfig(t *testing.T) {
	ib := models.Inbound{
		RealityEnabled:    true,
		RealityConfigJSON: `{"public_key":"test-pbk","shortIds":["sid1","sid2"],"serverNames":["sn1.example.com"],"fingerprint":"safari"}`,
	}
	info := getInboundRealityInfo(ib)
	require.NotNil(t, info)
	assert.Equal(t, "test-pbk", info.PublicKey)
	assert.Equal(t, "sid1", info.ShortID)
	assert.Equal(t, "safari", info.Fingerprint)
	assert.Equal(t, "sn1.example.com", info.SNI)
}

func TestGetInboundRealityInfo_DefaultFingerprint(t *testing.T) {
	ib := models.Inbound{
		RealityEnabled:    true,
		RealityConfigJSON: `{"public_key":"pk","shortIds":["sid"]}`,
	}
	info := getInboundRealityInfo(ib)
	require.NotNil(t, info)
	assert.Equal(t, "chrome", info.Fingerprint)
}

func TestGetInboundRealityInfo_CamelCasePublicKey(t *testing.T) {
	ib := models.Inbound{
		RealityEnabled:    true,
		RealityConfigJSON: `{"publicKey":"camel-pk","shortIds":["sid"]}`,
	}
	info := getInboundRealityInfo(ib)
	require.NotNil(t, info)
	assert.Equal(t, "camel-pk", info.PublicKey)
}

func TestCoreNameForInbound(t *testing.T) {
	tests := []struct {
		coreName     string
		expectedName string
	}{
		{"xray", "Xray"},
		{"singbox", "Sing-box"},
		{"mihomo", "Mihomo"},
		{"custom", "Custom"},
	}

	for _, tc := range tests {
		t.Run(tc.coreName, func(t *testing.T) {
			ib := models.Inbound{Core: &models.Core{Name: tc.coreName}}
			assert.Equal(t, tc.expectedName, coreNameForInbound(ib))
		})
	}
}

func TestCoreNameForInbound_NoCore(t *testing.T) {
	ib := models.Inbound{}
	assert.Equal(t, "", coreNameForInbound(ib))
}

func TestFormatCorePrefix(t *testing.T) {
	ib := models.Inbound{Core: &models.Core{Name: "xray"}}
	assert.Equal(t, "[Xray]", formatCorePrefix(ib))
}

func TestFormatCorePrefix_NoCore(t *testing.T) {
	ib := models.Inbound{}
	assert.Equal(t, "", formatCorePrefix(ib))
}

func TestGenerateV2Ray_VLESS_RealityWithTransport(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("xray")
	user := models.User{ID: 1, UUID: "test-uuid-vless"}
	inbound := models.Inbound{
		ID:                 1,
		Name:               "VLESS-Reality-WS",
		Protocol:           "vless",
		Port:               443,
		ListenAddress:      "0.0.0.0",
		TLSEnabled:         false,
		RealityEnabled:     true,
		RealityConfigJSON:  `{"public_key":"test-pbk","shortIds":["sid1"],"serverNames":["sni.example.com"],"fingerprint":"chrome","privateKey":"server-priv"}`,
		ConfigJSON:         `{"transport":"websocket","ws_path":"/ws","ws_host":"ws.example.com","flow":"xtls-rprx-vision"}`,
		Core:               &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateV2Ray(data)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)
	decodedStr := string(decoded)

	assert.Contains(t, decodedStr, "vless://test-uuid-vless@")
	assert.Contains(t, decodedStr, "security=reality")
	assert.Contains(t, decodedStr, "pbk=test-pbk")
	assert.Contains(t, decodedStr, "sid=sid1")
	assert.Contains(t, decodedStr, "fp=chrome")
	assert.Contains(t, decodedStr, "sni=sni.example.com")
	assert.Contains(t, decodedStr, "type=websocket")
	assert.Contains(t, decodedStr, "path=%2Fws")
	assert.Contains(t, decodedStr, "host=ws.example.com")
	assert.Contains(t, decodedStr, "flow=xtls-rprx-vision")

	parsed, err := url.Parse(strings.SplitN(decodedStr, "#", 1)[0])
	require.NoError(t, err)
	fragment, err := url.PathUnescape(strings.SplitN(decodedStr, "#", 2)[1])
	require.NoError(t, err)
	assert.Equal(t, "[Xray]VLESS-Reality-WS", fragment)
	assert.Equal(t, "/ws", parsed.Query().Get("path"))
}

func TestGenerateV2Ray_VMess_Transport(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("xray")
	user := models.User{ID: 1, UUID: "test-uuid-vmess"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "VMess-GRPC",
		Protocol:      "vmess",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"transport":"grpc","grpc_service_name":"grpc-svc"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateV2Ray(data)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)
	decodedStr := string(decoded)

	assert.Contains(t, decodedStr, "vmess://")

	var vmessObj map[string]interface{}
	vmessB64 := strings.TrimPrefix(decodedStr, "vmess://")
	vmessBytes, err := base64.StdEncoding.DecodeString(vmessB64)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(vmessBytes, &vmessObj))

	assert.Equal(t, "grpc", vmessObj["net"])
	assert.Equal(t, "grpc-svc", vmessObj["path"])
	assert.Equal(t, "[Xray]VMess-GRPC", vmessObj["ps"])
}

func TestGenerateV2Ray_Shadowsocks_UsesConfigPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	user := models.User{ID: 1, UUID: "user-uuid-ss"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "SS-Test",
		Protocol:      "shadowsocks",
		Port:          8388,
		ListenAddress: "1.2.3.4",
		ConfigJSON:    `{"method":"chacha20-ietf-poly1305","password":"auto-generated-pass"}`,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateV2Ray(data)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)
	decodedStr := string(decoded)

	assert.Contains(t, decodedStr, "ss://")
	userInfoB64 := strings.Split(strings.TrimPrefix(decodedStr, "ss://"), "@")[0]
	userInfoBytes, err := base64.StdEncoding.DecodeString(userInfoB64)
	require.NoError(t, err)
	assert.Contains(t, string(userInfoBytes), "auto-generated-pass")
	assert.NotContains(t, string(userInfoBytes), "user-uuid-ss")
}

func TestGenerateV2Ray_Hysteria2_Obfs(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("singbox")
	user := models.User{ID: 1, UUID: "hy2-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "HY2-Obfs",
		Protocol:      "hysteria2",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"obfs_type":"salamander","obfs_password":"obfs-pass-123"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateV2Ray(data)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	assert.Contains(t, string(decoded), "obfs=salamander")
	assert.Contains(t, string(decoded), "obfs-password=obfs-pass-123")
}

func TestGenerateClash_UsesBuildClashProxy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("xray")
	user := models.User{ID: 1, UUID: "clash-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "VLESS-WS",
		Protocol:      "vless",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"transport":"ws","ws_path":"/ws","ws_host":"ws.example.com","flow":"xtls-rprx-vision"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateClash(data)
	require.NoError(t, err)

	assert.Contains(t, result, "[Xray]VLESS-WS")
	assert.Contains(t, result, "network: ws")
	assert.Contains(t, result, "flow: xtls-rprx-vision")
	assert.Contains(t, result, "ws-opts:")
	assert.Contains(t, result, "path: /ws")
}

func TestGenerateClash_RealitySupport(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("mihomo")
	user := models.User{ID: 1, UUID: "reality-uuid"}
	inbound := models.Inbound{
		ID:                 1,
		Name:               "VLESS-Reality",
		Protocol:           "vless",
		Port:               443,
		ListenAddress:      "example.com",
		RealityEnabled:     true,
		RealityConfigJSON:  `{"public_key":"reality-pbk","shortIds":["sid1"],"serverNames":["sni.example.com"]}`,
		ConfigJSON:         `{"transport":"tcp","flow":"xtls-rprx-vision"}`,
		Core:               &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateClash(data)
	require.NoError(t, err)

	assert.Contains(t, result, "reality-opts:")
	assert.Contains(t, result, "public-key: reality-pbk")
	assert.Contains(t, result, "short-id: sid1")
	assert.Contains(t, result, "client-fingerprint: chrome")
	assert.Contains(t, result, "flow: xtls-rprx-vision")
}

func TestGenerateClash_ShadowsocksPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	user := models.User{ID: 1, UUID: "ss-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "SS-Custom",
		Protocol:      "shadowsocks",
		Port:          8388,
		ListenAddress: "1.2.3.4",
		ConfigJSON:    `{"method":"chacha20-ietf-poly1305","password":"custom-ss-pass"}`,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateClash(data)
	require.NoError(t, err)

	assert.Contains(t, result, "cipher: chacha20-ietf-poly1305")
	assert.Contains(t, result, "password: custom-ss-pass")
}

func TestGenerateSingbox_TransportObject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("singbox")
	user := models.User{ID: 1, UUID: "singbox-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "VLESS-WS",
		Protocol:      "vless",
		Port:          443,
		ListenAddress: "example.com",
		TLSEnabled:    true,
		ConfigJSON:    `{"transport":"ws","ws_path":"/ws","ws_host":"ws.example.com","flow":"xtls-rprx-vision"}`,
		Core:          &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateSingbox(data)
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &config))

	outbounds, ok := config["outbounds"].([]interface{})
	require.True(t, ok)

	var vlessOb map[string]interface{}
	for _, raw := range outbounds {
		ob := raw.(map[string]interface{})
		if ob["type"] == "vless" {
			vlessOb = ob
			break
		}
	}
	require.NotNil(t, vlessOb)

	assert.Equal(t, "[Sing-box]VLESS-WS", vlessOb["tag"])
	assert.Equal(t, "xtls-rprx-vision", vlessOb["flow"])

	transport, ok := vlessOb["transport"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ws", transport["type"])
	assert.Equal(t, "/ws", transport["path"])
}

func TestGenerateSingbox_RealityTLS(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := makeCore("singbox")
	user := models.User{ID: 1, UUID: "reality-uuid"}
	inbound := models.Inbound{
		ID:                 1,
		Name:               "VLESS-Reality",
		Protocol:           "vless",
		Port:               443,
		ListenAddress:      "example.com",
		RealityEnabled:     true,
		RealityConfigJSON:  `{"public_key":"reality-pbk","shortIds":["sid1"],"serverNames":["sni.example.com"]}`,
		ConfigJSON:         `{"transport":"tcp","flow":"xtls-rprx-vision"}`,
		Core:               &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateSingbox(data)
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &config))

	outbounds := config["outbounds"].([]interface{})
	var vlessOb map[string]interface{}
	for _, raw := range outbounds {
		ob := raw.(map[string]interface{})
		if ob["type"] == "vless" {
			vlessOb = ob
			break
		}
	}
	require.NotNil(t, vlessOb)

	tls, ok := vlessOb["tls"].(map[string]interface{})
	require.True(t, ok)
	reality, ok := tls["reality"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, reality["enabled"])
	assert.Equal(t, "reality-pbk", reality["public_key"])
	assert.Equal(t, "sid1", reality["short_id"])
}

func TestGenerateSingbox_ShadowsocksPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	user := models.User{ID: 1, UUID: "ss-uuid"}
	inbound := models.Inbound{
		ID:            1,
		Name:          "SS-Test",
		Protocol:      "shadowsocks",
		Port:          8388,
		ListenAddress: "1.2.3.4",
		ConfigJSON:    `{"method":"chacha20-ietf-poly1305","password":"custom-ss-pass"}`,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateSingbox(data)
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &config))

	outbounds := config["outbounds"].([]interface{})
	var ssOb map[string]interface{}
	for _, raw := range outbounds {
		ob := raw.(map[string]interface{})
		if ob["type"] == "shadowsocks" {
			ssOb = ob
			break
		}
	}
	require.NotNil(t, ssOb)
	assert.Equal(t, "custom-ss-pass", ssOb["password"])
}

func TestGenerateIsolate_BasicStructure(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	core := models.Core{ID: 1, Name: "xray", Version: "1.0"}
	user := models.User{
		ID:                1,
		Username:          "isolate-user",
		UUID:              "isolate-uuid",
		SubscriptionToken: "sub-token-123",
		TrafficUsedBytes:  12345,
	}
	inbound := models.Inbound{
		ID:                1,
		Name:              "VLESS-Reality",
		Protocol:          "vless",
		Port:              443,
		ListenAddress:     "0.0.0.0",
		RealityEnabled:    true,
		RealityConfigJSON: `{"public_key":"isolate-pbk","shortIds":["sid1"],"serverNames":["sni.example.com"]}`,
		ConfigJSON:        `{"transport":"tcp","flow":"xtls-rprx-vision"}`,
		CoreID:            1,
		Core:              &core,
	}

	data := &UserSubscriptionData{User: user, Inbounds: []models.Inbound{inbound}}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	assert.Equal(t, 1, sub.Version)
	assert.Equal(t, "isolate-user", sub.Profile.Username)
	assert.Equal(t, "isolate-uuid", sub.Profile.UUID)
	assert.Equal(t, int64(12345), sub.Profile.TrafficUsed)
	assert.Equal(t, 24, sub.Profile.UpdateIntervalHours)
	assert.Contains(t, sub.Profile.SubscriptionURL, "/sub/sub-token-123/isolate")

	xrayCore, ok := sub.Cores["Xray"]
	require.True(t, ok)
	require.Len(t, xrayCore.Inbounds, 1)

	ib := xrayCore.Inbounds[0]
	assert.Equal(t, uint(1), ib.ID)
	assert.Equal(t, "VLESS-Reality", ib.Name)
	assert.Equal(t, "vless", ib.Protocol)
	assert.NotEmpty(t, ib.RawLink)

	require.NotNil(t, ib.TLS)
	assert.Equal(t, true, ib.TLS["reality"])
	assert.Equal(t, "isolate-pbk", ib.TLS["public_key"])
	assert.Equal(t, "sid1", ib.TLS["short_id"])
	assert.Equal(t, "chrome", ib.TLS["fingerprint"])

	require.NotNil(t, ib.Transport)
	assert.Equal(t, "tcp", ib.Transport["type"])
	assert.Equal(t, "xtls-rprx-vision", ib.Transport["flow"])
}

func TestGenerateIsolate_MultiCore(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "http://test-panel")

	xrayCore := models.Core{ID: 1, Name: "xray", Version: "1.0"}
	singboxCore := models.Core{ID: 2, Name: "singbox", Version: "1.0"}

	user := models.User{ID: 1, UUID: "multi-uuid", SubscriptionToken: "multi-token"}
	inbounds := []models.Inbound{
		{
			ID:            1,
			Name:          "Xray-VLESS",
			Protocol:      "vless",
			Port:          443,
			ListenAddress: "example.com",
			TLSEnabled:    true,
			ConfigJSON:    `{"transport":"ws","ws_path":"/ws"}`,
			CoreID:        1,
			Core:          &xrayCore,
		},
		{
			ID:            2,
			Name:          "Singbox-HY2",
			Protocol:      "hysteria2",
			Port:          8443,
			ListenAddress: "example.com",
			TLSEnabled:    true,
			ConfigJSON:    `{"password":"hy2-pass"}`,
			CoreID:        2,
			Core:          &singboxCore,
		},
	}

	data := &UserSubscriptionData{User: user, Inbounds: inbounds}
	result, err := svc.GenerateIsolate(data)
	require.NoError(t, err)

	var sub IsolateSubscription
	require.NoError(t, json.Unmarshal([]byte(result), &sub))

	assert.Len(t, sub.Cores, 2)
	_, hasXray := sub.Cores["Xray"]
	_, hasSingbox := sub.Cores["Sing-box"]
	assert.True(t, hasXray)
	assert.True(t, hasSingbox)
}

func TestAddTransportParams_Websocket(t *testing.T) {
	params := url.Values{}
	config := map[string]interface{}{
		"transport": "websocket",
		"ws_path":   "/my-path",
		"ws_host":   "ws.example.com",
	}
	addTransportParams(&params, config)

	assert.Equal(t, "websocket", params.Get("type"))
	assert.Equal(t, "/my-path", params.Get("path"))
	assert.Equal(t, "ws.example.com", params.Get("host"))
}

func TestAddTransportParams_GRPC(t *testing.T) {
	params := url.Values{}
	config := map[string]interface{}{
		"transport":         "grpc",
		"grpc_service_name": "grpc-svc",
	}
	addTransportParams(&params, config)

	assert.Equal(t, "grpc", params.Get("type"))
	assert.Equal(t, "grpc-svc", params.Get("serviceName"))
}

func TestAddTransportParams_HTTPUpgrade(t *testing.T) {
	params := url.Values{}
	config := map[string]interface{}{
		"transport": "httpupgrade",
		"ws_path":   "/upgrade",
		"ws_host":   "upgrade.example.com",
	}
	addTransportParams(&params, config)

	assert.Equal(t, "httpupgrade", params.Get("type"))
	assert.Equal(t, "/upgrade", params.Get("path"))
	assert.Equal(t, "upgrade.example.com", params.Get("host"))
}

func TestBuildClashProxy_VLESS_Reality(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{
		"transport": "tcp",
		"flow":      "xtls-rprx-vision",
	}
	tlsInfo := makeTLSInfo("sni.example.com")
	realityInfo := &inboundRealityInfo{
		PublicKey:   "reality-pbk",
		ShortID:     "sid1",
		Fingerprint: "chrome",
	}

	p := buildClashProxy("vless", "[Xray]VLESS-R", "example.com", 443, user, config, tlsInfo, nil, realityInfo)
	require.NotNil(t, p)
	assert.Equal(t, "[Xray]VLESS-R", p.Name)
	assert.Equal(t, "xtls-rprx-vision", p.Extra["flow"])

	realityOpts, ok := p.Extra["reality-opts"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "reality-pbk", realityOpts["public-key"])
	assert.Equal(t, "sid1", realityOpts["short-id"])
	assert.Equal(t, "chrome", p.Extra["client-fingerprint"])
}

func TestBuildClashProxy_Hysteria2_Obfs(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{
		"obfs_type":     "salamander",
		"obfs_password": "obfs-pass",
	}
	tlsInfo := makeTLSInfo("hy2.example.com")

	p := buildClashProxy("hysteria2", "HY2-Obfs", "example.com", 443, user, config, tlsInfo, nil, nil)
	require.NotNil(t, p)

	obfs, ok := p.Extra["obfs"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "salamander", obfs["type"])
	assert.Equal(t, "obfs-pass", obfs["password"])
}
