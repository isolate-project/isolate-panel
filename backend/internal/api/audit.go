package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/services"
)

type AuditHandler struct {
	auditSvc *services.AuditService
}

func NewAuditHandler(auditSvc *services.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

// ListAuditLogs handles GET /api/audit-logs
//
// @Summary      List audit logs
// @Description  Returns paginated audit log entries. Super-admin only.
// @Tags         audit
// @Produce      json
// @Param        page      query  int     false  "Page number"      default(1)
// @Param        limit     query  int     false  "Items per page"   default(50)
// @Param        action    query  string  false  "Filter by action (e.g. user.delete)"
// @Param        admin_id  query  int     false  "Filter by admin ID"
// @Success      200       {object}  map[string]interface{}
// @Router       /audit-logs [get]
// @Security     BearerAuth
func (h *AuditHandler) ListAuditLogs(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	var adminID uint
	if raw := c.Query("admin_id"); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
			adminID = uint(v)
		}
	}

	result, err := h.auditSvc.List(services.AuditListOptions{
		Action:   c.Query("action"),
		AdminID:  adminID,
		Page:     page,
		PageSize: limit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list audit logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  result.Logs,
		"total": result.Total,
		"page":  page,
		"limit": limit,
	})
}
