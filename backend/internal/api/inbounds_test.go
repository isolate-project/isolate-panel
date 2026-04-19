package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/haproxy"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupInboundsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Inbound{},
		&models.Core{},
		&models.User{},
		&models.UserInboundMapping{},
	))
	return db
}

func setupInboundsApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupInboundsTestDB(t)
	svc := services.NewInboundService(db, nil, nil) // nil lifecycle + nil portManager
	pm := services.NewPortManager(db)
	validator := haproxy.NewPortValidator(db)
	handler := NewInboundsHandler(svc, pm, validator, db)

	app := fiber.New()
	app.Get("/inbounds", handler.ListInbounds)
	app.Get("/inbounds/check-port", handler.CheckPort)
	app.Post("/inbounds", handler.CreateInbound)
	app.Get("/inbounds/:id", handler.GetInbound)
	app.Put("/inbounds/:id", handler.UpdateInbound)
	app.Delete("/inbounds/:id", handler.DeleteInbound)
	app.Get("/cores/:core_id/inbounds", handler.GetInboundsByCore)
	app.Post("/inbounds/:id/users", handler.GetInboundUsers)
	app.Get("/inbounds/:id/users", handler.GetInboundUsers)
	return app, db
}

// --- ListInbounds ---

func TestInboundsHandler_ListInbounds_Empty(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInboundsHandler_ListInbounds_InvalidCoreID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds?core_id=abc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_ListInbounds_FilterByEnabled(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds?is_enabled=true", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- CreateInbound (validation) ---

func TestInboundsHandler_CreateInbound_MissingName(t *testing.T) {
	app, _ := setupInboundsApp(t)

	body, _ := json.Marshal(models.Inbound{
		Protocol: "vmess",
		Port:     1080,
		CoreID:   1,
	})
	req, _ := http.NewRequest(http.MethodPost, "/inbounds", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_CreateInbound_InvalidBody(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/inbounds", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- GetInbound ---

func TestInboundsHandler_GetInbound_NotFound(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds/99999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestInboundsHandler_GetInbound_InvalidID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds/abc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- UpdateInbound ---

func TestInboundsHandler_UpdateInbound_InvalidID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	body, _ := json.Marshal(map[string]interface{}{"name": "updated"})
	req, _ := http.NewRequest(http.MethodPut, "/inbounds/bad-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_UpdateInbound_InvalidBody(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodPut, "/inbounds/1", bytes.NewBufferString("bad-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- DeleteInbound ---

func TestInboundsHandler_DeleteInbound_InvalidID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/inbounds/xyz", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_DeleteInbound_NotFound(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/inbounds/99999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- GetInboundsByCore ---

func TestInboundsHandler_GetInboundsByCore_InvalidID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/cores/bad/inbounds", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_GetInboundsByCore_ValidID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/cores/1/inbounds", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- GetInboundUsers ---

func TestInboundsHandler_GetInboundUsers_InvalidID(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds/abc/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- CheckPort ---

func TestInboundsHandler_CheckPort_MissingParam(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds/check-port", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_CheckPort_InvalidPort(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds/check-port?port=abc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestInboundsHandler_CheckPort_ValidPort(t *testing.T) {
	app, _ := setupInboundsApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/inbounds/check-port?port=8888", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "available")
}
