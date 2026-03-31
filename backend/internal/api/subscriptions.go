package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/skip2/go-qrcode"

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

// GetAutoDetectSubscription inspects User-Agent to return the appropriate subscription format
func (h *SubscriptionsHandler) GetAutoDetectSubscription(c fiber.Ctx) error {
	userAgent := strings.ToLower(c.Get("User-Agent"))

	if strings.Contains(userAgent, "clash") || strings.Contains(userAgent, "mihomo") {
		return h.GetClashSubscription(c)
	}

	if strings.Contains(userAgent, "sing-box") {
		return h.GetSingboxSubscription(c)
	}

	// Default to V2Ray format (Base64 vless/vmess/etc.)
	return h.GetV2RaySubscription(c)
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
	c.Set("Profile-Update-Interval", "24") // 24 hours
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
	c.Set("Profile-Update-Interval", "24") // 24 hours
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
	c.Set("Profile-Update-Interval", "24") // 24 hours
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

// GetQRCode generates a QR code for the subscription URL
func (h *SubscriptionsHandler) GetQRCode(c fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	// Verify the token is valid
	_, err := h.subscriptionService.GetUserBySubscriptionToken(token)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	// Build subscription URL
	scheme := "https"
	if c.Protocol() == "http" {
		scheme = "http"
	}
	subscriptionURL := fmt.Sprintf("%s://%s/sub/%s", scheme, c.Hostname(), token)

	// Generate QR code
	png, err := qrcode.Encode(subscriptionURL, qrcode.Medium, 256)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to generate QR code")
	}

	c.Set("Content-Type", "image/png")
	return c.Send(png)
}

// GetAccessStats returns subscription access statistics (admin endpoint)
func (h *SubscriptionsHandler) GetAccessStats(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	daysStr := c.Query("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		days = 7
	}

	stats, err := h.subscriptionService.GetAccessStats(uint(userID), days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// RegenerateToken regenerates a user's subscription token (admin endpoint)
func (h *SubscriptionsHandler) RegenerateToken(c fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	newToken, err := h.subscriptionService.RegenerateToken(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Build URLs
	scheme := "https"
	if c.Protocol() == "http" {
		scheme = "http"
	}
	host := c.Hostname()

	return c.JSON(fiber.Map{
		"subscription_token": newToken,
		"subscription_url":   fmt.Sprintf("%s://%s/sub/%s", scheme, host, newToken),
		"clash_url":          fmt.Sprintf("%s://%s/sub/%s/clash", scheme, host, newToken),
		"singbox_url":        fmt.Sprintf("%s://%s/sub/%s/singbox", scheme, host, newToken),
		"qr_code_url":        fmt.Sprintf("%s://%s/sub/%s/qr", scheme, host, newToken),
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
