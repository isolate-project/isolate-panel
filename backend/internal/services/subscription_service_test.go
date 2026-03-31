package services

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func TestGenerateV2Ray(t *testing.T) {
	db := setupTestDB(t)
	// We intentionally do not pass cacheManager to keep cache nil for simplicity
	svc := NewSubscriptionService(db, "http://test-panel")

	user := models.User{
		ID:       1,
		Username: "testuser",
		UUID:     "12345678-1234-1234-1234-123456789012",
	}

	inbounds := []models.Inbound{
		{
			ID:             1,
			Name:           "Test VLESS",
			Protocol:       "vless",
			Port:           443,
			ListenAddress:  "example.com",
			TLSEnabled:     true,
			RealityEnabled: true,
			ConfigJSON:     `{"transport":"tcp"}`,
		},
	}

	data := &UserSubscriptionData{
		User:     user,
		Inbounds: inbounds,
	}

	result, err := svc.GenerateV2Ray(data)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// Result is base64 encoded
	decodedData, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	decoded := string(decodedData)
	if !strings.Contains(decoded, "vless://12345678-1234-1234-1234-123456789012@example.com:443") {
		t.Errorf("Decoded content does not contain expected vless link: %s", decoded)
	}
	if !strings.Contains(decoded, "security=reality") || !strings.Contains(decoded, "type=tcp") {
		t.Errorf("Decoded content missing parameters: %s", decoded)
	}
}

func TestGenerateClash(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "")

	user := models.User{
		ID:       1,
		Username: "testuser",
		UUID:     "uuid-test",
	}

	inbounds := []models.Inbound{
		{
			ID:             1,
			Name:           "Test-Clash-VLESS",
			Protocol:       "vless",
			Port:           8443,
			ListenAddress:  "example.com",
			TLSEnabled:     true,
			ConfigJSON:     `{"transport":"ws"}`,
		},
	}

	data := &UserSubscriptionData{
		User:     user,
		Inbounds: inbounds,
	}

	result, err := svc.GenerateClash(data)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	if !strings.Contains(result, "port: 7890") {
		t.Errorf("Missing base config: %s", result)
	}
	if !strings.Contains(result, "name: Test-Clash-VLESS") {
		t.Errorf("Missing proxy name: %s", result)
	}
	if !strings.Contains(result, "server: example.com") || !strings.Contains(result, "port: 8443") {
		t.Errorf("Missing server/port config: %s", result)
	}
	if !strings.Contains(result, "network: ws") {
		t.Errorf("Missing transport config: %s", result)
	}
}

func TestGenerateSingbox(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSubscriptionService(db, "")

	user := models.User{
		ID:       1,
		UUID:     "uuid-test",
	}
	inbounds := []models.Inbound{
		{
			ID:             1,
			Name:           "Test-Singbox",
			Protocol:       "vless",
			Port:           8443,
			ListenAddress:  "example.com",
			TLSEnabled:     true,
		},
	}
	data := &UserSubscriptionData{
		User:     user,
		Inbounds: inbounds,
	}

	result, err := svc.GenerateSingbox(data)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	outbounds, ok := parsed["outbounds"].([]interface{})
	if !ok || len(outbounds) == 0 {
		t.Fatalf("Missing outbounds array")
	}

	foundProxy := false
	for _, rawOb := range outbounds {
		ob := rawOb.(map[string]interface{})
		t_type, _ := ob["type"].(string)
		tag, _ := ob["tag"].(string)

		if t_type == "vless" && tag == "Test-Singbox" {
			foundProxy = true
			if ob["server"] != "example.com" || ob["server_port"].(float64) != float64(8443) {
				t.Errorf("Incorrect server or port: %v", ob)
			}
		}
	}

	if !foundProxy {
		t.Errorf("VLESS proxy not found in Sing-box config")
	}
}
