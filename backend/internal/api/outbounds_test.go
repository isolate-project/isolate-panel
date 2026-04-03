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

func setupOutboundsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Outbound{},
		&models.Core{},
	))
	return db
}

func setupOutboundsApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupOutboundsTestDB(t)
	svc := services.NewOutboundService(db, nil) // nil configService
	handler := NewOutboundsHandler(svc)

	app := fiber.New()
	app.Get("/outbounds", handler.ListOutbounds)
	app.Post("/outbounds", handler.CreateOutbound)
	app.Get("/outbounds/:id", handler.GetOutbound)
	app.Put("/outbounds/:id", handler.UpdateOutbound)
	app.Delete("/outbounds/:id", handler.DeleteOutbound)
	return app, db
}

// --- ListOutbounds ---

func TestOutboundsHandler_ListOutbounds_Empty(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/outbounds", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 0, len(result))
}

func TestOutboundsHandler_ListOutbounds_InvalidCoreID(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/outbounds?core_id=notanumber", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOutboundsHandler_ListOutbounds_WithProtocolFilter(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/outbounds?protocol=socks", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- CreateOutbound (validation) ---

func TestOutboundsHandler_CreateOutbound_MissingName(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	body, _ := json.Marshal(models.Outbound{
		Protocol: "socks",
		CoreID:   1,
	})
	req, _ := http.NewRequest(http.MethodPost, "/outbounds", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOutboundsHandler_CreateOutbound_InvalidBody(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/outbounds", bytes.NewBufferString("bad-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- GetOutbound ---

func TestOutboundsHandler_GetOutbound_NotFound(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/outbounds/99999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOutboundsHandler_GetOutbound_InvalidID(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/outbounds/nan", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- UpdateOutbound ---

func TestOutboundsHandler_UpdateOutbound_InvalidID(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	body, _ := json.Marshal(map[string]interface{}{"name": "new"})
	req, _ := http.NewRequest(http.MethodPut, "/outbounds/bad-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOutboundsHandler_UpdateOutbound_InvalidBody(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodPut, "/outbounds/1", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOutboundsHandler_UpdateOutbound_NotFound(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	body, _ := json.Marshal(map[string]interface{}{"name": "updated"})
	req, _ := http.NewRequest(http.MethodPut, "/outbounds/99999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- DeleteOutbound ---

func TestOutboundsHandler_DeleteOutbound_InvalidID(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/outbounds/xyz", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOutboundsHandler_DeleteOutbound_NotFound(t *testing.T) {
	app, _ := setupOutboundsApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/outbounds/99999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
