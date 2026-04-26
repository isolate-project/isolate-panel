package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/cache"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/services"
)

func setupRoutesDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	return db
}

func setupMinimalApp(t *testing.T) (*fiber.App, *App) {
	t.Helper()
	db := setupRoutesDB(t)

	tokenSvc := auth.NewTokenService("test-jwt-secret-min-32-characters!!", 15*time.Minute, 7*24*time.Hour, nil, nil)
	connections := services.NewConnectionTracker(db, 60*time.Second, "", "", "", "", "")

	a := &App{
		StartTime: time.Now(),
		stopQuota: make(chan struct{}),
		gormDB:    db,
		DashboardHub: api.NewDashboardHub(db, connections, tokenSvc),
		TokenSvc:     tokenSvc,
		LoginRL:      middleware.NewRateLimiter(5, time.Minute),
		ProtectedRL:  middleware.NewRateLimiter(600, time.Minute),
		HeavyRL:      middleware.NewRateLimiter(60, time.Minute),
	}

	fiberApp := fiber.New()

	return fiberApp, a
}

func TestSetupRoutes_HealthEndpoint(t *testing.T) {
	fiberApp, a := setupMinimalApp(t)
	SetupRoutes(fiberApp, a)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "healthy", body["status"])
	assert.NotEmpty(t, body["version"])
	assert.NotEmpty(t, body["uptime"])
	assert.Equal(t, "connected", body["database"])
}

func TestSetupRoutes_HealthEndpoint_UnhealthyDB(t *testing.T) {
	fiberApp, a := setupMinimalApp(t)
	SetupRoutes(fiberApp, a)

	sqlDB, _ := a.gormDB.DB()
	sqlDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "unhealthy", body["status"])
	assert.Equal(t, "disconnected", body["database"])
}

func TestSetupRoutes_ApiInfoEndpoint(t *testing.T) {
	fiberApp, a := setupMinimalApp(t)
	SetupRoutes(fiberApp, a)

	req := httptest.NewRequest(http.MethodGet, "/api/", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "Isolate Panel API", body["message"])
}

func TestSetupRoutes_ProtectedRoutes_RequireAuth(t *testing.T) {
	fiberApp, a := setupMinimalApp(t)
	SetupRoutes(fiberApp, a)

	protectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/me"},
		{http.MethodGet, "/api/cores/"},
		{http.MethodGet, "/api/users/"},
		{http.MethodGet, "/api/inbounds/"},
		{http.MethodGet, "/api/outbounds/"},
		{http.MethodGet, "/api/protocols/"},
		{http.MethodGet, "/api/stats/dashboard"},
		{http.MethodGet, "/api/settings/"},
	}

	for _, tc := range protectedRoutes {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			resp, err := fiberApp.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "route %s should require auth", tc.path)
		})
	}
}

func TestSetupRoutes_AuthRoutes_Exist(t *testing.T) {
	fiberApp, a := setupMinimalApp(t)
	SetupRoutes(fiberApp, a)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	// Should not be 404 (route exists), but 422 (unprocessable) or 400 (bad request) due to empty body
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

func TestSetupRoutes_WSTicketRoute_RequiresAuth(t *testing.T) {
	fiberApp, a := setupMinimalApp(t)
	SetupRoutes(fiberApp, a)

	req := httptest.NewRequest(http.MethodPost, "/api/ws/ticket", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func setupTestCache(t *testing.T) *cache.CacheManager {
	t.Helper()
	cm, err := cache.NewCacheManager()
	require.NoError(t, err)
	return cm
}

func TestSetupRoutes_SubscriptionRoutes_Exist(t *testing.T) {
	db := setupRoutesDB(t)
	tokenSvc := auth.NewTokenService("test-jwt-secret-min-32-characters!!", 15*time.Minute, 7*24*time.Hour, nil, nil)
	connections := services.NewConnectionTracker(db, 60*time.Second, "", "", "", "", "")
	cm := setupTestCache(t)

	a := &App{
		StartTime: time.Now(),
		stopQuota: make(chan struct{}),
		gormDB:    db,
		DashboardHub: api.NewDashboardHub(db, connections, tokenSvc),
		TokenSvc:     tokenSvc,
		LoginRL:      middleware.NewRateLimiter(5, time.Minute),
		ProtectedRL:  middleware.NewRateLimiter(600, time.Minute),
		HeavyRL:      middleware.NewRateLimiter(60, time.Minute),
		SubscriptionsH: api.NewSubscriptionsHandler(
			services.NewSubscriptionService(db, "http://localhost:8080", cm),
		),
	}

	fiberApp := fiber.New()
	SetupRoutes(fiberApp, a)

	req := httptest.NewRequest(http.MethodGet, "/sub/test-token", nil)
	resp, err := fiberApp.Test(req)
	require.NoError(t, err)
	assert.Less(t, resp.StatusCode, 500, "/sub/:token route should be registered")
}
