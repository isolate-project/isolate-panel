package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// TrafficResetScheduler is a minimal interface for the traffic reset scheduler,
// defined here to avoid an import cycle between api and scheduler packages.
type TrafficResetScheduler interface {
	GetSchedule() (string, error)
	UpdateSchedule(schedule string) error
}

// SettingsHandler handles settings-related HTTP requests
type SettingsHandler struct {
	settingsService       *services.SettingsService
	trafficCollector      *services.TrafficCollector
	trafficResetScheduler TrafficResetScheduler
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsService *services.SettingsService, trafficCollector *services.TrafficCollector) *SettingsHandler {
	return &SettingsHandler{
		settingsService:  settingsService,
		trafficCollector: trafficCollector,
	}
}

// SetTrafficResetScheduler wires in the scheduler after construction.
func (h *SettingsHandler) SetTrafficResetScheduler(sched TrafficResetScheduler) {
	h.trafficResetScheduler = sched
}

// GetMonitoring returns the current monitoring configuration
//
// @Summary      Get monitoring settings
// @Description  Returns current monitoring mode (lite/full) and collection interval
// @Tags         settings
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /settings/monitoring [get]
// @Security     BearerAuth
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
//
// @Summary      Update monitoring settings
// @Description  Set monitoring mode to 'lite' (traffic only) or 'full' (traffic + connections)
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]string  true  "{mode: lite|full}"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /settings/monitoring [put]
// @Security     BearerAuth
func (h *SettingsHandler) UpdateMonitoring(c fiber.Ctx) error {
	var req struct {
		Mode string `json:"mode"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.Mode != "lite" && req.Mode != "full" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid monitoring mode. Must be 'lite' or 'full'",
		})
	}

	err := h.settingsService.UpdateMonitoringMode(req.Mode)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
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
//
// @Summary      Get all settings
// @Description  Returns all key-value application settings
// @Tags         settings
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /settings [get]
// @Security     BearerAuth
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
//
// @Summary      Update settings
// @Description  Update one or more application settings (key-value pairs)
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]interface{}  true  "{settings: {key: value}}"
// @Success      200   {object}  map[string]interface{}
// @Router       /settings [put]
// @Security     BearerAuth
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

// GetTrafficResetSchedule returns the current traffic auto-reset schedule.
//
// @Summary      Get traffic reset schedule
// @Description  Returns the current automatic traffic reset schedule (disabled/weekly/monthly)
// @Tags         settings
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /settings/traffic-reset [get]
// @Security     BearerAuth
func (h *SettingsHandler) GetTrafficResetSchedule(c fiber.Ctx) error {
	if h.trafficResetScheduler == nil {
		return c.JSON(fiber.Map{"schedule": "disabled"})
	}
	schedule, err := h.trafficResetScheduler.GetSchedule()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"schedule": schedule})
}

// UpdateTrafficResetSchedule updates the traffic auto-reset schedule.
//
// @Summary      Update traffic reset schedule
// @Description  Set automatic traffic reset schedule. Options: disabled, weekly (Mondays), monthly (1st of month)
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]string  true  "{schedule: disabled|weekly|monthly}"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /settings/traffic-reset [put]
// @Security     BearerAuth
func (h *SettingsHandler) UpdateTrafficResetSchedule(c fiber.Ctx) error {
	var req struct {
		Schedule string `json:"schedule"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	switch req.Schedule {
	case "disabled", "weekly", "monthly":
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "schedule must be one of: disabled, weekly, monthly",
		})
	}
	if h.trafficResetScheduler == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "scheduler not available"})
	}
	if err := h.trafficResetScheduler.UpdateSchedule(req.Schedule); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "schedule": req.Schedule})
}
