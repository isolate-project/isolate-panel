package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Setting{}))
	return db
}

func setupSettingsApp(t *testing.T) *fiber.App {
	t.Helper()
	db := setupSettingsTestDB(t)
	svc := services.NewSettingsService(db)
	handler := NewSettingsHandler(svc, nil) // nil = no TrafficCollector

	app := fiber.New()
	app.Get("/settings", handler.GetAllSettings)
	app.Put("/settings", handler.UpdateSettings)
	app.Get("/settings/monitoring", handler.GetMonitoring)
	app.Put("/settings/monitoring", handler.UpdateMonitoring)
	return app
}

// --- GetAllSettings ---

func TestSettingsHandler_GetAllSettings_Empty(t *testing.T) {
	app := setupSettingsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/settings", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, true, result["success"])
}

// --- GetMonitoring ---

func TestSettingsHandler_GetMonitoring_DefaultsToLite(t *testing.T) {
	app := setupSettingsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/settings/monitoring", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "lite", result["mode"])
	assert.Equal(t, true, result["success"])
}

// --- UpdateMonitoring ---

func TestSettingsHandler_UpdateMonitoring_Full(t *testing.T) {
	app := setupSettingsApp(t)

	body, _ := json.Marshal(map[string]string{"mode": "full"})
	req, _ := http.NewRequest(http.MethodPut, "/settings/monitoring", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, true, result["success"])
	assert.Equal(t, "full", result["mode"])
}

func TestSettingsHandler_UpdateMonitoring_Lite(t *testing.T) {
	app := setupSettingsApp(t)

	body, _ := json.Marshal(map[string]string{"mode": "lite"})
	req, _ := http.NewRequest(http.MethodPut, "/settings/monitoring", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSettingsHandler_UpdateMonitoring_InvalidMode(t *testing.T) {
	app := setupSettingsApp(t)

	body, _ := json.Marshal(map[string]string{"mode": "ultra"})
	req, _ := http.NewRequest(http.MethodPut, "/settings/monitoring", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSettingsHandler_UpdateMonitoring_InvalidBody(t *testing.T) {
	app := setupSettingsApp(t)

	req, _ := http.NewRequest(http.MethodPut, "/settings/monitoring", bytes.NewBufferString("bad-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- UpdateSettings ---

func TestSettingsHandler_UpdateSettings_Success(t *testing.T) {
	app := setupSettingsApp(t)

	body, _ := json.Marshal(map[string]interface{}{
		"settings": map[string]string{
			"some_key": "some_value",
		},
	})
	req, _ := http.NewRequest(http.MethodPut, "/settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	// UpdateSettings uses a transaction; on empty DB it will silently succeed (0 rows updated is fine)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSettingsHandler_UpdateSettings_EmptyBody(t *testing.T) {
	app := setupSettingsApp(t)

	req, _ := http.NewRequest(http.MethodPut, "/settings", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Monitoring persistence round-trip ---

func TestSettingsHandler_MonitoringRoundTrip(t *testing.T) {
	app := setupSettingsApp(t)

	// Set to full
	body, _ := json.Marshal(map[string]string{"mode": "full"})
	req1, _ := http.NewRequest(http.MethodPut, "/settings/monitoring", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	// Read back
	req2, _ := http.NewRequest(http.MethodGet, "/settings/monitoring", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result))
	assert.Equal(t, "full", result["mode"])
}
