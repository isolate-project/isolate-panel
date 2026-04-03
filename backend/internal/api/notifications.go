package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
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
//
// @Summary      List notifications
// @Description  Returns recent notification history
// @Tags         notifications
// @Produce      json
// @Param        limit   query  int  false  "Max results"  default(50)
// @Param        offset  query  int  false  "Skip N results"
// @Success      200     {object}  map[string]interface{}
// @Router       /notifications [get]
// @Security     BearerAuth
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
//
// @Summary      Get notification
// @Description  Returns a single notification event by ID
// @Tags         notifications
// @Produce      json
// @Param        id   path  int  true  "Notification ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /notifications/{id} [get]
// @Security     BearerAuth
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
//
// @Summary      Delete notification
// @Description  Delete a notification event from history
// @Tags         notifications
// @Produce      json
// @Param        id   path  int  true  "Notification ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /notifications/{id} [delete]
// @Security     BearerAuth
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
//
// @Summary      Get notification settings
// @Description  Returns current Telegram and Webhook notification configuration
// @Tags         notifications
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /notifications/settings [get]
// @Security     BearerAuth
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
//
// @Summary      Update notification settings
// @Description  Configure Telegram bot and Webhook URL, triggers, and enabled state
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        body  body  models.NotificationSettings  true  "Notification configuration"
// @Success      200   {object}  map[string]interface{}
// @Router       /notifications/settings [put]
// @Security     BearerAuth
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
//
// @Summary      Send test notification
// @Description  Send a test notification to configured channels to verify the integration
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]string  true  "{channel: webhook|telegram|all}"
// @Success      200   {object}  map[string]interface{}
// @Router       /notifications/test [post]
// @Security     BearerAuth
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
