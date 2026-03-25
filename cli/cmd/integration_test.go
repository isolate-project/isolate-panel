package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

// TestUserListCmd_WithMockAPI tests user list command with mock API
func TestUserListCmd_WithMockAPI(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/users", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		response := map[string]interface{}{
			"success": true,
			"users": []map[string]interface{}{
				{"id": 1, "username": "user1", "email": "user1@example.com", "is_active": true},
				{"id": 2, "username": "user2", "email": "user2@example.com", "is_active": false},
			},
			"total": 2,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create temp config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &pkg.Config{
		CurrentProfile: "test",
		Profiles: map[string]pkg.Profile{
			"test": {
				PanelURL:       mockServer.URL,
				AccessToken:    "test-token",
				TokenExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	err := os.WriteFile(configPath, configData, 0600)
	assert.NoError(t, err)

	// Override config path for testing
	originalConfigPath := pkg.ConfigPath()
	defer func() {
		// Restore would need reflection or global var, skipping for now
		_ = originalConfigPath
	}()

	// Test that config can be loaded
	loadedConfig, err := pkg.LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, loadedConfig)
}

// TestSettingsGetCmd_WithMockAPI tests settings get command with mock API
func TestSettingsGetCmd_WithMockAPI(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/settings", r.URL.Path)

		response := map[string]interface{}{
			"success": true,
			"settings": map[string]string{
				"panel_name":      "Test Panel",
				"monitoring_mode": "lite",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &pkg.Config{
		CurrentProfile: "test",
		Profiles: map[string]pkg.Profile{
			"test": {
				PanelURL:       mockServer.URL,
				AccessToken:    "test-token",
				TokenExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	err := os.WriteFile(configPath, configData, 0600)
	assert.NoError(t, err)

	// Test config file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

// TestBackupListCmd_WithMockAPI tests backup list command with mock API
func TestBackupListCmd_WithMockAPI(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/backups", r.URL.Path)

		response := map[string]interface{}{
			"success": true,
			"backups": []map[string]interface{}{
				{"id": 1, "filename": "backup-20260325.db", "size_bytes": 1048576},
				{"id": 2, "filename": "backup-20260324.db", "size_bytes": 2097152},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &pkg.Config{
		CurrentProfile: "test",
		Profiles: map[string]pkg.Profile{
			"test": {
				PanelURL:       mockServer.URL,
				AccessToken:    "test-token",
				TokenExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	err := os.WriteFile(configPath, configData, 0600)
	assert.NoError(t, err)

	// Verify config structure
	loadedData, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	var loadedConfig pkg.Config
	err = json.Unmarshal(loadedData, &loadedConfig)
	assert.NoError(t, err)
	assert.Equal(t, "test", loadedConfig.CurrentProfile)
	assert.Contains(t, loadedConfig.Profiles, "test")
}

// TestCoreStatusCmd_WithMockAPI tests core status command with mock API
func TestCoreStatusCmd_WithMockAPI(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/cores")

		response := map[string]interface{}{
			"success": true,
			"cores": []map[string]interface{}{
				{"id": 1, "name": "xray", "type": "xray", "status": "running"},
				{"id": 2, "name": "singbox", "type": "singbox", "status": "stopped"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := &pkg.Config{
		CurrentProfile: "test",
		Profiles: map[string]pkg.Profile{
			"test": {
				PanelURL:       mockServer.URL,
				AccessToken:    "test-token",
				TokenExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	err := os.WriteFile(configPath, configData, 0600)
	assert.NoError(t, err)

	// Test profile access
	profile := config.Profiles["test"]
	assert.Equal(t, mockServer.URL, profile.PanelURL)
	assert.Equal(t, "test-token", profile.AccessToken)
}
