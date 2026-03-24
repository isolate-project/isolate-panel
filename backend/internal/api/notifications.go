package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

// NotificationHandler handles notification API requests
type NotificationHandler struct {
	notificationService *services.NotificationService
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// RegisterRoutes registers notification API routes
func (h *NotificationHandler) RegisterRoutes(router fiber.Router) {
	notifications := router.Group("/notifications")

	// Notification operations
	notifications.Get("/", h.ListNotifications)
	notifications.Get("/:id", h.GetNotification)
	notifications.Delete("/:id", h.DeleteNotification)

	// Settings
	notifications.Get("/settings", h.GetSettings)
	notifications.Put("/settings", h.UpdateSettings)

	// Test
	notifications.Post("/test", h.SendTestNotification)
}

// ListNotifications returns list of notifications
func (h *NotificationHandler) ListNotifications(c fiber.Ctx) error {
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	if limit > 100 {
		limit = 100
	}

	notifications, err := h.notificationService.ListNotifications(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": notifications,
	})
}

// GetNotification returns a single notification
func (h *NotificationHandler) GetNotification(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification ID",
		})
	}

	notification, err := h.notificationService.GetNotification(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "notification not found",
		})
	}

	return c.JSON(fiber.Map{
		"data": notification,
	})
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid notification ID",
		})
	}

	if err := h.notificationService.DeleteNotification(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Notification deleted",
	})
}

// GetSettings returns notification settings
func (h *NotificationHandler) GetSettings(c fiber.Ctx) error {
	settings, err := h.notificationService.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": settings,
	})
}

// UpdateSettings updates notification settings
func (h *NotificationHandler) UpdateSettings(c fiber.Ctx) error {
	var settings models.NotificationSettings

	if err := c.Bind().JSON(&settings); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.notificationService.UpdateSettings(&settings); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Settings updated",
		"data":    settings,
	})
}

// SendTestNotification sends a test notification
func (h *NotificationHandler) SendTestNotification(c fiber.Ctx) error {
	var req struct {
		Channel string `json:"channel"` // "webhook", "telegram", "all"
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get current settings
	settings, err := h.notificationService.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	results := make([]string, 0)

	// Test webhook
	if req.Channel == "all" || req.Channel == "webhook" {
		if settings.WebhookEnabled {
			webhookNotifier := services.NewWebhookNotifier(settings.WebhookURL, settings.WebhookSecret)
			notification := &models.Notification{
				EventType: "test",
				Severity:  models.NotificationSeverityInfo,
				Title:     "Test Webhook Notification",
				Message:   "This is a test webhook notification from Isolate Panel",
			}
			if err := webhookNotifier.Send(notification); err != nil {
				results = append(results, "webhook: failed - "+err.Error())
			} else {
				results = append(results, "webhook: sent successfully")
			}
		} else {
			results = append(results, "webhook: not enabled")
		}
	}

	// Test Telegram
	if req.Channel == "all" || req.Channel == "telegram" {
		if settings.TelegramEnabled {
			telegramNotifier := services.NewTelegramNotifier(settings.TelegramBotToken, settings.TelegramChatID)
			notification := &models.Notification{
				EventType: "test",
				Severity:  models.NotificationSeverityInfo,
				Title:     "Test Telegram Notification",
				Message:   "This is a test Telegram notification from Isolate Panel",
			}
			if err := telegramNotifier.Send(notification); err != nil {
				results = append(results, "telegram: failed - "+err.Error())
			} else {
				results = append(results, "telegram: sent successfully")
			}
		} else {
			results = append(results, "telegram: not enabled")
		}
	}

	return c.JSON(fiber.Map{
		"message": "Test notification processed",
		"results": results,
	})
}
