package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/haproxy"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
)

type InboundsHandler struct {
	inboundService *services.InboundService
	portManager    *services.PortManager
	portValidator  *haproxy.PortValidator
	db             *gorm.DB
}

func NewInboundsHandler(inboundService *services.InboundService, portManager *services.PortManager, portValidator *haproxy.PortValidator, db *gorm.DB) *InboundsHandler {
	return &InboundsHandler{
		inboundService: inboundService,
		portManager:    portManager,
		portValidator:  portValidator,
		db:             db,
	}
}

// ListInbounds returns all inbounds with optional filtering
//
// @Summary      List inbounds
// @Description  Returns all inbounds, optionally filtered by core_id or is_enabled
// @Tags         inbounds
// @Produce      json
// @Param        core_id     query  int     false  "Filter by core ID"
// @Param        is_enabled  query  bool    false  "Filter by enabled state"
// @Success      200         {object}  map[string]interface{}
// @Router       /inbounds [get]
// @Security     BearerAuth
func (h *InboundsHandler) ListInbounds(c fiber.Ctx) error {
	params := GetPagination(c)

	var coreID *uint
	if coreIDStr := c.Query("core_id"); coreIDStr != "" {
		id, err := strconv.ParseUint(coreIDStr, 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid core_id parameter",
			})
		}
		coreIDVal := uint(id)
		coreID = &coreIDVal
	}

	var isEnabled *bool
	if isEnabledStr := c.Query("is_enabled"); isEnabledStr != "" {
		enabled := isEnabledStr == "true"
		isEnabled = &enabled
	}

	inbounds, total, err := h.inboundService.ListInboundsPaginated(coreID, isEnabled, params.Page, params.PageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list inbounds",
		})
	}

	totalPages := (total + int64(params.PageSize) - 1) / int64(params.PageSize)

	return c.JSON(fiber.Map{
		"success":   true,
		"inbounds":  inbounds,
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
		"pages":     totalPages,
	})
}

// GetInbound returns a specific inbound
//
// @Summary      Get inbound
// @Description  Returns a single inbound by ID
// @Tags         inbounds
// @Produce      json
// @Param        id   path  int  true  "Inbound ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /inbounds/{id} [get]
// @Security     BearerAuth
func (h *InboundsHandler) GetInbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid inbound ID",
		})
	}

	inbound, err := h.inboundService.GetInbound(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Inbound not found",
		})
	}

	return c.JSON(inbound)
}

// CreateInbound creates a new inbound
//
// @Summary      Create inbound
// @Description  Create a new proxy inbound (listener) on a core
// @Tags         inbounds
// @Accept       json
// @Produce      json
// @Param        body  body  models.Inbound  true  "Inbound configuration"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /inbounds [post]
// @Security     BearerAuth
func (h *InboundsHandler) CreateInbound(c fiber.Ctx) error {
	inbound, err := middleware.BindAndValidate[models.Inbound](c)
	if err != nil {
		return err
	}

	if err := h.inboundService.CreateInbound(&inbound); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(inbound)
}

// UpdateInbound updates an existing inbound
//
// @Summary      Update inbound
// @Description  Update inbound fields (partial update via map)
// @Tags         inbounds
// @Accept       json
// @Produce      json
// @Param        id    path  int                     true  "Inbound ID"
// @Param        body  body  map[string]interface{}  true  "Fields to update"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /inbounds/{id} [put]
// @Security     BearerAuth
func (h *InboundsHandler) UpdateInbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid inbound ID",
		})
	}

	var req UpdateInboundDTO
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	updates := req.ToMap()
	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No fields to update",
		})
	}

	inbound, err := h.inboundService.UpdateInbound(uint(id), updates)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(inbound)
}

// DeleteInbound deletes an inbound
//
// @Summary      Delete inbound
// @Description  Delete an inbound and remove all user assignments
// @Tags         inbounds
// @Produce      json
// @Param        id   path  int  true  "Inbound ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /inbounds/{id} [delete]
// @Security     BearerAuth
func (h *InboundsHandler) DeleteInbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid inbound ID",
		})
	}

	if err := h.inboundService.DeleteInbound(uint(id)); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Inbound deleted successfully",
	})
}

// GetInboundsByCore returns all inbounds for a specific core
//
// @Summary      Get inbounds by core
// @Description  Returns all inbounds belonging to a specific core
// @Tags         inbounds
// @Produce      json
// @Param        core_id  path  int  true  "Core ID"
// @Success      200      {object}  map[string]interface{}
// @Router       /inbounds/core/{core_id} [get]
// @Security     BearerAuth
func (h *InboundsHandler) GetInboundsByCore(c fiber.Ctx) error {
	coreID, err := strconv.ParseUint(c.Params("core_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid core ID",
		})
	}

	inbounds, err := h.inboundService.GetInboundsByCore(uint(coreID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get inbounds",
		})
	}

	return c.JSON(inbounds)
}

// AssignInboundToUser assigns an inbound to a user
//
// @Summary      Assign inbound to user
// @Description  Grant a user access to an inbound
// @Tags         inbounds
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]interface{}  true  "{user_id, inbound_id}"
// @Success      200   {object}  map[string]interface{}
// @Router       /inbounds/assign [post]
// @Security     BearerAuth
func (h *InboundsHandler) AssignInboundToUser(c fiber.Ctx) error {
	type AssignRequest struct {
		UserID    uint `json:"user_id"`
		InboundID uint `json:"inbound_id"`
	}

	var req AssignRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.inboundService.AssignInboundToUser(req.UserID, req.InboundID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Inbound assigned to user successfully",
	})
}

// UnassignInboundFromUser removes an inbound assignment from a user
//
// @Summary      Unassign inbound from user
// @Description  Revoke a user's access to an inbound
// @Tags         inbounds
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]interface{}  true  "{user_id, inbound_id}"
// @Success      200   {object}  map[string]interface{}
// @Router       /inbounds/unassign [post]
// @Security     BearerAuth
func (h *InboundsHandler) UnassignInboundFromUser(c fiber.Ctx) error {
	type UnassignRequest struct {
		UserID    uint `json:"user_id"`
		InboundID uint `json:"inbound_id"`
	}

	var req UnassignRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.inboundService.UnassignInboundFromUser(req.UserID, req.InboundID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Inbound unassigned from user successfully",
	})
}

// GetInboundUsers returns all users assigned to an inbound
//
// @Summary      Get inbound users
// @Description  Returns all users that have access to a specific inbound
// @Tags         inbounds
// @Produce      json
// @Param        id   path  int  true  "Inbound ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /inbounds/{id}/users [get]
// @Security     BearerAuth
func (h *InboundsHandler) GetInboundUsers(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid inbound ID",
		})
	}

	users, err := h.inboundService.GetInboundUsers(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": len(users),
	})
}

// BulkAssignUsers bulk adds/removes users from an inbound
//
// @Summary      Bulk assign users
// @Description  Add or remove multiple users from an inbound in a single request
// @Tags         inbounds
// @Accept       json
// @Produce      json
// @Param        id    path  int                     true  "Inbound ID"
// @Param        body  body  map[string]interface{}  true  "{add_user_ids: [], remove_user_ids: []}"
// @Success      200   {object}  map[string]interface{}
// @Router       /inbounds/{id}/users/bulk [post]
// @Security     BearerAuth
func (h *InboundsHandler) BulkAssignUsers(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid inbound ID",
		})
	}

	type BulkRequest struct {
		AddUserIDs    []uint `json:"add_user_ids"`
		RemoveUserIDs []uint `json:"remove_user_ids"`
	}

	var req BulkRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	added, removed, err := h.inboundService.BulkAssignUsers(uint(id), req.AddUserIDs, req.RemoveUserIDs)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Bulk operation completed",
		"added":   added,
		"removed": removed,
	})
}

// CheckPort checks if a port is available
//
// @Summary      Check port availability
// @Description  Returns whether the given port is available or already in use
// @Tags         inbounds
// @Produce      json
// @Param        port        query  int  true   "Port number to check"
// @Param        exclude_id  query  int  false  "Inbound ID to exclude from conflict check"
// @Success      200         {object}  map[string]interface{}
// @Router       /inbounds/check-port [get]
// @Security     BearerAuth
func (h *InboundsHandler) CheckPort(c fiber.Ctx) error {
	portStr := c.Query("port")
	if portStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Port parameter is required",
		})
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid port number",
		})
	}

	var excludeID *uint
	if excludeIDStr := c.Query("exclude_id"); excludeIDStr != "" {
		id, err := strconv.ParseUint(excludeIDStr, 10, 32)
		if err == nil {
			idVal := uint(id)
			excludeID = &idVal
		}
	}

	available, reason, err := h.portManager.IsPortAvailable(port, excludeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"available": available,
		"reason":    reason,
	})
}

func (h *InboundsHandler) CheckPortAvailability(c fiber.Ctx) error {
	var req CheckPortRequestDTO
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Listen == "" {
		req.Listen = "0.0.0.0"
	}

	var inbounds []models.Inbound
	if err := h.db.Where("is_enabled = ?", true).Find(&inbounds).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch inbounds",
		})
	}

	result := h.portValidator.ValidatePortConflict(
		req.Port,
		req.Listen,
		req.Protocol,
		req.Transport,
		req.CoreType,
		inbounds,
	)

	return c.JSON(result)
}
