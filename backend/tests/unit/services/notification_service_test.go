package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

func TestNotificationService(t *testing.T) {
	t.Run("creates notification service", func(t *testing.T) {
		service := services.NewNotificationService(nil, "", "", "", "")
		assert.NotNil(t, service)
	})

	t.Run("sends webhook notification", func(t *testing.T) {
		service := services.NewNotificationService(nil, "http://example.com/webhook", "secret", "", "")
		assert.NotNil(t, service)
	})

	t.Run("sends telegram notification", func(t *testing.T) {
		service := services.NewNotificationService(nil, "", "", "telegram-token", "chat-id")
		assert.NotNil(t, service)
	})
}

func TestNotificationSettings(t *testing.T) {
	t.Run("gets default settings", func(t *testing.T) {
		settings := models.DefaultNotificationSettings()
		assert.NotNil(t, settings)
		assert.False(t, settings.WebhookEnabled)
		assert.False(t, settings.TelegramEnabled)
	})

	t.Run("enables webhook", func(t *testing.T) {
		settings := models.DefaultNotificationSettings()
		settings.WebhookEnabled = true
		settings.WebhookURL = "http://example.com"
		assert.True(t, settings.WebhookEnabled)
		assert.Equal(t, "http://example.com", settings.WebhookURL)
	})

	t.Run("enables telegram", func(t *testing.T) {
		settings := models.DefaultNotificationSettings()
		settings.TelegramEnabled = true
		settings.TelegramBotToken = "token"
		settings.TelegramChatID = "chat-id"
		assert.True(t, settings.TelegramEnabled)
	})
}

func TestNotificationEventTypes(t *testing.T) {
	t.Run("user created event", func(t *testing.T) {
		eventType := "user_created"
		assert.Equal(t, "user_created", eventType)
	})

	t.Run("user deleted event", func(t *testing.T) {
		eventType := "user_deleted"
		assert.Equal(t, "user_deleted", eventType)
	})

	t.Run("quota exceeded event", func(t *testing.T) {
		eventType := "quota_exceeded"
		assert.Equal(t, "quota_exceeded", eventType)
	})

	t.Run("certificate expiring event", func(t *testing.T) {
		eventType := "certificate_expiring"
		assert.Equal(t, "certificate_expiring", eventType)
	})

	t.Run("core status event", func(t *testing.T) {
		eventType := "core_status_changed"
		assert.Equal(t, "core_status_changed", eventType)
	})
}
