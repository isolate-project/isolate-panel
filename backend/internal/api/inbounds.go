package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

type InboundsHandler struct {
	inboundService *services.InboundService
}

func NewInboundsHandler(inboundService *services.InboundService) *InboundsHandler {
	return &InboundsHandler{
		inboundService: inboundService,
	}
}

// ListInbounds returns all inbounds with optional filtering
func (h *InboundsHandler) ListInbounds(c fiber.Ctx) error {
	// Parse query parameters
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

	inbounds, err := h.inboundService.ListInbounds(coreID, isEnabled)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list inbounds",
		})
	}

	return c.JSON(inbounds)
}

// GetInbound returns a specific inbound
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
func (h *InboundsHandler) CreateInbound(c fiber.Ctx) error {
	var inbound models.Inbound
	if err := c.Bind().JSON(&inbound); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.inboundService.CreateInbound(&inbound); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(inbound)
}

// UpdateInbound updates an existing inbound
func (h *InboundsHandler) UpdateInbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid inbound ID",
		})
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
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
