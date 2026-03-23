package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/core"
)

type CoresHandler struct {
	coreManager *core.CoreManager
}

func NewCoresHandler(coreManager *core.CoreManager) *CoresHandler {
	return &CoresHandler{
		coreManager: coreManager,
	}
}

// ListCores returns all cores
func (h *CoresHandler) ListCores(c fiber.Ctx) error {
	cores, err := h.coreManager.ListCores()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list cores",
		})
	}

	return c.JSON(cores)
}

// GetCore returns a specific core
func (h *CoresHandler) GetCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	coreInfo, err := h.coreManager.GetCoreStatus(name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Core not found",
		})
	}

	return c.JSON(coreInfo)
}

// StartCore starts a core
func (h *CoresHandler) StartCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	if err := h.coreManager.StartCore(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Core started successfully",
		"core":    name,
	})
}

// StopCore stops a core
func (h *CoresHandler) StopCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	if err := h.coreManager.StopCore(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Core stopped successfully",
		"core":    name,
	})
}

// RestartCore restarts a core
func (h *CoresHandler) RestartCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	if err := h.coreManager.RestartCore(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Core restarted successfully",
		"core":    name,
	})
}

// GetCoreStatus returns the status of a core
func (h *CoresHandler) GetCoreStatus(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	coreInfo, err := h.coreManager.GetCoreStatus(name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Core not found",
		})
	}

	return c.JSON(fiber.Map{
		"name":       coreInfo.Name,
		"is_running": coreInfo.IsRunning,
		"is_enabled": coreInfo.IsEnabled,
		"pid":        coreInfo.PID,
		"uptime":     coreInfo.UptimeSeconds,
		"restarts":   coreInfo.RestartCount,
		"last_error": coreInfo.LastError,
	})
}
