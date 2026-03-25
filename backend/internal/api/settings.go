package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

// SettingsHandler handles settings-related HTTP requests
type SettingsHandler struct {
	settingsService  *services.SettingsService
	trafficCollector *services.TrafficCollector
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsService *services.SettingsService, trafficCollector *services.TrafficCollector) *SettingsHandler {
	return &SettingsHandler{
		settingsService:  settingsService,
		trafficCollector: trafficCollector,
	}
}

// GetMonitoring returns the current monitoring configuration
// GET /api/settings/monitoring
func (h *SettingsHandler) GetMonitoring(c fiber.Ctx) error {
	mode, err := h.settingsService.GetMonitoringMode()
	if err != nil {
		return err
	}

	interval, err := h.settingsService.GetMonitoringInterval()
	if err != nil {
		interval = 60 // Default to 60 seconds
	}

	return c.JSON(fiber.Map{
		"mode":     mode,
		"interval": interval.Seconds(),
		"success":  true,
	})
}

// UpdateMonitoring updates the monitoring mode
// PUT /api/settings/monitoring
func (h *SettingsHandler) UpdateMonitoring(c fiber.Ctx) error {
	var req struct {
		Mode string `json:"mode"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if req.Mode != "lite" && req.Mode != "full" {
		return fiber.ErrBadRequest
	}

	err := h.settingsService.UpdateMonitoringMode(req.Mode)
	if err != nil {
		return err
	}

	// Notify traffic collector to reload interval
	if h.trafficCollector != nil {
		h.trafficCollector.ReloadInterval()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"mode":    req.Mode,
		"message": "Monitoring mode updated successfully",
	})
}

// GetAllSettings returns all application settings
// GET /api/settings
func (h *SettingsHandler) GetAllSettings(c fiber.Ctx) error {
	settings, err := h.settingsService.GetAllSettings()
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"settings": settings,
	})
}

// UpdateSettings updates multiple settings
// PUT /api/settings
func (h *SettingsHandler) UpdateSettings(c fiber.Ctx) error {
	var req struct {
		Settings map[string]string `json:"settings"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return fiber.ErrBadRequest
	}

	err := h.settingsService.UpdateSettings(req.Settings)
	if err != nil {
		return err
	}

	// Check if monitoring_mode was updated and reload if needed
	if mode, ok := req.Settings["monitoring_mode"]; ok {
		if mode == "lite" || mode == "full" {
			if h.trafficCollector != nil {
				h.trafficCollector.ReloadInterval()
			}
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Settings updated successfully",
	})
}
