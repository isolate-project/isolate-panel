package scheduler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	
	err = db.AutoMigrate(&models.Setting{}, &models.Backup{})
	if err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	
	return db
}

func TestBackupScheduler_UpdateSchedule(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	dataDir := t.TempDir()
	
	// Create required directories
	os.MkdirAll(filepath.Join(backupDir, "db"), 0755)
	
	backupService := services.NewBackupService(db, backupDir, dataDir)
	scheduler := NewBackupScheduler(db, backupService)
	defer scheduler.Stop()
	
	// Test updating with empty schedule
	err := scheduler.UpdateSchedule("")
	if err != nil {
		t.Errorf("UpdateSchedule(\"\") failed: %v", err)
	}
	
	// Test updating with invalid schedule
	err = scheduler.UpdateSchedule("invalid cron")
	if err == nil {
		t.Errorf("expected error for invalid cron")
	}
	
	// Test updating with valid schedule (run every hour)
	err = scheduler.UpdateSchedule("@hourly")
	if err != nil {
		t.Errorf("UpdateSchedule(\"@hourly\") failed: %v", err)
	}
	
	// Check schedule returned
	schedule, err := scheduler.GetSchedule()
	if err != nil {
		t.Errorf("GetSchedule() failed: %v", err)
	}
	if schedule != "@hourly" {
		t.Errorf("expected schedule @hourly, got %s", schedule)
	}
	
	// Check next run is returned
	nextRun, err := scheduler.GetNextRun()
	if err != nil {
		t.Errorf("GetNextRun() failed: %v", err)
	}
	if nextRun == nil {
		t.Errorf("expected non-nil nextRun")
	}
}

func TestBackupScheduler_Initialize(t *testing.T) {
	db := setupTestDB(t)
	backupDir := t.TempDir()
	dataDir := t.TempDir()
	
	backupService := services.NewBackupService(db, backupDir, dataDir)
	// Seed a schedule directly to the db via the service
	err := backupService.SetSchedule("@daily")
	if err != nil {
		t.Fatalf("failed to set initial schedule: %v", err)
	}
	
	scheduler := NewBackupScheduler(db, backupService)
	defer scheduler.Stop()
	
	err = scheduler.Initialize()
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
	
	// Ensure the cron is loaded
	nextRun, err := scheduler.GetNextRun()
	if err != nil || nextRun == nil {
		t.Errorf("expected nextRun to be populated after initialize")
	}
}
