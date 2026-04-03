package api

import (
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

func setupStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Core{},
		&models.Inbound{},
		&models.UserInboundMapping{},
		&models.ActiveConnection{},
	))
	// Create traffic_stats table manually (raw table used by StatsHandler)
	db.Exec(`CREATE TABLE IF NOT EXISTS traffic_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		inbound_id INTEGER,
		upload INTEGER DEFAULT 0,
		download INTEGER DEFAULT 0,
		total INTEGER DEFAULT 0,
		granularity TEXT DEFAULT 'daily',
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return db
}

func setupStatsApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupStatsTestDB(t)
	ct := services.NewConnectionTracker(db, 0, "", "", "", "", "")
	handler := NewStatsHandler(db, nil, ct) // nil TrafficCollector

	app := fiber.New()
	app.Get("/stats/users/:user_id", handler.GetUserTrafficStats)
	app.Get("/stats/connections", handler.GetActiveConnections)
	app.Delete("/stats/users/:user_id/disconnect", handler.DisconnectUser)
	app.Post("/stats/users/:user_id/kick", handler.KickUser)
	app.Get("/stats/dashboard", handler.GetDashboardStats)
	app.Get("/stats/traffic", handler.GetTrafficOverview)
	app.Get("/stats/top-users", handler.GetTopUsers)
	return app, db
}

// --- GetUserTrafficStats ---

func TestStatsHandler_GetUserTrafficStats_EmptyDB(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/users/1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(1), result["user_id"])
	assert.Equal(t, "daily", result["granularity"])
}

func TestStatsHandler_GetUserTrafficStats_InvalidID(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/users/abc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStatsHandler_GetUserTrafficStats_WithGranularity(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/users/1?granularity=hourly&days=7", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- GetActiveConnections ---

func TestStatsHandler_GetActiveConnections_Empty(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/connections", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStatsHandler_GetActiveConnections_InvalidUserID(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/connections?user_id=bad", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- DisconnectUser ---

func TestStatsHandler_DisconnectUser_InvalidID(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/stats/users/nan/disconnect", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- KickUser ---

func TestStatsHandler_KickUser_InvalidID(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/stats/users/abc/kick", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStatsHandler_KickUser_NotFound(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/stats/users/99999/kick", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- GetDashboardStats ---

func TestStatsHandler_GetDashboardStats(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/dashboard", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "total_users")
	assert.Contains(t, result, "cores_running")
	assert.Contains(t, result, "total_traffic")
}

// --- GetTrafficOverview ---

func TestStatsHandler_GetTrafficOverview_Default(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/traffic", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(7), result["days"])
}

func TestStatsHandler_GetTrafficOverview_CustomDays(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/traffic?days=30&granularity=daily", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStatsHandler_GetTrafficOverview_InvalidDaysClipped(t *testing.T) {
	app, _ := setupStatsApp(t)

	// days=999 should be clipped to 7 (out of range 0..365 → default 7)
	req, _ := http.NewRequest(http.MethodGet, "/stats/traffic?days=999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- GetTopUsers ---

func TestStatsHandler_GetTopUsers_Default(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/top-users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "users")
	assert.Contains(t, result, "total")
}

func TestStatsHandler_GetTopUsers_CustomLimit(t *testing.T) {
	app, _ := setupStatsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/stats/top-users?limit=5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
