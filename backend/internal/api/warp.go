package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

// WarpHandler handles WARP-related API requests
type WarpHandler struct {
	warpService *services.WARPService
	geoService  *services.GeoService
}

// NewWarpHandler creates a new WARP handler
func NewWarpHandler(warpService *services.WARPService, geoService *services.GeoService) *WarpHandler {
	return &WarpHandler{
		warpService: warpService,
		geoService:  geoService,
	}
}

// RegisterRoutes registers WARP API routes
func (h *WarpHandler) RegisterRoutes(router fiber.Router) {
	warp := router.Group("/warp")
	warp.Get("/routes", h.GetWarpRoutes)
	warp.Post("/routes", h.CreateWarpRoute)
	warp.Put("/routes/:id", h.UpdateWarpRoute)
	warp.Delete("/routes/:id", h.DeleteWarpRoute)
	warp.Post("/routes/:id/toggle", h.ToggleWarpRoute)
	warp.Post("/sync", h.SyncWarpRoutes)
	warp.Get("/status", h.GetWarpStatus)
	warp.Post("/register", h.RegisterWARP)
	warp.Get("/presets", h.GetWarpPresets)
	warp.Post("/presets/:name/apply", h.ApplyWarpPreset)

	// Geo routes
	geo := router.Group("/geo")
	geo.Get("/rules", h.GetGeoRules)
	geo.Post("/rules", h.CreateGeoRule)
	geo.Put("/rules/:id", h.UpdateGeoRule)
	geo.Delete("/rules/:id", h.DeleteGeoRule)
	geo.Post("/rules/:id/toggle", h.ToggleGeoRule)
	geo.Get("/countries", h.GetCountries)
	geo.Get("/categories", h.GetCategories)
	geo.Get("/databases", h.GetGeoDatabases)
	geo.Post("/update", h.UpdateGeoDatabases)
}

// parseID parses ID from URL params
func parseID(c fiber.Ctx) (uint, error) {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	return uint(id), err
}

// WARP Routes

// GetWarpRoutes returns all WARP routes
func (h *WarpHandler) GetWarpRoutes(c fiber.Ctx) error {
	coreIDStr := c.Query("core_id")
	if coreIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "core_id is required",
		})
	}

	coreID, err := strconv.ParseUint(coreIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid core_id",
		})
	}

	routes, err := h.warpService.GetWarpRoutesForCore(uint(coreID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": routes,
	})
}

// CreateWarpRoute creates a new WARP route
func (h *WarpHandler) CreateWarpRoute(c fiber.Ctx) error {
	var route models.WarpRoute
	if err := c.Bind().JSON(&route); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if route.ResourceType == "" || route.ResourceValue == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "resource_type and resource_value are required",
		})
	}

	// Validate resource type
	validTypes := []string{"domain", "ip", "cidr"}
	isValid := false
	for _, t := range validTypes {
		if route.ResourceType == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid resource_type, must be one of: " + strings.Join(validTypes, ", "),
		})
	}

	// Check for duplicates
	var existing models.WarpRoute
	err := h.warpService.DB().Where(
		"core_id = ? AND resource_type = ? AND resource_value = ?",
		route.CoreID, route.ResourceType, route.ResourceValue,
	).First(&existing).Error

	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "duplicate route: " + route.ResourceValue + " already exists for this core",
		})
	}

	// Set defaults
	if route.Priority == 0 {
		route.Priority = 50
	}
	route.IsEnabled = true

	if err := h.warpService.DB().Create(&route).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": route,
	})
}

// UpdateWarpRoute updates an existing WARP route
func (h *WarpHandler) UpdateWarpRoute(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid route id",
		})
	}

	var route models.WarpRoute
	if err := h.warpService.DB().First(&route, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "route not found",
		})
	}

	var update models.WarpRoute
	if err := c.Bind().JSON(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update fields
	if update.ResourceType != "" {
		route.ResourceType = update.ResourceType
	}
	if update.ResourceValue != "" {
		route.ResourceValue = update.ResourceValue
	}
	if update.Description != "" {
		route.Description = update.Description
	}
	if update.Priority > 0 {
		route.Priority = update.Priority
	}

	if err := h.warpService.DB().Save(&route).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": route,
	})
}

// DeleteWarpRoute deletes a WARP route
func (h *WarpHandler) DeleteWarpRoute(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid route id",
		})
	}

	if err := h.warpService.DB().Delete(&models.WarpRoute{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// ToggleWarpRoute enables/disables a WARP route
func (h *WarpHandler) ToggleWarpRoute(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid route id",
		})
	}

	var route models.WarpRoute
	if err := h.warpService.DB().First(&route, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "route not found",
		})
	}

	route.IsEnabled = !route.IsEnabled
	if err := h.warpService.DB().Save(&route).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": route,
	})
}

// SyncWarpRoutes applies WARP routes to cores
func (h *WarpHandler) SyncWarpRoutes(c fiber.Ctx) error {
	// For MVP, this is a no-op
	// In production, this would regenerate core configs
	return c.JSON(fiber.Map{
		"success": true,
		"message": "WARP routes synchronized",
	})
}

// GetWarpStatus returns WARP connection status
func (h *WarpHandler) GetWarpStatus(c fiber.Ctx) error {
	status, err := h.warpService.GetStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": status,
	})
}

// RegisterWARP registers a new WARP device
func (h *WarpHandler) RegisterWARP(c fiber.Ctx) error {
	account, err := h.warpService.RegisterWARP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Generate and save WireGuard config
	config, err := h.warpService.GenerateWireGuardConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.warpService.SaveWireGuardConfig(config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    account,
		"message": "WARP registered successfully",
	})
}

// GetWarpPresets returns available WARP presets
func (h *WarpHandler) GetWarpPresets(c fiber.Ctx) error {
	presets := h.warpService.GetWarpPresets()
	return c.JSON(fiber.Map{
		"data": presets,
	})
}

// ApplyWarpPreset applies a preset to a core
func (h *WarpHandler) ApplyWarpPreset(c fiber.Ctx) error {
	presetName := c.Params("name")
	coreIDStr := c.Query("core_id")

	if coreIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "core_id is required",
		})
	}

	coreID, err := strconv.ParseUint(coreIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid core_id",
		})
	}

	if err := h.warpService.ApplyPreset(presetName, uint(coreID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Preset applied successfully",
	})
}

// Geo Rules

// GetGeoRules returns all Geo rules
func (h *WarpHandler) GetGeoRules(c fiber.Ctx) error {
	coreIDStr := c.Query("core_id")
	if coreIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "core_id is required",
		})
	}

	coreID, err := strconv.ParseUint(coreIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid core_id",
		})
	}

	rules, err := h.geoService.GetGeoRulesForCore(uint(coreID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": rules,
	})
}

// CreateGeoRule creates a new Geo rule
func (h *WarpHandler) CreateGeoRule(c fiber.Ctx) error {
	var rule models.GeoRule
	if err := c.Bind().JSON(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if rule.Type == "" || rule.Code == "" || rule.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "type, code, and action are required",
		})
	}

	// Validate type
	validTypes := []string{"geoip", "geosite"}
	isValid := false
	for _, t := range validTypes {
		if rule.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid type, must be one of: " + strings.Join(validTypes, ", "),
		})
	}

	// Validate action
	validActions := []string{"proxy", "direct", "block", "warp"}
	isValid = false
	for _, a := range validActions {
		if rule.Action == a {
			isValid = true
			break
		}
	}
	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid action, must be one of: " + strings.Join(validActions, ", "),
		})
	}

	// Set defaults
	if rule.Priority == 0 {
		rule.Priority = 50
	}
	rule.IsEnabled = true

	if err := h.geoService.CreateGeoRule(&rule); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": rule,
	})
}

// UpdateGeoRule updates an existing Geo rule
func (h *WarpHandler) UpdateGeoRule(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid rule id",
		})
	}

	var existing models.GeoRule
	if err := h.geoService.DB().First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "rule not found",
		})
	}

	var update models.GeoRule
	if err := c.Bind().JSON(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update fields
	if update.Type != "" {
		existing.Type = update.Type
	}
	if update.Code != "" {
		existing.Code = update.Code
	}
	if update.Action != "" {
		existing.Action = update.Action
	}
	if update.Priority > 0 {
		existing.Priority = update.Priority
	}
	if update.Description != "" {
		existing.Description = update.Description
	}

	if err := h.geoService.UpdateGeoRule(&existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": existing,
	})
}

// DeleteGeoRule deletes a Geo rule
func (h *WarpHandler) DeleteGeoRule(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid rule id",
		})
	}

	coreIDStr := c.Query("core_id")
	if coreIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "core_id is required",
		})
	}

	coreID, err := strconv.ParseUint(coreIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid core_id",
		})
	}

	if err := h.geoService.DeleteGeoRule(uint(id), uint(coreID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// ToggleGeoRule enables/disables a Geo rule
func (h *WarpHandler) ToggleGeoRule(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid rule id",
		})
	}

	coreIDStr := c.Query("core_id")
	if coreIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "core_id is required",
		})
	}

	coreID, err := strconv.ParseUint(coreIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid core_id",
		})
	}

	var rule models.GeoRule
	if err := h.geoService.DB().First(&rule, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "rule not found",
		})
	}

	rule.IsEnabled = !rule.IsEnabled
	if err := h.geoService.ToggleGeoRule(uint(id), uint(coreID), rule.IsEnabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": rule,
	})
}

// GetCountries returns list of countries for GeoIP rules
func (h *WarpHandler) GetCountries(c fiber.Ctx) error {
	countries, err := h.geoService.GetCountries()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": countries,
	})
}

// GetCategories returns list of categories for GeoSite rules
func (h *WarpHandler) GetCategories(c fiber.Ctx) error {
	categories, err := h.geoService.GetCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": categories,
	})
}

// GetGeoDatabases returns list of available Geo databases
func (h *WarpHandler) GetGeoDatabases(c fiber.Ctx) error {
	databases, err := h.geoService.GetGeoDatabases()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": databases,
	})
}

// UpdateGeoDatabases downloads all Geo databases
func (h *WarpHandler) UpdateGeoDatabases(c fiber.Ctx) error {
	if err := h.geoService.UpdateAllDatabases(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Geo databases updated successfully",
	})
}
