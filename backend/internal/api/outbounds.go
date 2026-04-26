package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
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
//
// @Summary      List outbounds
// @Description  Returns all outbounds, optionally filtered by core_id or protocol
// @Tags         outbounds
// @Produce      json
// @Param        core_id   query  int     false  "Filter by core ID"
// @Param        protocol  query  string  false  "Filter by protocol name"
// @Success      200       {object}  map[string]interface{}
// @Router       /outbounds [get]
// @Security     BearerAuth
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
//
// @Summary      Get outbound
// @Description  Returns a single outbound by ID
// @Tags         outbounds
// @Produce      json
// @Param        id   path  int  true  "Outbound ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /outbounds/{id} [get]
// @Security     BearerAuth
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
//
// @Summary      Create outbound
// @Description  Create a new proxy outbound routing rule
// @Tags         outbounds
// @Accept       json
// @Produce      json
// @Param        body  body  models.Outbound  true  "Outbound configuration"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /outbounds [post]
// @Security     BearerAuth
func (h *OutboundsHandler) CreateOutbound(c fiber.Ctx) error {
	outbound, err := middleware.BindAndValidate[models.Outbound](c)
	if err != nil {
		return err
	}

	if err := h.outboundService.CreateOutbound(&outbound); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(outbound)
}

// UpdateOutbound updates an existing outbound
//
// @Summary      Update outbound
// @Description  Update outbound fields (partial update)
// @Tags         outbounds
// @Accept       json
// @Produce      json
// @Param        id    path  int                     true  "Outbound ID"
// @Param        body  body  map[string]interface{}  true  "Fields to update"
// @Success      200   {object}  map[string]interface{}
// @Router       /outbounds/{id} [put]
// @Security     BearerAuth
func (h *OutboundsHandler) UpdateOutbound(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid outbound ID",
		})
	}

	var req UpdateOutboundDTO
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

	outbound, err := h.outboundService.UpdateOutbound(uint(id), updates)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(outbound)
}

// DeleteOutbound deletes an outbound
//
// @Summary      Delete outbound
// @Description  Delete an outbound routing rule
// @Tags         outbounds
// @Produce      json
// @Param        id   path  int  true  "Outbound ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /outbounds/{id} [delete]
// @Security     BearerAuth
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
