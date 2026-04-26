package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/protocol"
)

type ProtocolsHandler struct{}

func NewProtocolsHandler() *ProtocolsHandler {
	return &ProtocolsHandler{}
}

// ListProtocols returns summaries of all registered protocols.
// Supports optional query params: ?core=xray&direction=inbound
//
// @Summary      List protocols
// @Description  Returns all registered protocol schemas (25+ protocols). Filter by core or direction.
// @Tags         protocols
// @Produce      json
// @Param        core       query  string  false  "Filter by core (xray, sing-box, mihomo)"
// @Param        direction  query  string  false  "Filter by direction (inbound, outbound, both)"
// @Success      200        {object}  map[string]interface{}
// @Router       /protocols [get]
// @Security     BearerAuth
func (h *ProtocolsHandler) ListProtocols(c fiber.Ctx) error {
	coreName := c.Query("core")
	direction := c.Query("direction")

	// Validate direction if provided
	if direction != "" && direction != "inbound" && direction != "outbound" && direction != "both" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid direction. Must be 'inbound', 'outbound', or 'both'",
		})
	}

	var summaries []protocol.ProtocolSummary

	switch {
	case coreName != "" && direction != "":
		summaries = protocol.GetProtocolsByCoreAndDirection(coreName, direction)
	case coreName != "":
		summaries = protocol.GetProtocolsByCore(coreName)
	default:
		summaries = protocol.GetAllProtocols()
		// If only direction filter provided, filter manually
		if direction != "" {
			filtered := make([]protocol.ProtocolSummary, 0)
			for _, s := range summaries {
				if s.Direction == direction || s.Direction == "both" {
					filtered = append(filtered, s)
				}
			}
			summaries = filtered
		}
	}

	return c.JSON(fiber.Map{
		"protocols": summaries,
		"total":     len(summaries),
	})
}

// GetProtocol returns the full schema for a specific protocol
//
// @Summary      Get protocol schema
// @Description  Returns the full parameter schema for a specific protocol (field types, validation, etc.)
// @Tags         protocols
// @Produce      json
// @Param        name  path  string  true  "Protocol name (e.g. vless, vmess, trojan)"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /protocols/{name} [get]
// @Security     BearerAuth
func (h *ProtocolsHandler) GetProtocol(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Protocol name is required",
		})
	}

	schema, ok := protocol.GetProtocolSchema(name)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Protocol not found",
		})
	}

	// Fill default widgets if not set
	for key, param := range schema.Parameters {
		if param.Widget == "" {
			param.Widget = protocol.DefaultWidget(param.Type)
			schema.Parameters[key] = param
		}
	}

	return c.JSON(schema)
}

// GetProtocolDefaults returns auto-generated default values for a protocol's parameters.
// This is used by the inbound creation wizard to pre-fill fields.
//
// @Summary      Get protocol defaults
// @Description  Returns auto-generated default values for all protocol parameters (UUID, port, etc.)
// @Tags         protocols
// @Produce      json
// @Param        name  path  string  true  "Protocol name"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /protocols/{name}/defaults [get]
// @Security     BearerAuth
func (h *ProtocolsHandler) GetProtocolDefaults(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Protocol name is required",
		})
	}

	schema, ok := protocol.GetProtocolSchema(name)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Protocol not found",
		})
	}

	defaults := make(map[string]interface{})
	for key, param := range schema.Parameters {
		if param.AutoGenerate && param.AutoGenFunc != "" {
			generated, err := protocol.AutoGenerate(param.AutoGenFunc)
			if err == nil {
				defaults[key] = generated
			}
		} else if param.Default != nil {
			defaults[key] = param.Default
		}
	}

	return c.JSON(fiber.Map{
		"protocol": name,
		"defaults": defaults,
	})
}
