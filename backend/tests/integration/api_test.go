package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vovk4morkovk4/isolate-panel/internal/api"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

// setupTestApp creates a test Fiber app with all handlers
func setupTestApp(t *testing.T) *fiber.App {
	t.Helper()

	db := testutil.SetupTestDB(t)
	testutil.SeedTestData(t, db)

	// Create services
	notificationService := services.NewNotificationService(nil, "", "", "", "")
	userService := services.NewUserService(db, notificationService)
	settingsService := services.NewSettingsService(db)

	// Create handlers
	usersHandler := api.NewUsersHandler(userService)
	settingsHandler := api.NewSettingsHandler(settingsService, nil)

	// Create app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// Setup routes
	apiGroup := app.Group("/api")
	usersGroup := apiGroup.Group("/users")
	usersGroup.Get("/", usersHandler.ListUsers)
	usersGroup.Post("/", usersHandler.CreateUser)
	usersGroup.Get("/:id", usersHandler.GetUser)
	usersGroup.Put("/:id", usersHandler.UpdateUser)
	usersGroup.Delete("/:id", usersHandler.DeleteUser)

	settingsGroup := apiGroup.Group("/settings")
	settingsGroup.Get("/monitoring", settingsHandler.GetMonitoring)
	settingsGroup.Put("/monitoring", settingsHandler.UpdateMonitoring)

	return app
}

func TestAPI_UsersList(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
}

func TestAPI_UsersGet(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "testuser1", response["username"])
}

func TestAPI_UsersGetNotFound(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/999", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAPI_SettingsGetMonitoring(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/monitoring", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "lite", response["mode"])
}

func TestAPI_SettingsUpdateMonitoring(t *testing.T) {
	app := setupTestApp(t)

	reqBody := map[string]string{"mode": "full"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/monitoring", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "full", response["mode"])
}

func TestAPI_SettingsUpdateMonitoringInvalid(t *testing.T) {
	app := setupTestApp(t)

	reqBody := map[string]string{"mode": "invalid"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/monitoring", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
