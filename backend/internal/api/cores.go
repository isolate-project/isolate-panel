package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/cores"
)

type CoresHandler struct {
	coreManager *cores.CoreManager
}

func NewCoresHandler(coreManager *cores.CoreManager) *CoresHandler {
	return &CoresHandler{
		coreManager: coreManager,
	}
}

// ListCores returns all cores
//
// @Summary      List cores
// @Description  Returns all installed proxy cores with their running status
// @Tags         cores
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /cores [get]
// @Security     BearerAuth
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
//
// @Summary      Get core
// @Description  Returns status and info for a specific proxy core
// @Tags         cores
// @Produce      json
// @Param        name  path  string  true  "Core name (xray, sing-box, mihomo)"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /cores/{name} [get]
// @Security     BearerAuth
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
//
// @Summary      Start core
// @Description  Start a proxy core via Supervisord
// @Tags         cores
// @Produce      json
// @Param        name  path  string  true  "Core name"
// @Success      200   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /cores/{name}/start [post]
// @Security     BearerAuth
func (h *CoresHandler) StartCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	if err := h.coreManager.StartCore(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Core started successfully",
		"core":    name,
	})
}

// StopCore stops a core
//
// @Summary      Stop core
// @Description  Stop a running proxy core via Supervisord
// @Tags         cores
// @Produce      json
// @Param        name  path  string  true  "Core name"
// @Success      200   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /cores/{name}/stop [post]
// @Security     BearerAuth
func (h *CoresHandler) StopCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	if err := h.coreManager.StopCore(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Core stopped successfully",
		"core":    name,
	})
}

// RestartCore restarts a core
//
// @Summary      Restart core
// @Description  Restart a proxy core (stop + start)
// @Tags         cores
// @Produce      json
// @Param        name  path  string  true  "Core name"
// @Success      200   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /cores/{name}/restart [post]
// @Security     BearerAuth
func (h *CoresHandler) RestartCore(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Core name is required",
		})
	}

	if err := h.coreManager.RestartCore(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Core restarted successfully",
		"core":    name,
	})
}

// GetCoreStatus returns the status of a core
//
// @Summary      Core status
// @Description  Returns detailed runtime status of a proxy core (PID, uptime, restarts)
// @Tags         cores
// @Produce      json
// @Param        name  path  string  true  "Core name"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /cores/{name}/status [get]
// @Security     BearerAuth
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
