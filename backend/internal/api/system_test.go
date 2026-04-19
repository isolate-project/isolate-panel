package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSystemApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	statsProvider := services.NewStatsClientProvider("", "", "", "", "", "")
	ct := services.NewConnectionTracker(db, 10*time.Second, statsProvider)
	cm := cores.NewCoreManager(db, "http://127.0.0.1:99999", nil)
	handler := NewSystemHandler(ct, cm)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return c.Next()
	})
	app.Get("/system/resources", handler.GetResources)
	app.Get("/system/connections", handler.GetConnections)
	app.Post("/system/emergency-cleanup", handler.EmergencyCleanup)

	return app, db
}

func setupSystemHealthApp(t *testing.T) *fiber.App {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	startTime := time.Now().Add(-5 * time.Minute)

	app := fiber.New()
	app.Get("/health", func(c fiber.Ctx) error {
		type HealthResponse struct {
			Status    string `json:"status"`
			Version   string `json:"version"`
			Uptime    string `json:"uptime"`
			Database  string `json:"database"`
			Timestamp string `json:"timestamp"`
		}
		resp := HealthResponse{
			Status:    "healthy",
			Version:   version.Version,
			Uptime:    time.Since(startTime).String(),
			Timestamp: time.Now().Format(time.RFC3339),
			Database:  "connected",
		}
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			resp.Database = "disconnected"
			resp.Status = "unhealthy"
		}
		code := fiber.StatusOK
		if resp.Status == "unhealthy" {
			code = fiber.StatusServiceUnavailable
		}
		return c.Status(code).JSON(resp)
	})

	return app
}

func setupWSTicketApp(t *testing.T) *fiber.App {
	t.Helper()
	hub := &DashboardHub{}

	app := fiber.New()
	app.Post("/ws/ticket", func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return hub.IssueWSTicket(c)
	})
	return app
}

// --- GetResources ---

func TestSystemHandler_GetResources_ReturnsOK(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/resources", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSystemHandler_GetResources_ContainsRAMFields(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/resources", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "ram_percent")
	assert.Contains(t, body, "cpu_percent")
	assert.Contains(t, body, "ram_total")
	assert.Contains(t, body, "ram_used")
	assert.Contains(t, body, "goroutines")
	assert.Contains(t, body, "num_cpu")
}

func TestSystemHandler_GetResources_RAMPercentIsNumeric(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/resources", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	_, ok := body["ram_percent"].(float64)
	assert.True(t, ok, "ram_percent should be numeric")
}

func TestSystemHandler_GetResources_CPUPercentInRange(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/resources", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	cpu, ok := body["cpu_percent"].(float64)
	assert.True(t, ok, "cpu_percent should be numeric")
	assert.GreaterOrEqual(t, cpu, float64(0))
	assert.LessOrEqual(t, cpu, float64(100))
}

func TestSystemHandler_GetResources_GoroutinesPositive(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/resources", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	gr, ok := body["goroutines"].(float64)
	assert.True(t, ok)
	assert.Greater(t, gr, float64(0))
}

func TestSystemHandler_GetResources_NumCPUPositive(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/resources", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	nc, ok := body["num_cpu"].(float64)
	assert.True(t, ok)
	assert.Greater(t, nc, float64(0))
}

// --- GetConnections ---

func TestSystemHandler_GetConnections_ReturnsOK(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/connections", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSystemHandler_GetConnections_ReturnsCount(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/connections", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "count")
	_, ok := body["count"].(float64)
	assert.True(t, ok, "count should be numeric")
}

func TestSystemHandler_GetConnections_EmptyDB(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/system/connections", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(0), body["count"])
}

// --- EmergencyCleanup ---

func TestSystemHandler_EmergencyCleanup_ReturnsOK(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/system/emergency-cleanup", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSystemHandler_EmergencyCleanup_ContainsMessage(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/system/emergency-cleanup", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "Emergency cleanup completed", body["message"])
	assert.Contains(t, body, "freed_bytes")
	assert.Contains(t, body, "alloc_before")
	assert.Contains(t, body, "alloc_after")
}

func TestSystemHandler_EmergencyCleanup_FreedBytesNonNegative(t *testing.T) {
	app, _ := setupSystemApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/system/emergency-cleanup", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	freed, ok := body["freed_bytes"].(float64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, freed, float64(0))
}

// --- Health Check ---

func TestSystemHandler_Health_Returns200(t *testing.T) {
	app := setupSystemHealthApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSystemHandler_Health_ResponseFormat(t *testing.T) {
	app := setupSystemHealthApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "healthy", body["status"])
	assert.Contains(t, body, "version")
	assert.Contains(t, body, "uptime")
	assert.Contains(t, body, "database")
	assert.Contains(t, body, "timestamp")
}

func TestSystemHandler_Health_VersionPopulated(t *testing.T) {
	app := setupSystemHealthApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEmpty(t, body["version"])
}

func TestSystemHandler_Health_DatabaseConnected(t *testing.T) {
	app := setupSystemHealthApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "connected", body["database"])
}

// --- WS Ticket ---

func TestSystemHandler_WSTicket_ReturnsTicket(t *testing.T) {
	app := setupWSTicketApp(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body, "ticket")
	ticket, ok := body["ticket"].(string)
	assert.True(t, ok)
	assert.Len(t, ticket, 32)
}

func TestSystemHandler_WSTicket_TicketIsHex(t *testing.T) {
	app := setupWSTicketApp(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	ticket, ok := body["ticket"].(string)
	assert.True(t, ok)
	for _, c := range ticket {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"ticket should be hex, got char: %c", c)
	}
}

func TestSystemHandler_WSTicket_UniquePerRequest(t *testing.T) {
	app := setupWSTicketApp(t)

	req1 := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	var body1 map[string]interface{}
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&body1))

	req2 := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	var body2 map[string]interface{}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body2))

	assert.NotEqual(t, body1["ticket"], body2["ticket"],
		"each request should get a unique ticket")
}
