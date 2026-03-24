package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

type SubscriptionsHandler struct {
	subscriptionService *services.SubscriptionService
}

func NewSubscriptionsHandler(subscriptionService *services.SubscriptionService) *SubscriptionsHandler {
	return &SubscriptionsHandler{
		subscriptionService: subscriptionService,
	}
}

// GetV2RaySubscription serves V2Ray format subscription (base64-encoded links)
func (h *SubscriptionsHandler) GetV2RaySubscription(c fiber.Ctx) error {
	start := time.Now()
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	data, err := h.subscriptionService.GetUserSubscriptionData(token)
	if err != nil {
		// Return 404 to prevent enumeration
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	result, err := h.subscriptionService.GenerateV2Ray(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	// Log access
	elapsed := int(time.Since(start).Milliseconds())
	h.subscriptionService.LogAccess(data.User.ID, c.IP(), c.Get("User-Agent"), "v2ray", elapsed, false)

	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", data.User.Username))
	c.Set("Subscription-Userinfo", h.buildUserinfo(data))
	return c.SendString(result)
}

// GetClashSubscription serves Clash format subscription (YAML)
func (h *SubscriptionsHandler) GetClashSubscription(c fiber.Ctx) error {
	start := time.Now()
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	data, err := h.subscriptionService.GetUserSubscriptionData(token)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	result, err := h.subscriptionService.GenerateClash(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	elapsed := int(time.Since(start).Milliseconds())
	h.subscriptionService.LogAccess(data.User.ID, c.IP(), c.Get("User-Agent"), "clash", elapsed, false)

	c.Set("Content-Type", "text/yaml; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.yaml", data.User.Username))
	c.Set("Subscription-Userinfo", h.buildUserinfo(data))
	return c.SendString(result)
}

// GetSingboxSubscription serves Sing-box format subscription (JSON)
func (h *SubscriptionsHandler) GetSingboxSubscription(c fiber.Ctx) error {
	start := time.Now()
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	data, err := h.subscriptionService.GetUserSubscriptionData(token)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	result, err := h.subscriptionService.GenerateSingbox(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	elapsed := int(time.Since(start).Milliseconds())
	h.subscriptionService.LogAccess(data.User.ID, c.IP(), c.Get("User-Agent"), "singbox", elapsed, false)

	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", data.User.Username))
	c.Set("Subscription-Userinfo", h.buildUserinfo(data))
	return c.SendString(result)
}

// RedirectShortURL resolves a short code and redirects
func (h *SubscriptionsHandler) RedirectShortURL(c fiber.Ctx) error {
	shortCode := c.Params("code")
	if shortCode == "" {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	shortURL, err := h.subscriptionService.ResolveShortURL(shortCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	return c.Redirect().To(shortURL.FullURL)
}

// GetUserShortURL returns the short URL for a user (admin endpoint)
func (h *SubscriptionsHandler) GetUserShortURL(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Subscription token is required",
		})
	}

	shortURL, err := h.subscriptionService.GetOrCreateShortURL(uint(userID), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"short_code": shortURL.ShortCode,
		"short_url":  fmt.Sprintf("/s/%s", shortURL.ShortCode),
	})
}

// buildUserinfo builds the Subscription-Userinfo header
func (h *SubscriptionsHandler) buildUserinfo(data *services.UserSubscriptionData) string {
	parts := []string{
		fmt.Sprintf("upload=0; download=%d", data.User.TrafficUsedBytes),
	}
	if data.User.TrafficLimitBytes != nil {
		parts = append(parts, fmt.Sprintf("total=%d", *data.User.TrafficLimitBytes))
	}
	if data.User.ExpiryDate != nil {
		parts = append(parts, fmt.Sprintf("expire=%d", data.User.ExpiryDate.Unix()))
	}
	return joinParts(parts)
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "; "
		}
		result += p
	}
	return result
}
