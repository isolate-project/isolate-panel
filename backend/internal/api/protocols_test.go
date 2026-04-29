package api

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/protocol"
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

// --- Widget Hints Tests ---

func TestGetProtocol_ContainsWidgetHints(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/vless", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	parameters, ok := result["parameters"].(map[string]interface{})
	require.True(t, ok, "parameters should be a map")

	// Check that uuid parameter has widget field
	uuidParam, ok := parameters["uuid"].(map[string]interface{})
	require.True(t, ok, "uuid parameter should exist")
	widget, ok := uuidParam["widget"].(string)
	require.True(t, ok, "uuid parameter should have widget field")
	assert.Equal(t, "input", widget, "uuid should have input widget")

	// Check that flow parameter has widget field
	flowParam, ok := parameters["flow"].(map[string]interface{})
	require.True(t, ok, "flow parameter should exist")
	widget, ok = flowParam["widget"].(string)
	require.True(t, ok, "flow parameter should have widget field")
	assert.Equal(t, "select", widget, "flow should have select widget")
}

func TestGetProtocol_PasswordWidget(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/trojan", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	parameters, ok := result["parameters"].(map[string]interface{})
	require.True(t, ok, "parameters should be a map")

	// Check that password parameter has password widget
	passwordParam, ok := parameters["password"].(map[string]interface{})
	require.True(t, ok, "password parameter should exist")
	widget, ok := passwordParam["widget"].(string)
	require.True(t, ok, "password parameter should have widget field")
	assert.Equal(t, "password", widget, "password should have password widget")
}

func TestGetProtocol_ObfuscationPasswordWidget(t *testing.T) {
	app := setupProtocolsApp()

	req, _ := http.NewRequest(http.MethodGet, "/protocols/hysteria2", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	parameters, ok := result["parameters"].(map[string]interface{})
	require.True(t, ok, "parameters should be a map")

	// Check that obfs_password parameter has password widget
	obfsPasswordParam, ok := parameters["obfs_password"].(map[string]interface{})
	require.True(t, ok, "obfs_password parameter should exist")
	widget, ok := obfsPasswordParam["widget"].(string)
	require.True(t, ok, "obfs_password parameter should have widget field")
	assert.Equal(t, "password", widget, "obfs_password should have password widget")
}

func TestMain(m *testing.M) {
	protocol.RegisterAllProtocols()
	os.Exit(m.Run())
}
