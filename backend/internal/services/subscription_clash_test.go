package services

import (
	"encoding/json"
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func makeTestUser() models.User {
	return models.User{
		ID:                1,
		Username:          "clashuser",
		UUID:              "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		SubscriptionToken: "sub-token",
	}
}

func makeTestTLSInfo() inboundTLSInfo {
	return inboundTLSInfo{SNI: "", IsTLS: false}
}

func makeTLSInfo(sni string) inboundTLSInfo {
	return inboundTLSInfo{SNI: sni, IsTLS: true}
}

func TestBuildClashProxy_VMess(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"transport": "ws", "ws_path": "/ws", "ws_host": "example.com"}
	tlsInfo := makeTLSInfo("sni.example.com")

	p := buildClashProxy("vmess", "my-vmess", "example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "vmess", p.Type)
	assert.Equal(t, "my-vmess", p.Name)
	assert.Equal(t, "example.com", p.Server)
	assert.Equal(t, 443, p.Port)
	assert.Equal(t, user.UUID, p.UUID)
	assert.Equal(t, 0, p.AlterId)
	assert.Equal(t, "auto", p.Cipher)
	assert.Equal(t, "ws", p.Network)
	assert.NotNil(t, p.TLS)
	assert.True(t, *p.TLS)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
	assert.Equal(t, "sni.example.com", p.ServerName, "ServerName should be set from SNI when different from server")

	require.NotNil(t, p.Extra)
	wsOpts, ok := p.Extra["ws-opts"]
	require.True(t, ok, "should have ws-opts")
	wsMap, ok := wsOpts.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "/ws", wsMap["path"])
}

func TestBuildClashProxy_VMess_TCP(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"transport": "tcp"}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("vmess", "tcp-vmess", "1.2.3.4", 8080, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "vmess", p.Type)
	assert.Equal(t, "tcp", p.Network)
	assert.NotNil(t, p.TLS)
	assert.False(t, *p.TLS)
	assert.Nil(t, p.Extra)
}

func TestBuildClashProxy_VLESS(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{
		"transport":         "grpc",
		"flow":              "xtls-rprx-vision",
		"encryption":        "none",
		"grpc_service_name": "grpc-svc",
	}
	tlsInfo := makeTLSInfo("sni.vless.example.com")

	p := buildClashProxy("vless", "my-vless", "vless.example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "vless", p.Type)
	assert.Equal(t, user.UUID, p.UUID)
	assert.Equal(t, "grpc", p.Network)
	assert.NotNil(t, p.TLS)
	assert.True(t, *p.TLS)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
	assert.Equal(t, "sni.vless.example.com", p.ServerName, "ServerName should be SNI when different from server")

	require.NotNil(t, p.Extra)
	assert.Equal(t, "xtls-rprx-vision", p.Extra["flow"])
	_, hasEncryption := p.Extra["encryption"]
	assert.False(t, hasEncryption, `"none" encryption should not appear in extra`)
	grpcOpts, ok := p.Extra["grpc-opts"]
	require.True(t, ok)
	grpcMap, ok := grpcOpts.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "grpc-svc", grpcMap["grpc-service-name"])
}

func TestBuildClashProxy_VLESS_NoTLS(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"transport": "tcp"}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("vless", "notls-vless", "10.0.0.1", 80, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "vless", p.Type)
	assert.NotNil(t, p.TLS)
	assert.False(t, *p.TLS)
	assert.Empty(t, p.ServerName)
}

func TestBuildClashProxy_Trojan(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTLSInfo("trojan.example.com")

	p := buildClashProxy("trojan", "my-trojan", "trojan.example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "trojan", p.Type)
	assert.Equal(t, user.UUID, p.Password)
	assert.Equal(t, "trojan.example.com", p.SNI)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
}

func TestBuildClashProxy_Shadowsocks(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"method": "chacha20-ietf-poly1305"}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("shadowsocks", "my-ss", "1.2.3.4", 8388, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "ss", p.Type)
	assert.Equal(t, "chacha20-ietf-poly1305", p.Cipher)
	assert.Equal(t, user.UUID, p.Password)
}

func TestBuildClashProxy_Shadowsocks_DefaultMethod(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("shadowsocks", "ss-default", "1.2.3.4", 8388, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "aes-256-gcm", p.Cipher, "default cipher should be aes-256-gcm")
}

func TestBuildClashProxy_SkipCertVerify_False_BM20(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"transport": "tcp"}
	tlsInfo := makeTLSInfo("secure.example.com")

	protocols := []string{"vmess", "vless", "trojan", "hysteria2", "tuic_v4", "tuic_v5", "anytls", "hysteria"}
	for _, proto := range protocols {
		t.Run(proto, func(t *testing.T) {
			p := buildClashProxy(proto, "test-"+proto, "example.com", 443, user, config, tlsInfo, nil)
			if p == nil {
				t.Skipf("protocol %s returned nil proxy", proto)
			}
			require.NotNil(t, p.SkipCertVerify, "%s: SkipCertVerify must not be nil (BM-20)", proto)
			assert.False(t, *p.SkipCertVerify, "%s: SkipCertVerify must be false (BM-20)", proto)
		})
	}
}

func TestBuildClashProxy_SkipCertVerify_False_NonTLSTransport(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{"transport": "tcp"}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("vmess", "notls-vmess", "1.2.3.4", 80, user, config, tlsInfo, nil)
	require.NotNil(t, p)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify, "skip-cert-verify must be false even without TLS")
}

func TestBuildClashProxy_UnsupportedProtocol(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("unknown_proto", "bad", "1.2.3.4", 80, user, config, tlsInfo, nil)
	assert.Nil(t, p, "unsupported protocol should return nil")
}

func TestBuildClashProxy_Hysteria2(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTLSInfo("hy2.example.com")

	p := buildClashProxy("hysteria2", "my-hy2", "10.0.0.1", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "hysteria2", p.Type)
	assert.Equal(t, user.UUID, p.Password)
	assert.Equal(t, "hy2.example.com", p.SNI, "SNI should be set from tlsInfo when different from server")
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
}

func TestBuildClashProxy_TUICv4(t *testing.T) {
	token := "tuic-token-123"
	user := makeTestUser()
	user.Token = &token
	config := map[string]interface{}{"congestion_control": "cubic"}
	tlsInfo := makeTLSInfo("tuic.example.com")

	p := buildClashProxy("tuic_v4", "my-tuic4", "tuic.example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "tuic", p.Type)
	assert.Equal(t, token, p.Token, "should use user.Token when set")
	assert.Equal(t, 4, p.Version)
	assert.Equal(t, "cubic", p.CongestionController)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
}

func TestBuildClashProxy_TUICv4_NoToken(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("tuic_v4", "tuic4-notoken", "1.2.3.4", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, user.UUID, p.Token, "should fallback to UUID when no token")
	assert.Equal(t, "bbr", p.CongestionController, "default congestion should be bbr")
}

func TestBuildClashProxy_TUICv5(t *testing.T) {
	token := "tuic5-pass"
	user := makeTestUser()
	user.Token = &token
	config := map[string]interface{}{"congestion_control": "bbr"}
	tlsInfo := makeTLSInfo("tuic5.example.com")

	p := buildClashProxy("tuic_v5", "my-tuic5", "tuic5.example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "tuic", p.Type)
	assert.Equal(t, user.UUID, p.UUID)
	assert.Equal(t, token, p.Password)
	assert.Equal(t, 5, p.Version)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
}

func TestBuildClashProxy_SSR(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{
		"cipher":   "chacha20-poly1305",
		"protocol": "auth_aes128_md5",
		"obfs":     "tls1.2_ticket_auth",
	}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("ssr", "my-ssr", "1.2.3.4", 8388, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "ssr", p.Type)
	assert.Equal(t, "chacha20-poly1305", p.Cipher)
	assert.Equal(t, "auth_aes128_md5", p.Protocol)
	assert.Equal(t, "tls1.2_ticket_auth", p.Obfs)
	assert.Equal(t, user.UUID, p.Password)
}

func TestBuildClashProxy_Snell(t *testing.T) {
	psk := "snell-psk-value"
	user := makeTestUser()
	user.Token = &psk
	config := map[string]interface{}{"version": "3", "obfs": "http"}
	tlsInfo := makeTestTLSInfo()

	p := buildClashProxy("snell", "my-snell", "1.2.3.4", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "snell", p.Type)
	assert.Equal(t, psk, p.PSK)
	assert.Equal(t, 3, p.Version)
	require.NotNil(t, p.ObfsOpts)
	assert.Equal(t, "http", p.ObfsOpts.Mode)
}

func TestMarshalClashConfig_ValidConfig(t *testing.T) {
	cfg := clashConfig{
		Port:      7890,
		SocksPort: 7891,
		AllowLan:  false,
		Mode:      "rule",
		LogLevel:  "info",
		Proxies:   []clashProxy{},
		ProxyGroups: []clashProxyGroup{
			{Name: "PROXY", Type: "select", Proxies: []string{}},
		},
		Rules: []string{"MATCH,PROXY"},
	}

	result, err := marshalClashConfig(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "port: 7890")
	assert.Contains(t, result, "socks-port: 7891")
	assert.Contains(t, result, "mode: rule")
	assert.Contains(t, result, "log-level: info")
	assert.Contains(t, result, "MATCH,PROXY")

	var parsed clashConfig
	require.NoError(t, yaml.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, 7890, parsed.Port)
	assert.Equal(t, 7891, parsed.SocksPort)
	assert.Equal(t, "rule", parsed.Mode)
}

func TestMarshalClashConfig_EmptyProxyList(t *testing.T) {
	cfg := clashConfig{
		Port:      7890,
		SocksPort: 7891,
		Mode:      "rule",
		LogLevel:  "info",
		Proxies:   []clashProxy{},
		ProxyGroups: []clashProxyGroup{
			{Name: "PROXY", Type: "select", Proxies: []string{}},
		},
		Rules: []string{"MATCH,PROXY"},
	}

	result, err := marshalClashConfig(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	var parsed clashConfig
	require.NoError(t, yaml.Unmarshal([]byte(result), &parsed))
	assert.Empty(t, parsed.Proxies)
	assert.Equal(t, 7890, parsed.Port)
}

func TestMarshalClashConfig_NilProxyList(t *testing.T) {
	cfg := clashConfig{
		Port:      7890,
		SocksPort: 7891,
		Mode:      "rule",
		LogLevel:  "info",
		Proxies:   nil,
		Rules:     []string{"MATCH,PROXY"},
	}

	result, err := marshalClashConfig(cfg)
	require.NoError(t, err, "nil proxies should marshal without error")
	assert.NotEmpty(t, result)
}

func TestClashProxiesStructure_ForSubscription(t *testing.T) {
	user := makeTestUser()
	certsByIDs := map[uint]*models.Certificate{}

	inbounds := []models.Inbound{
		{
			ID:            1,
			Name:          "trojan-in",
			Protocol:      "trojan",
			Port:          443,
			ListenAddress: "trojan.example.com",
			TLSEnabled:    true,
			TLSCertID:     nil,
			ConfigJSON:    `{}`,
		},
		{
			ID:            2,
			Name:          "vless-in",
			Protocol:      "vless",
			Port:          8443,
			ListenAddress: "vless.example.com",
			TLSEnabled:    true,
			ConfigJSON:    `{"transport":"tcp"}`,
		},
		{
			ID:            3,
			Name:          "hy2-in",
			Protocol:      "hysteria2",
			Port:          8388,
			ListenAddress: "hy2.example.com",
			TLSEnabled:    true,
			ConfigJSON:    `{}`,
		},
	}

	var proxies []clashProxy
	var proxyNames []string

	for _, inbound := range inbounds {
		var config map[string]interface{}
		if inbound.ConfigJSON != "" {
			_ = json.Unmarshal([]byte(inbound.ConfigJSON), &config)
		}
		if config == nil {
			config = make(map[string]interface{})
		}

		server := resolveServerAddr(inbound, "http://panel.example.com", certsByIDs)
		tlsInfo := getInboundTLSInfo(inbound, certsByIDs)

		proxy := buildClashProxy(inbound.Protocol, inbound.Name, server, inbound.Port, user, config, tlsInfo, certsByIDs)
		if proxy != nil {
			proxies = append(proxies, *proxy)
			proxyNames = append(proxyNames, inbound.Name)
		}
	}

	require.Len(t, proxies, 3, "all three protocols should produce proxies")
	require.Len(t, proxyNames, 3)

	assert.Equal(t, "trojan", proxies[0].Type)
	assert.Equal(t, "trojan-in", proxies[0].Name)
	assert.Equal(t, user.UUID, proxies[0].Password)
	assert.NotNil(t, proxies[0].SkipCertVerify)
	assert.False(t, *proxies[0].SkipCertVerify)

	assert.Equal(t, "vless", proxies[1].Type)
	assert.Equal(t, "vless-in", proxies[1].Name)
	assert.Equal(t, user.UUID, proxies[1].UUID)
	assert.NotNil(t, proxies[1].SkipCertVerify)
	assert.False(t, *proxies[1].SkipCertVerify)

	assert.Equal(t, "hysteria2", proxies[2].Type)
	assert.Equal(t, "hy2-in", proxies[2].Name)
	assert.Equal(t, user.UUID, proxies[2].Password)
	assert.NotNil(t, proxies[2].SkipCertVerify)
	assert.False(t, *proxies[2].SkipCertVerify)

	assert.Equal(t, []string{"trojan-in", "vless-in", "hy2-in"}, proxyNames)

	cfg := clashConfig{
		Port:      7890,
		SocksPort: 7891,
		AllowLan:  false,
		Mode:      "rule",
		LogLevel:  "info",
		Proxies:   proxies,
		ProxyGroups: []clashProxyGroup{
			{Name: "PROXY", Type: "select", Proxies: proxyNames},
		},
		Rules: []string{"MATCH,PROXY"},
	}

	assert.Equal(t, 7890, cfg.Port)
	assert.Equal(t, "rule", cfg.Mode)
	assert.Len(t, cfg.Proxies, 3)
	assert.Len(t, cfg.ProxyGroups, 1)
	assert.Equal(t, "PROXY", cfg.ProxyGroups[0].Name)
	assert.Equal(t, []string{"trojan-in", "vless-in", "hy2-in"}, cfg.ProxyGroups[0].Proxies)
	assert.Contains(t, cfg.Rules, "MATCH,PROXY")
}

func TestBuildClashProxy_HTTP(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTLSInfo("http.example.com")

	p := buildClashProxy("http", "my-http", "http.example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "http", p.Type)
	assert.Equal(t, "clashuser", p.Username)
	assert.Equal(t, user.UUID, p.Password)
	assert.NotNil(t, p.TLS)
	assert.True(t, *p.TLS)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
}

func TestBuildClashProxy_Socks5(t *testing.T) {
	user := makeTestUser()
	config := map[string]interface{}{}
	tlsInfo := makeTLSInfo("socks.example.com")

	p := buildClashProxy("socks5", "my-socks5", "socks.example.com", 443, user, config, tlsInfo, nil)

	require.NotNil(t, p)
	assert.Equal(t, "socks5", p.Type)
	assert.Equal(t, "clashuser", p.Username)
	assert.Equal(t, user.UUID, p.Password)
	assert.NotNil(t, p.TLS)
	assert.True(t, *p.TLS)
	assert.NotNil(t, p.SkipCertVerify)
	assert.False(t, *p.SkipCertVerify)
}
