package services

import (
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Auto migrate all tables
	err = db.AutoMigrate(&models.Core{}, &models.Inbound{}, &models.Outbound{})
	if err != nil {
		t.Fatalf("Failed to migrate test DB: %v", err)
	}

	return db
}

func TestPortManager_IsPortAvailable_ValidPort(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	// Port in valid range, not reserved, not in use
	available, _, err := pm.IsPortAvailable(50000, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !available {
		t.Error("Expected port 50000 to be available")
	}
}

func TestPortManager_IsPortAvailable_BelowRange(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	available, reason, err := pm.IsPortAvailable(500, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if available {
		t.Error("Expected port 500 to be unavailable (below 1024)")
	}
	if reason == "" {
		t.Error("Expected reason for unavailability")
	}
}

func TestPortManager_IsPortAvailable_AboveRange(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	available, _, err := pm.IsPortAvailable(70000, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if available {
		t.Error("Expected port 70000 to be unavailable (above 65535)")
	}
}

func TestPortManager_IsPortAvailable_Reserved(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	reservedPorts := []int{22, 53, 80, 443, 8080, 8443, 9090, 10085, 9097}
	for _, port := range reservedPorts {
		available, reason, err := pm.IsPortAvailable(port, nil)
		if err != nil {
			t.Errorf("Unexpected error for port %d: %v", port, err)
			continue
		}
		if available {
			t.Errorf("Expected port %d to be unavailable (reserved)", port)
		}
		if reason == "" {
			t.Errorf("Expected reason for port %d unavailability", port)
		}
	}
}

func TestPortManager_IsPortAvailable_InUse(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	// Create an inbound using port 12345
	inbound := models.Inbound{
		Name:     "Test Inbound",
		Protocol: "vless",
		CoreID:   1,
		Port:     12345,
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatalf("Failed to create test inbound: %v", err)
	}

	available, reason, err := pm.IsPortAvailable(12345, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if available {
		t.Error("Expected port 12345 to be unavailable (in use)")
	}
	if reason == "" {
		t.Error("Expected reason for unavailability")
	}
}

func TestPortManager_IsPortAvailable_ExcludeInbound(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	// Create an inbound using port 12345
	inbound := models.Inbound{
		Name:     "Test Inbound",
		Protocol: "vless",
		CoreID:   1,
		Port:     12345,
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatalf("Failed to create test inbound: %v", err)
	}

	// Check with exclude - should be available
	inboundID := inbound.ID
	available, _, err := pm.IsPortAvailable(12345, &inboundID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !available {
		t.Error("Expected port 12345 to be available when excluded")
	}
}

func TestPortManager_AllocatePort(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	port, err := pm.AllocatePort()
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	if port < 10000 || port > 60000 {
		t.Errorf("Allocated port %d out of range [10000, 60000]", port)
	}

	// Verify port is actually available
	available, _, err := pm.IsPortAvailable(port, nil)
	if err != nil {
		t.Fatalf("Unexpected error checking allocated port: %v", err)
	}
	if !available {
		t.Errorf("Allocated port %d should be available", port)
	}
}

func TestPortManager_AllocatePort_AvoidsUsed(t *testing.T) {
	db := setupTestDB(t)
	pm := NewPortManager(db)

	// Create several inbounds to occupy ports
	ports := []int{20000, 20001, 20002}
	for i, port := range ports {
		inbound := models.Inbound{
			Name:     "Test Inbound",
			Protocol: "vless",
			CoreID:   1,
			Port:     port,
		}
		if err := db.Create(&inbound).Error; err != nil {
			t.Fatalf("Failed to create test inbound %d: %v", i, err)
		}
	}

	// Allocate multiple ports and ensure they don't conflict
	allocatedPorts := make(map[int]bool)
	for i := 0; i < 5; i++ {
		port, err := pm.AllocatePort()
		if err != nil {
			t.Fatalf("Failed to allocate port %d: %v", i, err)
		}

		if allocatedPorts[port] {
			t.Errorf("Duplicate port allocated: %d", port)
		}
		allocatedPorts[port] = true

		// Should not be in used ports
		for _, used := range ports {
			if port == used {
				t.Errorf("Allocated port %d conflicts with existing inbound", port)
			}
		}
	}
}

func TestMain(m *testing.M) {
	// Set up test environment
	code := m.Run()
	os.Exit(code)
}
