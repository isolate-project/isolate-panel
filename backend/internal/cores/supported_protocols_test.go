package cores_test

import (
	"testing"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	"github.com/isolate-project/isolate-panel/internal/cores/singbox"
	"github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	mihomo.Register()
	singbox.Register()
	xray.Register()
}

// TestMihomo_SupportedProtocols_Naming tests Mihomo protocol naming conventions
func TestMihomo_SupportedProtocols_Naming(t *testing.T) {
	adapter, err := cores.GetCoreAdapter("mihomo")
	require.NoError(t, err, "Should get mihomo adapter")
	require.NotNil(t, adapter, "Adapter should not be nil")

	protocols := adapter.SupportedProtocols()

	assert.Contains(t, protocols, "shadowsocks", "Should contain shadowsocks (not ss)")
	assert.NotContains(t, protocols, "ss", "Should not contain ss (use shadowsocks)")

	assert.Contains(t, protocols, "shadowsocksr", "Should contain shadowsocksr (not ssr)")
	assert.NotContains(t, protocols, "ssr", "Should not contain ssr (use shadowsocksr)")

	assert.Contains(t, protocols, "tuic_v4", "Should contain tuic_v4")
	assert.Contains(t, protocols, "tuic_v5", "Should contain tuic_v5")
	assert.NotContains(t, protocols, "tuic", "Should not contain tuic (use tuic_v4 or tuic_v5)")

	assert.Contains(t, protocols, "mieru", "Should contain mieru")
	assert.Contains(t, protocols, "sudoku", "Should contain sudoku")
	assert.Contains(t, protocols, "trusttunnel", "Should contain trusttunnel")
	assert.Contains(t, protocols, "snell", "Should contain snell")
	assert.Contains(t, protocols, "mixed", "Should contain mixed")
	assert.Contains(t, protocols, "redirect", "Should contain redirect")
	assert.Contains(t, protocols, "hysteria", "Should contain hysteria")

	assert.NotContains(t, protocols, "vless-reality", "Should not contain vless-reality")
	assert.NotContains(t, protocols, "trojan-reality", "Should not contain trojan-reality")
}

// TestSingbox_SupportedProtocols_Naming tests Singbox protocol naming conventions
func TestSingbox_SupportedProtocols_Naming(t *testing.T) {
	adapter, err := cores.GetCoreAdapter("singbox")
	require.NoError(t, err, "Should get singbox adapter")
	require.NotNil(t, adapter, "Adapter should not be nil")

	protocols := adapter.SupportedProtocols()

	assert.Contains(t, protocols, "tuic_v4", "Should contain tuic_v4")
	assert.Contains(t, protocols, "tuic_v5", "Should contain tuic_v5")

	assert.Contains(t, protocols, "anytls", "Should contain anytls")
	assert.Contains(t, protocols, "naive", "Should contain naive")
	assert.Contains(t, protocols, "mixed", "Should contain mixed")
	assert.Contains(t, protocols, "redirect", "Should contain redirect")
	assert.Contains(t, protocols, "hysteria", "Should contain hysteria")

	assert.NotContains(t, protocols, "vless-reality", "Should not contain vless-reality")
	assert.NotContains(t, protocols, "trojan-reality", "Should not contain trojan-reality")
}

// TestXray_SupportedProtocols_NoPhantoms tests Xray protocol naming (no phantom protocols)
func TestXray_SupportedProtocols_NoPhantoms(t *testing.T) {
	adapter, err := cores.GetCoreAdapter("xray")
	require.NoError(t, err, "Should get xray adapter")
	require.NotNil(t, adapter, "Adapter should not be nil")

	protocols := adapter.SupportedProtocols()

	assert.Contains(t, protocols, "xhttp", "Should contain xhttp")
	assert.Contains(t, protocols, "hysteria2", "Should contain hysteria2")
	assert.Contains(t, protocols, "http", "Should contain http")
	assert.Contains(t, protocols, "socks5", "Should contain socks5")

	assert.NotContains(t, protocols, "vless-reality", "Should not contain vless-reality")
	assert.NotContains(t, protocols, "trojan-reality", "Should not contain trojan-reality")
}