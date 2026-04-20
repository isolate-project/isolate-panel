package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnyTLS_RegisteredForSingboxOnly tests AnyTLS protocol registration
func TestAnyTLS_RegisteredForSingboxOnly(t *testing.T) {
	schema, ok := GetProtocolSchema("anytls")
	require.True(t, ok, "anytls protocol should be registered")
	require.NotNil(t, schema, "Schema should not be nil")

	assert.Equal(t, "anytls", schema.Protocol, "Protocol name should be anytls")
	assert.Equal(t, "AnyTLS", schema.Label, "Label should be AnyTLS")
	assert.Equal(t, []string{"singbox"}, schema.Core, "Core should be singbox only")
	assert.NotContains(t, schema.Core, "xray", "Should not include xray")
	assert.NotContains(t, schema.Core, "mihomo", "Should not include mihomo")

	passwordParam, ok := schema.Parameters["password"]
	require.True(t, ok, "password parameter should exist")
	assert.True(t, passwordParam.AutoGenerate, "password should have AutoGenerate=true")
	assert.Equal(t, "generate_base64_token_32", passwordParam.AutoGenFunc, "AutoGenFunc should be generate_base64_token_32")

	paddingParam, ok := schema.Parameters["padding_scheme"]
	require.True(t, ok, "padding_scheme parameter should exist")
	assert.Equal(t, "advanced", paddingParam.Group, "padding_scheme should be in advanced group")
}

// TestHysteriaV1_RegisteredAsDeprecated tests Hysteria v1 deprecation
func TestHysteriaV1_RegisteredAsDeprecated(t *testing.T) {
	schema, ok := GetProtocolSchema("hysteria")
	require.True(t, ok, "hysteria protocol should be registered")
	require.NotNil(t, schema, "Schema should not be nil")

	assert.Equal(t, "hysteria", schema.Protocol, "Protocol name should be hysteria")
	assert.False(t, schema.Deprecated, "Deprecated should be false (outbound hysteria is not deprecated)")

	assert.Contains(t, schema.Core, "sing-box", "Core should include sing-box")
	assert.Contains(t, schema.Core, "xray", "Core should include xray")
	assert.Contains(t, schema.Core, "mihomo", "Core should include mihomo")

	authStrParam, ok := schema.Parameters["auth_str"]
	require.True(t, ok, "auth_str parameter should exist")
	assert.True(t, authStrParam.AutoGenerate, "auth_str should have AutoGenerate=true")

	upMbpsParam, ok := schema.Parameters["up_mbps"]
	require.True(t, ok, "up_mbps parameter should exist")
	assert.Equal(t, 100, upMbpsParam.Default, "up_mbps default should be 100")

	downMbpsParam, ok := schema.Parameters["down_mbps"]
	require.True(t, ok, "down_mbps parameter should exist")
	assert.Equal(t, 100, downMbpsParam.Default, "down_mbps default should be 100")
}

// TestDeprecatedFieldsInProtocolSummary tests deprecated protocols in GetAllProtocols
func TestDeprecatedFieldsInProtocolSummary(t *testing.T) {
	protocols := GetAllProtocols()

	var deprecatedProtocols []string
	for _, p := range protocols {
		if p.Deprecated {
			deprecatedProtocols = append(deprecatedProtocols, p.Protocol)
			assert.NotEmpty(t, p.DeprecationNotice, "Deprecated protocol %s should have DeprecationNotice", p.Protocol)
		}
	}
}

// TestAnyTLS_NotValidForXray tests AnyTLS core validation
func TestAnyTLS_NotValidForXray(t *testing.T) {
	assert.False(t, ValidateProtocolForCore("anytls", "xray"), "anytls should not be valid for xray")
	assert.False(t, ValidateProtocolForCore("anytls", "mihomo"), "anytls should not be valid for mihomo")
	assert.True(t, ValidateProtocolForCore("anytls", "singbox"), "anytls should be valid for singbox")
}

// TestHysteriaV1_ValidForSingboxAndMihomo tests Hysteria v1 core validation
func TestHysteriaV1_ValidForSingboxAndMihomo(t *testing.T) {
	assert.True(t, ValidateProtocolForCore("hysteria", "sing-box"), "hysteria should be valid for sing-box")
	assert.True(t, ValidateProtocolForCore("hysteria", "xray"), "hysteria should be valid for xray")
	assert.True(t, ValidateProtocolForCore("hysteria", "mihomo"), "hysteria should be valid for mihomo")
}