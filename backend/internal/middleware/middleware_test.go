package middleware_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/middleware"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Create token service
	secret := "this-is-a-very-long-test-secret-that-exceeds-the-minimum-64-byte-requirement-for-jwt-hs256"
	tokenService , _ := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, nil, nil)

	// Generate valid token
	token, err := tokenService.GenerateAccessToken(1, "testuser", false, false)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create Fiber app with middleware
	app := fiber.New()
	app.Use(middleware.AuthMiddleware(tokenService, nil))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	// Create request with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Test request
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	secret := "this-is-a-very-long-test-secret-that-exceeds-the-minimum-64-byte-requirement-for-jwt-hs256"
	tokenService , _ := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, nil, nil)

	app := fiber.New()
	app.Use(middleware.AuthMiddleware(tokenService, nil))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	// Request without Authorization header
	req := httptest.NewRequest("GET", "/test", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	secret := "this-is-a-very-long-test-secret-that-exceeds-the-minimum-64-byte-requirement-for-jwt-hs256"
	tokenService , _ := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, nil, nil)

	app := fiber.New()
	app.Use(middleware.AuthMiddleware(tokenService, nil))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	// Request with invalid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 401 {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	secret := "this-is-a-very-long-test-secret-that-exceeds-the-minimum-64-byte-requirement-for-jwt-hs256"
	tokenService , _ := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, nil, nil)

	app := fiber.New()
	app.Use(middleware.AuthMiddleware(tokenService, nil))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing Bearer prefix",
			header: "token-without-bearer",
		},
		{
			name:   "wrong prefix",
			header: "Basic token",
		},
		{
			name:   "empty after Bearer",
			header: "Bearer ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != 401 {
				t.Errorf("Expected status 401, got %d", resp.StatusCode)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := middleware.NewRateLimiter(3, 1*time.Second)

	app := fiber.New()
	app.Post("/login", middleware.LoginRateLimiter(limiter), func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/login", nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("POST", "/login", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429 (rate limited), got %d", resp.StatusCode)
	}
}

func TestRateLimiter_IgnoresXForwardedFor(t *testing.T) {
	// Verify that rate limiter uses c.IP() not X-Forwarded-For header,
	// preventing rate limit bypass via header spoofing.
	limiter := middleware.NewRateLimiter(2, 1*time.Second)

	app := fiber.New()
	app.Post("/login", middleware.LoginRateLimiter(limiter), func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	// Use up the limit with 2 requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 3rd request with spoofed X-Forwarded-For should still be rate limited
	req := httptest.NewRequest("POST", "/login", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.99")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 429 {
		t.Errorf("Spoofed XFF should NOT bypass rate limit: expected 429, got %d", resp.StatusCode)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := middleware.NewRateLimiter(2, 100*time.Millisecond)

	app := fiber.New()
	app.Post("/login", middleware.LoginRateLimiter(limiter), func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		app.Test(req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("POST", "/login", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should succeed again
	req = httptest.NewRequest("POST", "/login", nil)
	resp, _ = app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 after reset, got %d", resp.StatusCode)
	}
}
