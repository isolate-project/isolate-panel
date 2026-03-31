package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/services"
	"gorm.io/gorm"
)

func TestAuthHandler(t *testing.T) {
	t.Run("creates auth handler", func(t *testing.T) {
		handler := api.NewAuthHandler(nil, nil, nil)
		assert.NotNil(t, handler)
	})
}

func TestUsersHandler(t *testing.T) {
	t.Run("creates users handler", func(t *testing.T) {
		db := setupTestDB(t)
		notificationService := services.NewNotificationService(nil, "", "", "", "")
		userService := services.NewUserService(db, notificationService)
		handler := api.NewUsersHandler(userService)
		assert.NotNil(t, handler)
	})
}

func TestInboundsHandler(t *testing.T) {
	t.Run("creates inbounds handler", func(t *testing.T) {
		db := setupTestDB(t)
		service := services.NewInboundService(db, nil)
		handler := api.NewInboundsHandler(service)
		assert.NotNil(t, handler)
	})
}

func TestCoresHandler(t *testing.T) {
	t.Run("creates cores handler", func(t *testing.T) {
		handler := api.NewCoresHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestSettingsHandler(t *testing.T) {
	t.Run("creates settings handler", func(t *testing.T) {
		db := setupTestDB(t)
		settingsService := services.NewSettingsService(db)
		handler := api.NewSettingsHandler(settingsService, nil)
		assert.NotNil(t, handler)
	})
}

func TestNotificationHandler(t *testing.T) {
	t.Run("creates notification handler", func(t *testing.T) {
		notificationService := services.NewNotificationService(nil, "", "", "", "")
		handler := api.NewNotificationHandler(notificationService)
		assert.NotNil(t, handler)
	})
}

func TestStatsHandler(t *testing.T) {
	t.Run("creates stats handler", func(t *testing.T) {
		db := setupTestDB(t)
		handler := api.NewStatsHandler(db, nil, nil)
		assert.NotNil(t, handler)
	})
}

func TestWarpHandler(t *testing.T) {
	t.Run("creates warp handler", func(t *testing.T) {
		db := setupTestDB(t)
		warpService := services.NewWARPService(db, "/tmp")
		geoService := services.NewGeoService(db, "/tmp")
		handler := api.NewWarpHandler(warpService, geoService)
		assert.NotNil(t, handler)
	})
}

func setupTestDB(t *testing.T) *gorm.DB {
	return nil
}
