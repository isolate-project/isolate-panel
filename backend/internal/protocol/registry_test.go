package protocol

import (
	"strings"
	"testing"
)

func TestGenerateUUIDv4(t *testing.T) {
	uuid := GenerateUUIDv4()

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	if len(uuid) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(uuid))
	}

	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Errorf("Expected 5 parts, got %d", len(parts))
	}

	// Check version 4
	if !strings.HasPrefix(parts[2], "4") {
		t.Errorf("Expected version 4, got %s", parts[2])
	}
}

func TestGeneratePassword(t *testing.T) {
	password, err := GeneratePassword(32)
	if err != nil {
		t.Fatalf("GeneratePassword(32) failed: %v", err)
	}

	if len(password) != 32 {
		t.Errorf("Expected password length 32, got %d", len(password))
	}

	password2, err := GeneratePassword(32)
	if err != nil {
		t.Fatalf("GeneratePassword(32) failed: %v", err)
	}
	if password == password2 {
		t.Error("Generated passwords should be different")
	}
}

func TestGenerateBase64Token(t *testing.T) {
	token := GenerateBase64Token(16)

	// Base64 encoding of 16 bytes = 24 characters (with padding)
	if len(token) < 20 {
		t.Errorf("Expected token length >= 20, got %d", len(token))
	}

	// Should only contain base64url characters (RFC 4648)
	// base64url uses - and _ instead of + and /
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_="
	for _, c := range token {
		if !strings.ContainsRune(validChars, c) {
			t.Errorf("Invalid character in token: %c", c)
		}
	}
}

func TestGenerateRandomPath(t *testing.T) {
	path, err := GenerateRandomPath("")
	if err != nil {
		t.Fatalf("GenerateRandomPath failed: %v", err)
	}

	if len(path) == 0 {
		t.Error("Expected non-empty path")
	}

	if !strings.HasPrefix(path, "/") {
		t.Error("Expected path to start with /")
	}

	path2, err := GenerateRandomPath("")
	if err != nil {
		t.Fatalf("GenerateRandomPath failed: %v", err)
	}
	if path == path2 {
		t.Error("Generated paths should be different")
	}

	pathWithPrefix, err := GenerateRandomPath("test")
	if err != nil {
		t.Fatalf("GenerateRandomPath with prefix failed: %v", err)
	}
	if !strings.Contains(pathWithPrefix, "/test/") {
		t.Errorf("Expected path to contain /test/, got %s", pathWithPrefix)
	}
}

func TestGenerateShortID(t *testing.T) {
	id, err := GenerateShortID(8)
	if err != nil {
		t.Fatalf("GenerateShortID(8) failed: %v", err)
	}

	if len(id) != 8 {
		t.Errorf("Expected ID length 8, got %d", len(id))
	}

	// Should only contain lowercase alphanumeric
	validChars := "abcdefghijklmnopqrstuvwxyz0123456789"
	for _, c := range id {
		if !strings.ContainsRune(validChars, c) {
			t.Errorf("Invalid character in ID: %c", c)
		}
	}
}

func TestAutoGenerate(t *testing.T) {
	tests := []struct {
		funcName string
		wantErr  bool
	}{
		{"generate_uuid_v4", false},
		{"generate_password_8", false},
		{"generate_password_16", false},
		{"generate_password_32", false},
		{"generate_base64_token_32", false},
		{"generate_base64_token_44", false},
		{"generate_random_path", false},
		{"generate_short_id_8", false},
		{"unknown_func", true},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			result, err := AutoGenerate(tt.funcName)
			if (err != nil) != tt.wantErr {
				t.Errorf("AutoGenerate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestRegisterAndGetProtocolSchema(t *testing.T) {
	// Test that registry is populated (protocols.go should have registered protocols)
	schemas := GetAllProtocols()

	if len(schemas) == 0 {
		t.Error("Expected at least some protocols in registry")
	}

	// Try to get a known protocol (vless should exist)
	schema, ok := GetProtocolSchema("vless")
	if !ok {
		t.Error("Expected to find vless protocol")
	}

	if schema == nil {
		t.Fatal("GetProtocolSchema returned nil")
	}

	if schema.Protocol != "vless" {
		t.Errorf("Expected protocol name 'vless', got '%s'", schema.Protocol)
	}
}

func TestGetProtocolsByCore(t *testing.T) {
	protocols := GetProtocolsByCore("xray")

	if len(protocols) == 0 {
		t.Error("Expected at least some protocols for xray core")
	}

	// All returned protocols should have xray in their core list
	for _, p := range protocols {
		found := false
		for _, core := range p.Core {
			if core == "xray" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Protocol %s should have xray core", p.Protocol)
		}
	}
}

func TestGetProtocolsByCoreAndDirection(t *testing.T) {
	protocols := GetProtocolsByCoreAndDirection("xray", "inbound")

	for _, p := range protocols {
		if p.Direction != "inbound" && p.Direction != "both" {
			t.Errorf("Protocol %s should be inbound or both, got %s", p.Protocol, p.Direction)
		}
	}
}

func TestValidateProtocolForCore(t *testing.T) {
	// Valid combination
	if !ValidateProtocolForCore("vless", "xray") {
		t.Error("Expected vless to be valid for xray")
	}

	// Invalid combination
	if ValidateProtocolForCore("vless", "singbox") {
		t.Error("Expected vless to be invalid for singbox")
	}

	// Non-existent protocol
	if ValidateProtocolForCore("nonexistent", "xray") {
		t.Error("Expected nonexistent protocol to be invalid")
	}
}

func TestRegistryCompleteness(t *testing.T) {
	protocols := GetAllProtocols()

	// We expect 25 protocols as per the analysis
	if len(protocols) < 20 {
		t.Errorf("Expected at least 20 protocols, got %d", len(protocols))
	}
}

func TestAllProtocolsHaveRequiredFields(t *testing.T) {
	protocols := GetAllProtocols()

	for _, p := range protocols {
		if p.Label == "" {
			t.Errorf("Protocol %s has no label", p.Protocol)
		}
		if len(p.Core) == 0 {
			t.Errorf("Protocol %s has no assigned cores", p.Protocol)
		}
		if p.Direction == "" {
			t.Errorf("Protocol %s has no direction", p.Protocol)
		}
		if p.Category == "" {
			t.Errorf("Protocol %s has no category", p.Protocol)
		}
	}
}

func TestAutoGenFuncsAreValid(t *testing.T) {
	// Access the internal registry map directly since we are in the same package
	for _, p := range registry {
		for key, param := range p.Parameters {
			if param.AutoGenerate && param.AutoGenFunc != "" {
				result, err := AutoGenerate(param.AutoGenFunc)
				if err != nil {
					t.Errorf("Protocol %s, param %s: AutoGenFunc %s failed: %v", p.Protocol, key, param.AutoGenFunc, err)
				}
				if result == nil {
					t.Errorf("Protocol %s, param %s: AutoGenFunc %s returned nil", p.Protocol, key, param.AutoGenFunc)
				}
			}
		}
	}
}

func TestDependsOnReferencesExist(t *testing.T) {
	// Access the internal registry map directly since we are in the same package
	for _, p := range registry {
		for key, param := range p.Parameters {
			if param.DependsOn != nil {
				for _, dep := range param.DependsOn {
					depField := dep.Field
					if _, ok := p.Parameters[depField]; !ok {
						t.Errorf("Protocol %s, param %s: DependsOn field %s does not exist", p.Protocol, key, depField)
					}
				}
			}
		}
	}
}

func TestProtocolCategories(t *testing.T) {
	validCategories := map[string]bool{
		"proxy":   true,
		"tunnel":  true,
		"utility": true,
	}

	protocols := GetAllProtocols()
	for _, p := range protocols {
		if !validCategories[p.Category] {
			t.Errorf("Protocol %s has invalid category: %s", p.Protocol, p.Category)
		}
	}
}

func TestProtocolDirections(t *testing.T) {
	validDirections := map[string]bool{
		"inbound":  true,
		"outbound": true,
		"both":     true,
	}

	protocols := GetAllProtocols()
	for _, p := range protocols {
		if !validDirections[string(p.Direction)] {
			t.Errorf("Protocol %s has invalid direction: %s", p.Protocol, p.Direction)
		}
	}
}

func TestDefaultWidget_Mapping(t *testing.T) {
	tests := []struct {
		name     string
		paramType ParameterType
		expected string
	}{
		{"String to input", TypeString, "input"},
		{"Integer to number", TypeInteger, "number"},
		{"Boolean to checkbox", TypeBoolean, "checkbox"},
		{"Select to select", TypeSelect, "select"},
		{"UUID to input", TypeUUID, "input"},
		{"Array to tags", TypeArray, "tags"},
		{"Object to textarea", TypeObject, "textarea"},
		{"Unknown to input", ParameterType("unknown"), "input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultWidget(tt.paramType)
			if result != tt.expected {
				t.Errorf("DefaultWidget(%v) = %v, want %v", tt.paramType, result, tt.expected)
			}
		})
	}
}
