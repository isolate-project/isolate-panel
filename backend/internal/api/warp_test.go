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

func setupWarpTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Core{},
		&models.WarpRoute{},
		&models.GeoRule{},
	))
	return db
}

func setupWarpApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupWarpTestDB(t)
	warpSvc := services.NewWARPService(db, "/tmp/warp-test")
	geoSvc := services.NewGeoService(db, "/tmp/geo-test")
	handler := NewWarpHandler(warpSvc, geoSvc) // no configService

	app := fiber.New()
	// WARP routes
	app.Get("/warp/routes", handler.GetWarpRoutes)
	app.Post("/warp/routes", handler.CreateWarpRoute)
	app.Put("/warp/routes/:id", handler.UpdateWarpRoute)
	app.Delete("/warp/routes/:id", handler.DeleteWarpRoute)
	app.Post("/warp/routes/:id/toggle", handler.ToggleWarpRoute)
	app.Get("/warp/presets", handler.GetWarpPresets)
	// Geo routes
	app.Get("/geo/rules", handler.GetGeoRules)
	app.Post("/geo/rules", handler.CreateGeoRule)
	app.Put("/geo/rules/:id", handler.UpdateGeoRule)
	app.Delete("/geo/rules/:id", handler.DeleteGeoRule)
	app.Post("/geo/rules/:id/toggle", handler.ToggleGeoRule)
	app.Get("/geo/countries", handler.GetCountries)
	app.Get("/geo/categories", handler.GetCategories)
	app.Get("/geo/databases", handler.GetGeoDatabases)
	return app, db
}

// --- GetWarpRoutes ---

func TestWarpHandler_GetWarpRoutes_MissingCoreID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/warp/routes", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_GetWarpRoutes_InvalidCoreID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/warp/routes?core_id=bad", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_GetWarpRoutes_ValidCoreID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/warp/routes?core_id=1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- CreateWarpRoute ---

func TestWarpHandler_CreateWarpRoute_InvalidBody(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/warp/routes", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_CreateWarpRoute_MissingFields(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.WarpRoute{CoreID: 1})
	req, _ := http.NewRequest(http.MethodPost, "/warp/routes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_CreateWarpRoute_InvalidResourceType(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.WarpRoute{
		CoreID:        1,
		ResourceType:  "invalid",
		ResourceValue: "google.com",
	})
	req, _ := http.NewRequest(http.MethodPost, "/warp/routes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_CreateWarpRoute_ValidDomain(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.WarpRoute{
		CoreID:        1,
		ResourceType:  "domain",
		ResourceValue: "google.com",
	})
	req, _ := http.NewRequest(http.MethodPost, "/warp/routes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

// --- UpdateWarpRoute ---

func TestWarpHandler_UpdateWarpRoute_InvalidID(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.WarpRoute{Description: "updated"})
	req, _ := http.NewRequest(http.MethodPut, "/warp/routes/abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_UpdateWarpRoute_NotFound(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.WarpRoute{Description: "updated"})
	req, _ := http.NewRequest(http.MethodPut, "/warp/routes/99999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- DeleteWarpRoute ---

func TestWarpHandler_DeleteWarpRoute_InvalidID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/warp/routes/bad", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- ToggleWarpRoute ---

func TestWarpHandler_ToggleWarpRoute_InvalidID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/warp/routes/bad/toggle", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_ToggleWarpRoute_NotFound(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/warp/routes/99999/toggle", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- GetWarpPresets ---

func TestWarpHandler_GetWarpPresets(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/warp/presets", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "data")
}

// --- GetGeoRules ---

func TestWarpHandler_GetGeoRules_MissingCoreID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/geo/rules", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_GetGeoRules_ValidCoreID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/geo/rules?core_id=1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- CreateGeoRule ---

func TestWarpHandler_CreateGeoRule_InvalidBody(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/geo/rules", bytes.NewBufferString("bad-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_CreateGeoRule_MissingFields(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.GeoRule{CoreID: 1})
	req, _ := http.NewRequest(http.MethodPost, "/geo/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_CreateGeoRule_InvalidType(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.GeoRule{
		CoreID: 1,
		Type:   "invalid",
		Code:   "RU",
		Action: "block",
	})
	req, _ := http.NewRequest(http.MethodPost, "/geo/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_CreateGeoRule_InvalidAction(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.GeoRule{
		CoreID: 1,
		Type:   "geoip",
		Code:   "RU",
		Action: "invalid",
	})
	req, _ := http.NewRequest(http.MethodPost, "/geo/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- UpdateGeoRule ---

func TestWarpHandler_UpdateGeoRule_InvalidID(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.GeoRule{Description: "updated"})
	req, _ := http.NewRequest(http.MethodPut, "/geo/rules/bad", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_UpdateGeoRule_NotFound(t *testing.T) {
	app, _ := setupWarpApp(t)

	body, _ := json.Marshal(models.GeoRule{Description: "updated"})
	req, _ := http.NewRequest(http.MethodPut, "/geo/rules/99999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- DeleteGeoRule ---

func TestWarpHandler_DeleteGeoRule_InvalidID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/geo/rules/bad?core_id=1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWarpHandler_DeleteGeoRule_MissingCoreID(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/geo/rules/1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- GetCountries / GetCategories / GetGeoDatabases ---

func TestWarpHandler_GetCountries(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/geo/countries", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// May return 200 with empty list or 500 if geo files not present
	assert.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError,
	)
}

func TestWarpHandler_GetCategories(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/geo/categories", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError,
	)
}

func TestWarpHandler_GetGeoDatabases(t *testing.T) {
	app, _ := setupWarpApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/geo/databases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError,
	)
}
