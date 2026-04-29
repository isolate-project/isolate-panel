package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/eventbus"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
)

func init() {
	logger.Init(&logger.Config{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	})
}

func setupAuthTestApp(t *testing.T) (*fiber.App, *gorm.DB, *auth.TokenService) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	passwordHash, err := auth.HashPassword("testpassword123")
	require.NoError(t, err)

	admin := &models.Admin{
		Username:     "testadmin",
		PasswordHash: passwordHash,
		IsSuperAdmin: true,
		IsActive:     true,
	}
	require.NoError(t, db.Create(admin).Error)

	secret := "this-is-a-very-long-secret-key-that-is-at-least-64-characters-long-for-hs256-security"
	require.True(t, len(secret) >= 64)

	validator := func(adminID uint) (bool, bool, bool, error) {
		var a models.Admin
		if err := db.First(&a, adminID).Error; err != nil {
			return false, false, false, err
		}
		return a.IsActive, a.IsSuperAdmin, a.MustChangePassword, nil
	}

	tokenService, err := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, validator, db)
	require.NoError(t, err)

	sessionManager := auth.NewBFFSessionManager(7 * 24 * time.Hour)
	authHandler := api.NewAuthHandler(db, tokenService, sessionManager, nil, nil)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})

	authGroup := app.Group("/auth")
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh", authHandler.Refresh)
	authGroup.Post("/logout", authHandler.Logout)

	protected := app.Group("/api/protected")
	protected.Use(middleware.AuthMiddleware(tokenService, sessionManager))
	protected.Get("/me", authHandler.Me)

	return app, db, tokenService
}

func TestAuthFlow_EndToEnd(t *testing.T) {
	app, _, tokenService := setupAuthTestApp(t)
	defer tokenService.Stop()

	var accessToken, refreshToken string

	t.Run("Login with correct credentials returns tokens", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testadmin",
			"password": "testpassword123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["access_token"])
		assert.NotEmpty(t, response["refresh_token"])

		accessToken = response["access_token"].(string)
		refreshToken = response["refresh_token"].(string)
	})

	t.Run("Access protected route with valid access_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/protected/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "testadmin", response["username"])
	})

	t.Run("Refresh token returns new access_token", func(t *testing.T) {
		reqBody := map[string]string{
			"refresh_token": refreshToken,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["access_token"])
		assert.NotEmpty(t, response["refresh_token"])

		accessToken = response["access_token"].(string)
		refreshToken = response["refresh_token"].(string)
	})

	t.Run("Access with blacklisted access_token returns 401", func(t *testing.T) {
		tokenService.BlacklistAccessToken(accessToken)

		req := httptest.NewRequest(http.MethodGet, "/api/protected/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Logout revokes refresh token", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testadmin",
			"password": "testpassword123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		var loginResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&loginResponse)
		require.NoError(t, err)

		freshRefreshToken := loginResponse["refresh_token"].(string)
		freshAccessToken := loginResponse["access_token"].(string)

		logoutBody := map[string]string{
			"refresh_token": freshRefreshToken,
		}
		logoutBodyJSON, _ := json.Marshal(logoutBody)

		logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(logoutBodyJSON))
		logoutReq.Header.Set("Content-Type", "application/json")
		logoutReq.Header.Set("Authorization", "Bearer "+freshAccessToken)
		logoutResp, err := app.Test(logoutReq)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, logoutResp.StatusCode)

		refreshBody := map[string]string{
			"refresh_token": freshRefreshToken,
		}
		refreshBodyJSON, _ := json.Marshal(refreshBody)

		refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(refreshBodyJSON))
		refreshReq.Header.Set("Content-Type", "application/json")
		refreshResp, err := app.Test(refreshReq)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, refreshResp.StatusCode)
	})
}

func setupRateLimitTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	passwordHash, err := auth.HashPassword("testpassword123")
	require.NoError(t, err)

	admin := &models.Admin{
		Username:     "testadmin",
		PasswordHash: passwordHash,
		IsSuperAdmin: true,
		IsActive:     true,
	}
	require.NoError(t, db.Create(admin).Error)

	secret := "this-is-a-very-long-secret-key-that-is-at-least-64-characters-long-for-hs256-security"
	validator := func(adminID uint) (bool, bool, bool, error) {
		var a models.Admin
		if err := db.First(&a, adminID).Error; err != nil {
			return false, false, false, err
		}
		return a.IsActive, a.IsSuperAdmin, a.MustChangePassword, nil
	}

	tokenService, err := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, validator, db)
	require.NoError(t, err)
	t.Cleanup(func() { tokenService.Stop() })

	sessionManager := auth.NewBFFSessionManager(7 * 24 * time.Hour)
	authHandler := api.NewAuthHandler(db, tokenService, sessionManager, nil, nil)

	loginLimiter := middleware.NewRateLimiter(5, time.Minute)

	app := fiber.New()
	app.Post("/auth/login", middleware.LoginRateLimiter(loginLimiter), authHandler.Login)

	return app, db
}

func TestRateLimit_LoginAfter5Attempts(t *testing.T) {
	app, _ := setupRateLimitTestApp(t)

	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("Failed attempt %d", i+1), func(t *testing.T) {
			reqBody := map[string]string{
				"username": "testadmin",
				"password": "wrongpassword",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}

	t.Run("6th attempt returns 429 Too Many Requests", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testadmin",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	})
}

func setupSubscriptionSignedURLApp(t *testing.T) (*fiber.App, *gorm.DB, *auth.SubscriptionSigner, *models.User) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	trafficLimit := int64(107374182400)
	user := &models.User{
		UUID:              "test-uuid-subscription",
		Username:          "testuser_subscription",
		Email:             "testuser@example.com",
		SubscriptionToken: "test-subscription-token-000000000000000000000000000000",
		IsActive:          true,
		TrafficLimitBytes: &trafficLimit,
		TrafficUsedBytes:  0,
	}
	require.NoError(t, db.Create(user).Error)

	core := &models.Core{
		Name:      "xray",
		Version:   "26.3.27",
		IsEnabled: true,
	}
	require.NoError(t, db.Create(core).Error)

	inbound := &models.Inbound{
		Name:          "Test Inbound",
		Protocol:      "vless",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          12345,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(inbound).Error)

	mapping := &models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	}
	require.NoError(t, db.Create(mapping).Error)

	secret := "subscription-signing-secret-key-for-hmac-sha256-signature-verification"
	signer := auth.NewSubscriptionSigner(secret)

	subscriptionService := services.NewSubscriptionService(db, "http://localhost:8080")
	subscriptionsHandler := api.NewSubscriptionsHandler(subscriptionService)

	app := fiber.New()
	app.Get("/sub/:token", func(c fiber.Ctx) error {
		token := c.Params("token")
		sig := c.Query("sig")
		expStr := c.Query("exp")

		if sig == "" || expStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing signature"})
		}

		exp, err := strconv.ParseInt(expStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid expiration"})
		}

		if !signer.Verify(token, sig, exp) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired signature"})
		}

		return subscriptionsHandler.GetAutoDetectSubscription(c)
	})

	return app, db, signer, user
}

func TestSubscriptionSignedURL_ValidAndInvalid(t *testing.T) {
	app, _, signer, user := setupSubscriptionSignedURLApp(t)

	t.Run("Valid signed URL returns 200", func(t *testing.T) {
		sig, exp := signer.Sign(user.SubscriptionToken, time.Hour)
		url := fmt.Sprintf("/sub/%s?sig=%s&exp=%d", user.SubscriptionToken, sig, exp)

		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("User-Agent", "v2rayNG/1.8.5")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Tampered signature returns 401", func(t *testing.T) {
		_, exp := signer.Sign(user.SubscriptionToken, time.Hour)
		tamperedSig := "tamperedsignature123456789abcdef"
		url := fmt.Sprintf("/sub/%s?sig=%s&exp=%d", user.SubscriptionToken, tamperedSig, exp)

		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("User-Agent", "v2rayNG/1.8.5")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Expired signature returns 401", func(t *testing.T) {
		sig, exp := signer.Sign(user.SubscriptionToken, 1*time.Second)
		url := fmt.Sprintf("/sub/%s?sig=%s&exp=%d", user.SubscriptionToken, sig, exp)

		time.Sleep(2 * time.Second)

		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("User-Agent", "v2rayNG/1.8.5")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Random signature returns 401", func(t *testing.T) {
		randomSig := "72616e646f6d7369676e6174757265"
		exp := time.Now().Add(time.Hour).Unix()
		url := fmt.Sprintf("/sub/%s?sig=%s&exp=%d", user.SubscriptionToken, randomSig, exp)

		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("User-Agent", "v2rayNG/1.8.5")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestEventBus_PublishSubscribe(t *testing.T) {
	t.Run("Subscribe and receive events", func(t *testing.T) {
		bus := eventbus.NewEventBus[eventbus.UserCreatedEvent](logger.Log)
		defer bus.Close()

		received := make(chan eventbus.UserCreatedEvent, 1)
		handler := func(event eventbus.UserCreatedEvent) error {
			received <- event
			return nil
		}

		subID := bus.Subscribe(handler)
		assert.NotZero(t, subID)
		assert.Equal(t, 1, bus.SubscriberCount())

		testEvent := eventbus.UserCreatedEvent{
			User: models.User{
				ID:       1,
				Username: "testuser",
				UUID:     "test-uuid-123",
			},
			CreatedBy: 1,
			Timestamp: time.Now(),
		}
		bus.Publish(testEvent)

		select {
		case receivedEvent := <-received:
			assert.Equal(t, uint(1), receivedEvent.User.ID)
			assert.Equal(t, "testuser", receivedEvent.User.Username)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("Multiple subscribers receive events", func(t *testing.T) {
		bus := eventbus.NewEventBus[eventbus.CoreStartedEvent](logger.Log)
		defer bus.Close()

		received1 := make(chan eventbus.CoreStartedEvent, 1)
		received2 := make(chan eventbus.CoreStartedEvent, 1)

		bus.Subscribe(func(event eventbus.CoreStartedEvent) error {
			received1 <- event
			return nil
		})

		bus.Subscribe(func(event eventbus.CoreStartedEvent) error {
			received2 <- event
			return nil
		})

		assert.Equal(t, 2, bus.SubscriberCount())

		testEvent := eventbus.CoreStartedEvent{
			CoreID:    1,
			CoreName:  "xray",
			PID:       12345,
			Timestamp: time.Now(),
		}
		bus.Publish(testEvent)

		select {
		case event1 := <-received1:
			assert.Equal(t, "xray", event1.CoreName)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for subscriber 1")
		}

		select {
		case event2 := <-received2:
			assert.Equal(t, "xray", event2.CoreName)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for subscriber 2")
		}
	})

	t.Run("Different event types don't interfere", func(t *testing.T) {
		userBus := eventbus.NewEventBus[eventbus.UserCreatedEvent](logger.Log)
		coreBus := eventbus.NewEventBus[eventbus.CoreStartedEvent](logger.Log)
		defer userBus.Close()
		defer coreBus.Close()

		userReceived := make(chan eventbus.UserCreatedEvent, 1)
		coreReceived := make(chan eventbus.CoreStartedEvent, 1)

		userBus.Subscribe(func(event eventbus.UserCreatedEvent) error {
			userReceived <- event
			return nil
		})

		coreBus.Subscribe(func(event eventbus.CoreStartedEvent) error {
			coreReceived <- event
			return nil
		})

		userBus.Publish(eventbus.UserCreatedEvent{User: models.User{ID: 1, Username: "user1"}})
		coreBus.Publish(eventbus.CoreStartedEvent{CoreID: 1, CoreName: "singbox"})

		select {
		case userEvent := <-userReceived:
			assert.Equal(t, "user1", userEvent.User.Username)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for user event")
		}

		select {
		case coreEvent := <-coreReceived:
			assert.Equal(t, "singbox", coreEvent.CoreName)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for core event")
		}
	})

	t.Run("Async publish doesn't block", func(t *testing.T) {
		bus := eventbus.NewEventBus[string](logger.Log)
		defer bus.Close()

		bus.Subscribe(func(event string) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		start := time.Now()
		bus.Publish("test")
		elapsed := time.Since(start)

		assert.Less(t, elapsed, 50*time.Millisecond)
	})

	t.Run("Panic recovery in subscriber doesn't crash bus", func(t *testing.T) {
		bus := eventbus.NewEventBus[int](logger.Log)
		defer bus.Close()

		received := make(chan int, 1)
		panicTriggered := make(chan bool, 1)

		bus.Subscribe(func(event int) error {
			if event == 42 {
				panicTriggered <- true
				panic("intentional panic")
			}
			return nil
		})

		bus.Subscribe(func(event int) error {
			if event == 100 {
				received <- event
			}
			return nil
		})

		bus.Publish(42)

		select {
		case <-panicTriggered:
			time.Sleep(50 * time.Millisecond)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for panic trigger")
		}

		bus.Publish(100)

		select {
		case val := <-received:
			assert.Equal(t, 100, val)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event after panic recovery")
		}

		assert.Equal(t, 2, bus.SubscriberCount())
	})

	t.Run("Unsubscribe removes handler", func(t *testing.T) {
		bus := eventbus.NewEventBus[string](logger.Log)
		defer bus.Close()

		handler := func(event string) error {
			return nil
		}

		subID := bus.Subscribe(handler)
		assert.Equal(t, 1, bus.SubscriberCount())

		ok := bus.Unsubscribe(subID)
		assert.True(t, ok)
		assert.Equal(t, 0, bus.SubscriberCount())

		ok = bus.Unsubscribe(subID)
		assert.False(t, ok)
	})

	t.Run("Registry creates all event buses", func(t *testing.T) {
		registry := eventbus.NewRegistry()
		defer registry.Close()

		assert.NotNil(t, registry.UserCreated)
		assert.NotNil(t, registry.UserUpdated)
		assert.NotNil(t, registry.UserDeleted)
		assert.NotNil(t, registry.CoreStarted)
		assert.NotNil(t, registry.CoreStopped)
		assert.NotNil(t, registry.CoreRestarted)
		assert.NotNil(t, registry.InboundCreated)
		assert.NotNil(t, registry.InboundDeleted)
		assert.NotNil(t, registry.BackupCreated)
		assert.NotNil(t, registry.AdminLogin)
		assert.NotNil(t, registry.AdminAction)

		assert.Equal(t, 0, registry.UserCreated.SubscriberCount())
		assert.Equal(t, 0, registry.CoreStarted.SubscriberCount())
	})
}

func setupRBACTestApp(t *testing.T) (*fiber.App, *gorm.DB, *auth.TokenService) {
	t.Helper()

	db := testutil.SetupTestDB(t)

	passwordHash, err := auth.HashPassword("testpassword123")
	require.NoError(t, err)

	perms := auth.NewPermissions(auth.PermViewDashboard, auth.PermManageUsers)

	admin := &models.Admin{
		Username:     "limitedadmin",
		PasswordHash: passwordHash,
		IsSuperAdmin: false,
		Permissions:  uint64(perms),
		IsActive:     true,
	}
	require.NoError(t, db.Create(admin).Error)

	superAdmin := &models.Admin{
		Username:     "superadmin",
		PasswordHash: passwordHash,
		IsSuperAdmin: true,
		IsActive:     true,
	}
	require.NoError(t, db.Create(superAdmin).Error)

	secret := "this-is-a-very-long-secret-key-that-is-at-least-64-characters-long-for-hs256-security"
	validator := func(adminID uint) (bool, bool, bool, error) {
		var a models.Admin
		if err := db.First(&a, adminID).Error; err != nil {
			return false, false, false, err
		}
		return a.IsActive, a.IsSuperAdmin, a.MustChangePassword, nil
	}

	tokenService, err := auth.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, validator, db)
	require.NoError(t, err)

	sessionManager := auth.NewBFFSessionManager(7 * 24 * time.Hour)
	authHandler := api.NewAuthHandler(db, tokenService, sessionManager, nil, nil)

	app := fiber.New()

	app.Post("/auth/login", authHandler.Login)

	protected := app.Group("/api")
	protected.Use(middleware.AuthMiddleware(tokenService, sessionManager))

	protected.Get("/users",
		middleware.RequirePermission(auth.PermManageUsers),
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "users list"})
		},
	)

	protected.Get("/cores",
		middleware.RequirePermission(auth.PermManageCores),
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "cores list"})
		},
	)

	protected.Get("/dashboard",
		middleware.RequirePermission(auth.PermViewDashboard),
		func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "dashboard"})
		},
	)

	return app, db, tokenService
}

func TestRBAC_RequiresPermission(t *testing.T) {
	app, _, tokenService := setupRBACTestApp(t)
	defer tokenService.Stop()

	var limitedAccessToken string
	t.Run("Login as limited admin", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "limitedadmin",
			"password": "testpassword123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		limitedAccessToken = response["access_token"].(string)
	})

	var superAccessToken string
	t.Run("Login as super admin", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "superadmin",
			"password": "testpassword123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		superAccessToken = response["access_token"].(string)
	})

	t.Run("Access with missing permission returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cores", nil)
		req.Header.Set("Authorization", "Bearer "+limitedAccessToken)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "permission")
	})

	t.Run("Access with correct permission returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+limitedAccessToken)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "users list", response["message"])
	})

	t.Run("Super admin can access all routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cores", nil)
		req.Header.Set("Authorization", "Bearer "+superAccessToken)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Limited admin can access dashboard with ViewDashboard permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+limitedAccessToken)
		resp, err := app.Test(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
