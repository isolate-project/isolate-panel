package services

import (
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

func setupBackupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate schema
	err = db.AutoMigrate(&models.Backup{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestBackupService_Initialize(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir

	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)

	err := service.Initialize()
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	// Check if backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}

	// Check if encryption key was generated
	keyPath := filepath.Join(dataDir, ".backup_key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Encryption key was not created")
	}
}

func TestBackupService_GetEncryptionKey(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir

	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	service.Initialize()

	key := service.GetEncryptionKey()
	if key == "" {
		t.Error("GetEncryptionKey() returned empty key")
	}

	// Key should be base64 encoded (32 bytes = 44 base64 chars)
	if len(key) != 44 {
		t.Errorf("GetEncryptionKey() length = %d, want 44", len(key))
	}
}

func TestBackupService_ListBackups(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir

	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	service.Initialize()

	// Create test backups
	backups := []models.Backup{
		{Filename: "backup1.tar.gz", FilePath: "/path/backup1.tar.gz", Status: models.BackupStatusCompleted, BackupType: models.BackupTypeManual},
		{Filename: "backup2.tar.gz", FilePath: "/path/backup2.tar.gz", Status: models.BackupStatusCompleted, BackupType: models.BackupTypeScheduled},
	}

	for i := range backups {
		db.Create(&backups[i])
	}

	result, err := service.ListBackups()
	if err != nil {
		t.Errorf("ListBackups() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("ListBackups() count = %d, want 2", len(result))
	}
}

func TestBackupService_GetBackup(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir

	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	service.Initialize()

	// Create test backup
	backup := models.Backup{
		Filename:   "backup1.tar.gz",
		FilePath:   "/path/backup1.tar.gz",
		Status:     models.BackupStatusCompleted,
		BackupType: models.BackupTypeManual,
	}
	db.Create(&backup)

	result, err := service.GetBackup(backup.ID)
	if err != nil {
		t.Errorf("GetBackup() error = %v", err)
	}

	if result.Filename != "backup1.tar.gz" {
		t.Errorf("GetBackup() filename = %v, want backup1.tar.gz", result.Filename)
	}

	// Test non-existent backup
	_, err = service.GetBackup(999)
	if err == nil {
		t.Error("GetBackup() expected error for non-existent backup")
	}
}

func TestBackupService_DeleteBackup(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir

	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	service.Initialize()

	// Create test backup file
	backupFile := filepath.Join(backupDir, "test.tar.gz")
	os.WriteFile(backupFile, []byte("test"), 0644)

	backup := models.Backup{
		Filename:   "test.tar.gz",
		FilePath:   backupFile,
		Status:     models.BackupStatusCompleted,
		BackupType: models.BackupTypeManual,
	}
	db.Create(&backup)

	err := service.DeleteBackup(backup.ID)
	if err != nil {
		t.Errorf("DeleteBackup() error = %v", err)
	}

	// Check if backup was deleted from database
	var count int64
	db.Model(&models.Backup{}).Count(&count)
	if count != 0 {
		t.Errorf("DeleteBackup() count = %d, want 0", count)
	}

	// Check if file was deleted
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Error("DeleteBackup() file was not deleted")
	}
}

func TestBackupService_Schedule(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir

	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	service.Initialize()

	// Test GetSchedule when empty
	schedule, err := service.GetSchedule()
	if err != nil {
		t.Errorf("GetSchedule() error = %v", err)
	}
	if schedule != "" {
		t.Errorf("GetSchedule() = %v, want empty", schedule)
	}

	// Test SetSchedule
	err = service.SetSchedule("0 3 * * *")
	if err != nil {
		t.Errorf("SetSchedule() error = %v", err)
	}

	// Verify schedule was set
	schedule, err = service.GetSchedule()
	if err != nil {
		t.Errorf("GetSchedule() error = %v", err)
	}
	if schedule != "0 3 * * *" {
		t.Errorf("GetSchedule() = %v, want 0 3 * * *", schedule)
	}

	// Test clearing schedule
	err = service.SetSchedule("")
	if err != nil {
		t.Errorf("SetSchedule() clear error = %v", err)
	}

	schedule, err = service.GetSchedule()
	if err != nil {
		t.Errorf("GetSchedule() error = %v", err)
	}
	if schedule != "" {
		t.Errorf("GetSchedule() after clear = %v, want empty", schedule)
	}
}

func TestBackupService_RotateBackups(t *testing.T) {
	db := setupBackupTestDB(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := tmpDir
	settingsService := NewSettingsService(db)

	service := NewBackupService(db, settingsService, backupDir, dataDir)
	service.Initialize()

	// Create 5 test backups
	for i := 0; i < 5; i++ {
		backup := models.Backup{
			Filename:   filepath.Join(backupDir, "test.tar.gz"),
			FilePath:   filepath.Join(backupDir, "test.tar.gz"),
			Status:     models.BackupStatusCompleted,
			BackupType: models.BackupTypeManual,
		}
		db.Create(&backup)
	}

	// Call rotate
	err := service.rotateBackups()
	if err != nil {
		t.Errorf("rotateBackups() error = %v", err)
	}

	// Check if only 3 backups remain
	var count int64
	db.Model(&models.Backup{}).Count(&count)
	if count != 3 {
		t.Errorf("rotateBackups() count = %d, want 3", count)
	}
}
