package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseConnection(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cfg := &Config{
		Path:         dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	dbManager, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create database manager: %v", err)
	}

	if dbManager == nil || dbManager.DB == nil {
		t.Fatal("expected non-nil database manager and DB")
	}

	// Wait for connection to establish
	sqlDB, err := dbManager.DB.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping db: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected sqlite db file to exist at %s", dbPath)
	}
}
