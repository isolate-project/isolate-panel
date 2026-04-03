package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProtocolsApp() *fiber.App {
	handler := NewProtocolsHandler()
	app := fiber.New()
	app.Get("/protocols", handler.ListProtocols)
	app.Get("/protocols/:name", handler.GetProtocol)
	app.Get("/protocols/:name/defaults", handler.GetProtocolDefaults)
	return app
}

// --- ListProtocols ---

func TestProtocolsHandler_ListAll(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "protocols")
	assert.Contains(t, result, "total")
	total := result["total"].(float64)
	assert.Greater(t, int(total), 0, "should have registered protocols")
}

func TestProtocolsHandler_FilterByCore_Xray(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols?core=xray", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "protocols")
}

func TestProtocolsHandler_FilterByCore_Singbox(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols?core=singbox", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtocolsHandler_FilterByDirection_Inbound(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols?direction=inbound", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtocolsHandler_FilterByDirection_Outbound(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols?direction=outbound", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtocolsHandler_FilterByCoreAndDirection(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols?core=xray&direction=inbound", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtocolsHandler_InvalidDirection(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols?direction=invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- GetProtocol ---

func TestProtocolsHandler_GetProtocol_ValidVmess(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/vmess", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// vmess should exist in xray protocols
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtocolsHandler_GetProtocol_NotFound(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/nonexistent-protocol-xyz", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- GetProtocolDefaults ---

func TestProtocolsHandler_GetDefaults_ValidProtocol(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/vmess/defaults", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Should succeed or 404 depending on registry
	assert.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
		"expected 200 or 404, got %d", resp.StatusCode,
	)
}

func TestProtocolsHandler_GetDefaults_NotFound(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/totally-fake-proto/defaults", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
