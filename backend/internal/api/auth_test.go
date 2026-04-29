package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestEnv(t *testing.T) (*fiber.App, *gorm.DB) {
	// Setup in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	err = db.AutoMigrate(&models.Admin{}, &models.LoginAttempt{}, &models.RefreshToken{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create test admin
	hash, _ := auth.HashPassword("password123")
	
	admin := models.Admin{
		Username:     "testadmin",
		PasswordHash: hash,
		IsActive:     true,
		IsSuperAdmin: true,
	}
	db.Create(&admin)

	// Setup token service
	tokenService , _ := auth.NewTokenService("this-is-a-very-long-test-secret-that-exceeds-the-minimum-64-byte-requirement-for-jwt-hs256", 900*time.Second, 7*24*time.Hour, nil, nil)

	// Setup handler
	handler := NewAuthHandler(db, tokenService, nil, nil, nil)

	// Setup fiber app
	app := fiber.New()
	app.Post("/login", handler.Login)

	return app, db
}

func TestAuthHandler_Login_Success(t *testing.T) {
	app, _ := setupAuthTestEnv(t)

	// Create login request
	reqBody, _ := json.Marshal(LoginRequest{
		Username: "testadmin",
		Password: "password123",
	})
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Decode response
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if loginResp.AccessToken == "" {
		t.Error("expected access token to be set")
	}
	if loginResp.Admin.Username != "testadmin" {
		t.Errorf("expected username testadmin, got %s", loginResp.Admin.Username)
	}
}

func TestAuthHandler_Login_Failure(t *testing.T) {
	app, _ := setupAuthTestEnv(t)

	// Create invalid login request
	reqBody, _ := json.Marshal(LoginRequest{
		Username: "testadmin",
		Password: "wrongpassword",
	})
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}
