package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/scheduler"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupBackupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Backup{},
		&models.Setting{},
	))
	return db
}

func setupBackupApp(t *testing.T) (*fiber.App, *services.BackupService) {
	t.Helper()
	db := setupBackupTestDB(t)

	// Create temp dirs
	tmpDir, err := os.MkdirTemp("", "backup-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	settingsService := services.NewSettingsService(db)
	backupService := services.NewBackupService(db, settingsService, tmpDir, tmpDir)
	require.NoError(t, backupService.Initialize())

	backupScheduler := scheduler.NewBackupScheduler(db, backupService)
	handler := NewBackupHandler(backupService, backupScheduler)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		c.Locals("is_super_admin", true)
		return c.Next()
	})
	
	handler.RegisterRoutes(app)
	return app, backupService
}

func TestBackupHandler_ListBackups(t *testing.T) {
	app, _ := setupBackupApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/backups/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotNil(t, result["data"])
}

func TestBackupHandler_CreateBackup(t *testing.T) {
	app, _ := setupBackupApp(t)

	body, _ := json.Marshal(services.BackupRequest{
		Type:              "manual",
		EncryptionEnabled: false,
		IncludeCores:      false,
		IncludeCerts:      false,
		IncludeWARP:       false,
		IncludeGeo:        false,
	})
	req, _ := http.NewRequest(http.MethodPost, "/backups/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestBackupHandler_GetBackup(t *testing.T) {
	app, svc := setupBackupApp(t)

	mockBackup, err := svc.CreateBackup(services.BackupRequest{})
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond) // Give the goroutine time to finish creating backup
	
	req, _ := http.NewRequest(http.MethodGet, "/backups/"+uint2str(mockBackup.ID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBackupHandler_DeleteBackup(t *testing.T) {
	app, svc := setupBackupApp(t)

	mockBackup, err := svc.CreateBackup(services.BackupRequest{})
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond) // Give the goroutine time to finish creating backup
	
	req, _ := http.NewRequest(http.MethodDelete, "/backups/"+uint2str(mockBackup.ID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
