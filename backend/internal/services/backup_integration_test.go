package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupStreamingIntegrity(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(backupDir, 0755)
	os.MkdirAll(dataDir, 0755)

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Backup{}, &models.Setting{})
	
	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// 1. Create a "large" dummy file to test streaming (e.g., 1MB)
	sourceFile := filepath.Join(dataDir, "test_large.bin")
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(sourceFile, content, 0644)

	// 2. Test Encryption (Streaming)
	encryptedFile := filepath.Join(backupDir, "test.enc")
	err = service.encryptFile(sourceFile, encryptedFile)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Verify encrypted file exists and is different from source
	encInfo, _ := os.Stat(encryptedFile)
	if encInfo.Size() == 0 {
		t.Error("Encrypted file is empty")
	}

	// 3. Test Decryption (Streaming)
	decryptedFile := filepath.Join(dataDir, "test_decrypted.bin")
	err = service.decryptFile(encryptedFile, decryptedFile)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// 4. Verify Integrity
	decryptedContent, _ := os.ReadFile(decryptedFile)
	if !bytes.Equal(content, decryptedContent) {
		t.Error("Decrypted content does not match original source")
	}

	// 5. Test Gzip Compression Integrity
	archiveFile := filepath.Join(backupDir, "test.tar.gz")
	
	// Create a dummy structure for tar
	subDir := filepath.Join(dataDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("hello world"), 0644)

	err = service.createTarArchive(dataDir, archiveFile)
	if err != nil {
		t.Fatalf("Archiving failed: %v", err)
	}

	// Verify it's actually a gzip file
	f, _ := os.Open(archiveFile)
	header := make([]byte, 2)
	f.Read(header)
	f.Close()
	if header[0] != 0x1f || header[1] != 0x8b {
		t.Error("Archive is not a valid gzip file (missing magic bytes)")
	}

	// 6. Test Extraction
	extractDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(extractDir, 0755)
	err = service.extractTarArchive(archiveFile, extractDir)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// Verify extracted content
	extractedFile := filepath.Join(extractDir, "subdir", "file1.txt")
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Errorf("Extracted file missing: %s", extractedFile)
	} else {
		extractedContent, _ := os.ReadFile(extractedFile)
		if string(extractedContent) != "hello world" {
			t.Errorf("Extracted content mismatch: got %s, want hello world", string(extractedContent))
		}
	}
}

func TestBackupService_RetentionIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(backupDir, 0755)
	os.MkdirAll(dataDir, 0755)

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Backup{}, &models.Setting{})
	
	settingsService := NewSettingsService(db)
	service := NewBackupService(db, settingsService, backupDir, dataDir)
	
	// Set retention to 2
	db.Create(&models.Setting{Key: "backup_retention_count", Value: "2"})

	// Create 5 fake backups in DB and FS
	for i := 1; i <= 5; i++ {
		path := filepath.Join(backupDir, "test_backup.tar.gz")
		os.WriteFile(path, []byte("fake"), 0644)
		
		backup := models.Backup{
			Filename: "backup.tar.gz",
			FilePath: path,
			Status:   models.BackupStatusCompleted,
		}
		db.Create(&backup)
	}

	// Run rotation
	err := service.rotateBackups()
	if err != nil {
		t.Fatalf("Rotation failed: %v", err)
	}

	// Verify only 2 remain
	var count int64
	db.Model(&models.Backup{}).Count(&count)
	if count != 2 {
		t.Errorf("Retention failed: got %d backups, want 2", count)
	}
}
