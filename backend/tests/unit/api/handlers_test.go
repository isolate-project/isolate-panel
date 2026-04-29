package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Admin{},
		&models.User{},
		&models.Inbound{},
		&models.Setting{},
		&models.Notification{},
	)
	require.NoError(t, err)
	return db
}

func TestUsersHandler_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	notificationService := services.NewNotificationService(nil, "", "", "", "")
	userService := services.NewUserService(db, notificationService)
	handler := api.NewUsersHandler(userService)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return c.Next()
	})
	app.Get("/users", handler.ListUsers)

	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.NotNil(t, body["data"])
}

func TestSettingsHandler_GetMonitoring(t *testing.T) {
	db := setupTestDB(t)
	settingsService := services.NewSettingsService(db)
	handler := api.NewSettingsHandler(settingsService, nil)

	app := fiber.New()
	app.Get("/settings/monitoring", handler.GetMonitoring)

	req, _ := http.NewRequest(http.MethodGet, "/settings/monitoring", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.True(t, body["success"].(bool))
}

func TestNotificationHandler_ListNotifications(t *testing.T) {
	db := setupTestDB(t)
	notificationService := services.NewNotificationService(db, "", "", "", "")
	handler := api.NewNotificationHandler(notificationService)

	app := fiber.New()
	app.Get("/notifications", handler.ListNotifications)

	req, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.NotNil(t, body["data"])
}

func TestCoresHandler_Create(t *testing.T) {
	handler := api.NewCoresHandler(nil)
	assert.NotNil(t, handler)
}

func TestAuthHandler_Create(t *testing.T) {
	handler := api.NewAuthHandler(nil, nil, nil, nil, nil)
	assert.NotNil(t, handler)
}
