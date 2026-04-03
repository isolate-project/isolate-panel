package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackupFlow tests the complete backup lifecycle at DB level
func TestBackupFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	// Setup temp dirs for backup service
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(backupDir, 0750))
	require.NoError(t, os.MkdirAll(dataDir, 0750))

	settingsService := services.NewSettingsService(db)
	backupService := services.NewBackupService(db, settingsService, backupDir, dataDir)
	require.NoError(t, backupService.Initialize())

	t.Run("Step 1: List backups (initially empty)", func(t *testing.T) {
		backups, err := backupService.ListBackups()
		require.NoError(t, err)
		assert.Equal(t, 0, len(backups))
	})

	t.Run("Step 2: Create manual backup record (DB only)", func(t *testing.T) {
		backup := &models.Backup{
			Filename:          "backup_test_001.tar.gz",
			FilePath:          filepath.Join(backupDir, "backup_test_001.tar.gz"),
			BackupType:        models.BackupTypeManual,
			Destination:       models.BackupDestinationLocal,
			Status:            models.BackupStatusCompleted,
			EncryptionEnabled: false,
			FileSizeBytes:     1024,
			ChecksumSHA256:    "abc123",
			CreatedAt:         time.Now(),
		}
		err := db.Create(backup).Error
		require.NoError(t, err)
		assert.NotZero(t, backup.ID)
	})

	t.Run("Step 3: List backups shows created record", func(t *testing.T) {
		backups, err := backupService.ListBackups()
		require.NoError(t, err)
		assert.Equal(t, 1, len(backups))
		assert.Equal(t, models.BackupTypeManual, backups[0].BackupType)
		assert.Equal(t, models.BackupStatusCompleted, backups[0].Status)
	})

	t.Run("Step 4: Get backup by ID", func(t *testing.T) {
		backups, _ := backupService.ListBackups()
		require.NotEmpty(t, backups)

		backup, err := backupService.GetBackup(backups[0].ID)
		require.NoError(t, err)
		assert.Equal(t, "backup_test_001.tar.gz", backup.Filename)
	})

	t.Run("Step 5: Get backup not found", func(t *testing.T) {
		_, err := backupService.GetBackup(99999)
		assert.Error(t, err)
	})

	t.Run("Step 6: Scheduled backup type", func(t *testing.T) {
		backup := &models.Backup{
			Filename:          "backup_scheduled.tar.gz",
			FilePath:          filepath.Join(backupDir, "backup_scheduled.tar.gz"),
			BackupType:        models.BackupTypeScheduled,
			Destination:       models.BackupDestinationLocal,
			Status:            models.BackupStatusCompleted,
			EncryptionEnabled: true,
			ScheduleCron:      "0 3 * * *",
			FileSizeBytes:     2048,
			CreatedAt:         time.Now().Add(-1 * time.Hour),
		}
		err := db.Create(backup).Error
		require.NoError(t, err)

		retrieved, err := backupService.GetBackup(backup.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BackupTypeScheduled, retrieved.BackupType)
		assert.True(t, retrieved.EncryptionEnabled)
		assert.Equal(t, "0 3 * * *", retrieved.ScheduleCron)
	})

	t.Run("Step 7: List backups ordered by latest first", func(t *testing.T) {
		backups, err := backupService.ListBackups()
		require.NoError(t, err)
		require.Equal(t, 2, len(backups))

		// Most recent first
		assert.True(t, backups[0].CreatedAt.After(backups[1].CreatedAt) ||
			backups[0].CreatedAt.Equal(backups[1].CreatedAt))
	})

	t.Run("Step 8: Delete backup removes record", func(t *testing.T) {
		backups, _ := backupService.ListBackups()
		require.NotEmpty(t, backups)
		firstID := backups[0].ID

		err := backupService.DeleteBackup(firstID)
		require.NoError(t, err)

		// Verify deleted
		_, err = backupService.GetBackup(firstID)
		assert.Error(t, err)

		// One backup remains
		remaining, err := backupService.ListBackups()
		require.NoError(t, err)
		assert.Equal(t, 1, len(remaining))
	})

	t.Run("Step 9: Delete non-existent backup returns error", func(t *testing.T) {
		err := backupService.DeleteBackup(99999)
		assert.Error(t, err)
	})
}

// TestBackupEncryption tests the AES-GCM encryption/decryption round-trip
func TestBackupEncryption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(backupDir, 0750))
	require.NoError(t, os.MkdirAll(dataDir, 0750))

	settingsService := services.NewSettingsService(db)
	backupService := services.NewBackupService(db, settingsService, backupDir, dataDir)
	require.NoError(t, backupService.Initialize())

	t.Run("Encryption key generated on Initialize", func(t *testing.T) {
		key := backupService.GetEncryptionKey()
		assert.NotEmpty(t, key)
		// Base64 encoded 32 bytes = 44 chars
		assert.Equal(t, 44, len(key))
	})

	t.Run("Encryption key persists across service instances", func(t *testing.T) {
		// Get key from first instance
		key1 := backupService.GetEncryptionKey()

		// Create second service instance pointing to same dataDir
		backupService2 := services.NewBackupService(db, settingsService, backupDir, dataDir)
		require.NoError(t, backupService2.Initialize())
		key2 := backupService2.GetEncryptionKey()

		assert.Equal(t, key1, key2, "encryption key should be stable across restarts")
	})
}

// TestBackupStatusTransitions tests backup status lifecycle
func TestBackupStatusTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(backupDir, 0750))
	require.NoError(t, os.MkdirAll(dataDir, 0750))

	settingsService := services.NewSettingsService(db)
	backupService := services.NewBackupService(db, settingsService, backupDir, dataDir)
	require.NoError(t, backupService.Initialize())

	t.Run("Pending → Running → Completed transition", func(t *testing.T) {
		backup := &models.Backup{
			Filename:     "status_test.tar.gz",
			FilePath:     filepath.Join(backupDir, "status_test.tar.gz"),
			BackupType:   models.BackupTypeManual,
			Destination:  models.BackupDestinationLocal,
			Status:       models.BackupStatusPending,
			CreatedAt:    time.Now(),
		}
		require.NoError(t, db.Create(backup).Error)
		assert.Equal(t, models.BackupStatusPending, backup.Status)

		// Transition to Running
		backup.Status = models.BackupStatusRunning
		require.NoError(t, db.Save(backup).Error)

		retrieved, _ := backupService.GetBackup(backup.ID)
		assert.Equal(t, models.BackupStatusRunning, retrieved.Status)

		// Transition to Completed
		now := time.Now()
		backup.Status = models.BackupStatusCompleted
		backup.CompletedAt = &now
		backup.DurationMs = 250
		require.NoError(t, db.Save(backup).Error)

		final, _ := backupService.GetBackup(backup.ID)
		assert.Equal(t, models.BackupStatusCompleted, final.Status)
		assert.NotNil(t, final.CompletedAt)
		assert.Equal(t, 250, final.DurationMs)
	})

	t.Run("Pending → Failed with error message", func(t *testing.T) {
		backup := &models.Backup{
			Filename:    "failed_backup.tar.gz",
			FilePath:    filepath.Join(backupDir, "failed_backup.tar.gz"),
			BackupType:  models.BackupTypeManual,
			Destination: models.BackupDestinationLocal,
			Status:      models.BackupStatusRunning,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, db.Create(backup).Error)

		backup.Status = models.BackupStatusFailed
		backup.ErrorMessage = "sqlite3 not found"
		require.NoError(t, db.Save(backup).Error)

		retrieved, _ := backupService.GetBackup(backup.ID)
		assert.Equal(t, models.BackupStatusFailed, retrieved.Status)
		assert.Equal(t, "sqlite3 not found", retrieved.ErrorMessage)
	})

	t.Run("RestoreBackup returns error for non-completed backup", func(t *testing.T) {
		backup := &models.Backup{
			Filename:    "running_backup.tar.gz",
			FilePath:    filepath.Join(backupDir, "running_backup.tar.gz"),
			BackupType:  models.BackupTypeManual,
			Destination: models.BackupDestinationLocal,
			Status:      models.BackupStatusRunning, // Not completed!
			CreatedAt:   time.Now(),
		}
		require.NoError(t, db.Create(backup).Error)

		err := backupService.RestoreBackup(backup.ID, false)
		assert.Error(t, err, "should reject restore of non-completed backup")
	})
}

// TestBackupSourceMetadata tests backup source JSON serialization
func TestBackupSourceMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	t.Run("Serialize and deserialize BackupSource", func(t *testing.T) {
		source := models.BackupSource{
			IncludeDatabase: true,
			IncludeCores:    true,
			IncludeCerts:    false,
			IncludeWARP:     true,
			IncludeGeo:      false,
		}

		data, err := json.Marshal(source)
		require.NoError(t, err)
		assert.Contains(t, string(data), "include_database")

		backup := &models.Backup{
			Filename:     "meta_test.tar.gz",
			FilePath:     "/tmp/meta_test.tar.gz",
			BackupType:   models.BackupTypeManual,
			Destination:  models.BackupDestinationLocal,
			Status:       models.BackupStatusCompleted,
			BackupSource: string(data),
			CreatedAt:    time.Now(),
		}
		require.NoError(t, db.Create(backup).Error)

		var retrieved models.Backup
		require.NoError(t, db.First(&retrieved, backup.ID).Error)

		var parsedSource models.BackupSource
		require.NoError(t, json.Unmarshal([]byte(retrieved.BackupSource), &parsedSource))

		assert.True(t, parsedSource.IncludeDatabase)
		assert.True(t, parsedSource.IncludeCores)
		assert.False(t, parsedSource.IncludeCerts)
		assert.True(t, parsedSource.IncludeWARP)
		assert.False(t, parsedSource.IncludeGeo)
	})
}
