package api

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupCoresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

func setupCoresApp(t *testing.T) (*fiber.App, *cores.CoreManager) {
	t.Helper()
	db := setupCoresTestDB(t)

	coreManager := cores.NewCoreManager(db, "http://127.0.0.1:9001", nil)
	handler := NewCoresHandler(coreManager)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return c.Next()
	})

	coresGroup := app.Group("/cores")
	coresGroup.Get("/", handler.ListCores)
	coresGroup.Get("/:name", handler.GetCore)
	coresGroup.Get("/:name/status", handler.GetCoreStatus)
	coresGroup.Post("/:name/start", handler.StartCore)
	coresGroup.Post("/:name/stop", handler.StopCore)
	coresGroup.Post("/:name/restart", handler.RestartCore)

	return app, coreManager
}

func TestCoresHandler_ListCores(t *testing.T) {
	app, _ := setupCoresApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/cores", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Supervisor URL is unreachable in tests, so handler returns 500
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCoresHandler_GetCore(t *testing.T) {
	app, _ := setupCoresApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/cores/xray", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	// Core "xray" doesn't exist in test DB
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCoresHandler_StartCore(t *testing.T) {
	app, _ := setupCoresApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/cores/xray/start", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCoresHandler_StopCore(t *testing.T) {
	app, _ := setupCoresApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/cores/xray/stop", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCoresHandler_RestartCore(t *testing.T) {
	app, _ := setupCoresApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/cores/xray/restart", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
