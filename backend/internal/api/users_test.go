package api

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testEncryptionKeyInit sync.Once

func initTestEncryptionKey() {
	testEncryptionKeyInit.Do(func() {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			panic("failed to generate test encryption key: " + err.Error())
		}
		auth.SetTestEncryptionKey(key)
	})
}

// setupUsersTestDB sets up an in-memory DB with required migrations
func setupUsersTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Inbound{},
		&models.Admin{},
		&models.UserInboundMapping{},
	))
	return db
}

func setupUsersApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	initTestEncryptionKey()
	db := setupUsersTestDB(t)
	svc := services.NewUserService(db, nil)
	handler := NewUsersHandler(svc)

	app := fiber.New()
	// Inject fake admin_id into locals for authenticated routes
	app.Use(func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return c.Next()
	})
	app.Get("/users", handler.ListUsers)
	app.Post("/users", handler.CreateUser)
	app.Get("/users/:id", handler.GetUser)
	app.Put("/users/:id", handler.UpdateUser)
	app.Delete("/users/:id", handler.DeleteUser)
	app.Post("/users/:id/regenerate", handler.RegenerateCredentials)
	app.Get("/users/:id/inbounds", handler.GetUserInbounds)
	return app, db
}

func createTestUser(t *testing.T, app *fiber.App) uint {
	t.Helper()
	testName := strings.ReplaceAll(t.Name(), "TestUsersHandler_", "")
	body, _ := json.Marshal(services.CreateUserRequest{
		Username: "user_" + testName,
		Password: "pass12345678",
		Email:    "test@example.com",
	})
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		ID uint `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result.ID
}

// --- ListUsers ---

func TestUsersHandler_ListUsers_Empty(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, true, result["success"])
}

func TestUsersHandler_ListUsers_WithPagination(t *testing.T) {
	app, _ := setupUsersApp(t)
	createTestUser(t, app)

	req, _ := http.NewRequest(http.MethodGet, "/users?page=1&page_size=10", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- CreateUser ---

func TestUsersHandler_CreateUser_Success(t *testing.T) {
	app, _ := setupUsersApp(t)

	body, _ := json.Marshal(services.CreateUserRequest{
		Username: "newuser",
		Password: "secure123456",
		Email:    "new@example.com",
	})
	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "newuser", result["username"])
}

func TestUsersHandler_CreateUser_InvalidBody(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- GetUser ---

func TestUsersHandler_GetUser_Success(t *testing.T) {
	app, _ := setupUsersApp(t)
	id := createTestUser(t, app)

	req, _ := http.NewRequest(http.MethodGet, "/users/"+uint2str(id), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUsersHandler_GetUser_NotFound(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/users/99999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUsersHandler_GetUser_InvalidID(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/users/abc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- UpdateUser ---

func TestUsersHandler_UpdateUser_Success(t *testing.T) {
	app, _ := setupUsersApp(t)
	id := createTestUser(t, app)

	email := "updated@example.com"
	body, _ := json.Marshal(services.UpdateUserRequest{Email: &email})
	req, _ := http.NewRequest(http.MethodPut, "/users/"+uint2str(id), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUsersHandler_UpdateUser_InvalidID(t *testing.T) {
	app, _ := setupUsersApp(t)

	body, _ := json.Marshal(services.UpdateUserRequest{})
	req, _ := http.NewRequest(http.MethodPut, "/users/bad-id", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- DeleteUser ---

func TestUsersHandler_DeleteUser_Success(t *testing.T) {
	app, _ := setupUsersApp(t)
	id := createTestUser(t, app)

	req, _ := http.NewRequest(http.MethodDelete, "/users/"+uint2str(id), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUsersHandler_DeleteUser_InvalidID(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodDelete, "/users/nan", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- RegenerateCredentials ---

func TestUsersHandler_RegenerateCredentials_Success(t *testing.T) {
	app, _ := setupUsersApp(t)
	id := createTestUser(t, app)

	req, _ := http.NewRequest(http.MethodPost, "/users/"+uint2str(id)+"/regenerate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUsersHandler_RegenerateCredentials_InvalidID(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodPost, "/users/xyz/regenerate", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- GetUserInbounds ---

func TestUsersHandler_GetUserInbounds_Empty(t *testing.T) {
	app, _ := setupUsersApp(t)
	id := createTestUser(t, app)

	req, _ := http.NewRequest(http.MethodGet, "/users/"+uint2str(id)+"/inbounds", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUsersHandler_GetUserInbounds_InvalidID(t *testing.T) {
	app, _ := setupUsersApp(t)

	req, _ := http.NewRequest(http.MethodGet, "/users/notanid/inbounds", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- formatUserResponse ---

func TestUsersHandler_FormatUserResponse_WithExpiry(t *testing.T) {
	db := setupUsersTestDB(t)
	svc := services.NewUserService(db, nil)
	handler := NewUsersHandler(svc)

	expiry := time.Now().Add(30 * 24 * time.Hour)
	tok := "tok"
	user := &models.User{
		Username:          "fmt_user",
		UUID:              "test-uuid",
		Password:          "pass",
		Token:             &tok,
		SubscriptionToken: "sub",
		IsActive:          true,
		ExpiryDate:        &expiry,
	}
	resp := handler.formatUserResponse(user)
	assert.NotNil(t, resp.ExpiryDate)
	assert.Equal(t, "fmt_user", resp.Username)
}

func TestUsersHandler_FormatUserResponse_NoExpiry(t *testing.T) {
	db := setupUsersTestDB(t)
	svc := services.NewUserService(db, nil)
	handler := NewUsersHandler(svc)

	user := &models.User{Username: "no_expiry", ExpiryDate: nil}
	resp := handler.formatUserResponse(user)
	assert.Nil(t, resp.ExpiryDate)
}

// Helper - convert uint to string without strconv (avoid import cycle)
func uint2str(id uint) string {
	if id == 0 {
		return "0"
	}
	var s string
	for n := id; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}
