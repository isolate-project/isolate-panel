package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/isolate-project/isolate-panel/cli/pkg"
)

func setupTestEnvironment(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func()) {
	mockServer := httptest.NewServer(handler)

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

	os.Setenv("ISOLATE_PANEL_CONFIG", configPath)

	cleanup := func() {
		mockServer.Close()
		os.Unsetenv("ISOLATE_PANEL_CONFIG")
		pkg.DefaultClient = nil // Reset any mocked client
	}

	return mockServer, cleanup
}

func TestE2E_UserList(t *testing.T) {
	_, cleanup := setupTestEnvironment(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/users", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		response := map[string]interface{}{
			"success": true,
			"data": []map[string]interface{}{
				{"id": 1, "username": "admin", "email": "admin@local.host", "is_active": true},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer cleanup()

	cmd := UserCmd()
	cmd.SetArgs([]string{"list", "--format=json"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "admin@local.host")
	assert.Contains(t, output, "admin")
}

func TestE2E_CoreStatus(t *testing.T) {
	_, cleanup := setupTestEnvironment(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/cores/xray/status", r.URL.Path)

		response := map[string]interface{}{
			"name": "xray",
			"status": "running",
			"uptime": "10m",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer cleanup()

	cmd := CoreCmd()
	cmd.SetArgs([]string{"status", "xray", "--format=table"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "xray"), "Output should contain core name")
	assert.True(t, strings.Contains(output, "running"), "Output should contain running status")
}
