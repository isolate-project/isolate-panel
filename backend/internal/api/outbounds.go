package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

type OutboundsHandler struct {
	outboundService *services.OutboundService
}

func NewOutboundsHandler(outboundService *services.OutboundService) *OutboundsHandler {
	return &OutboundsHandler{
		outboundService: outboundService,
	}
}

// ListOutbounds returns all outbounds with optional filtering
func (h *OutboundsHandler) ListOutbounds(c fiber.Ctx) error {
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

	protocolFilter := c.Query("protocol")

	outbounds, err := h.outboundService.ListOutbounds(coreID, protocolFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list outbounds",
		})
	}

	return c.JSON(outbounds)
}

// GetOutbound returns a specific outbound
func (h *OutboundsHandler) GetOutbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid outbound ID",
		})
	}

	outbound, err := h.outboundService.GetOutbound(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Outbound not found",
		})
	}

	return c.JSON(outbound)
}

// CreateOutbound creates a new outbound
func (h *OutboundsHandler) CreateOutbound(c fiber.Ctx) error {
	var outbound models.Outbound
	if err := c.Bind().JSON(&outbound); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.outboundService.CreateOutbound(&outbound); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(outbound)
}

// UpdateOutbound updates an existing outbound
func (h *OutboundsHandler) UpdateOutbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid outbound ID",
		})
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	outbound, err := h.outboundService.UpdateOutbound(uint(id), updates)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(outbound)
}

// DeleteOutbound deletes an outbound
func (h *OutboundsHandler) DeleteOutbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid outbound ID",
		})
	}

	if err := h.outboundService.DeleteOutbound(uint(id)); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Outbound deleted successfully",
	})
}
