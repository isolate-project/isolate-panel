package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidate(t *testing.T) {
	// Create a temporary directory and config file for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	yamlContent := `
app:
  name: "Test Panel"
  env: "development"
  port: 8080
  host: "127.0.0.1"

database:
  path: "/tmp/test.db"

jwt:
  secret: "test-secret"
  access_token_ttl: 900
  refresh_token_ttl: 604800

logging:
  level: "info"
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// Test 1: Load config from file
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Name != "Test Panel" {
		t.Errorf("expected app name Test Panel, got %s", cfg.App.Name)
	}
	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("expected db path /tmp/test.db, got %s", cfg.Database.Path)
	}
	
	// Test env override
	os.Setenv("ISOLATE_PORT", "9090")
	os.Setenv("ISOLATE_APP_ENV", "production")
	defer os.Unsetenv("ISOLATE_PORT")
	defer os.Unsetenv("ISOLATE_APP_ENV")
	
	cfgEnv, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfgEnv.App.Port != 9090 {
		t.Errorf("expected port 9090 from env, got %d", cfgEnv.App.Port)
	}
	if cfgEnv.App.Env != "production" || cfgEnv.IsProduction() != true {
		t.Errorf("expected production env, got %s", cfgEnv.App.Env)
	}

	// Test validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() expected no err, got %v", err)
	}

	// Test validation failure
	cfg.App.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("Validate() expected error for invalid port")
	}
	
	cfg.App.Port = 8080
	cfg.JWT.Secret = ""
	_ = cfg.Validate() // JWT secret validation now warns instead of erroring (auto-generation handles it)
}
