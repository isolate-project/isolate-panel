package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/services"
)

// AuditAction returns a post-handler middleware that writes an audit entry
// when the handler completes successfully (status < 400).
// action and resource are constants like "user.delete" / "user".
// The resource ID is read from the "id" route param when present.
func AuditAction(auditSvc *services.AuditService, action, resource string) fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()

		// Only log on success
		if err != nil || c.Response().StatusCode() >= 400 {
			return err
		}

		adminID, _ := c.Locals("admin_id").(uint)

		var resourceID *uint
		if raw := c.Params("id"); raw != "" {
			var id uint
			if parseUintParam(raw, &id) == nil {
				resourceID = &id
			}
		}

		auditSvc.Log(adminID, action, resource, resourceID, nil, c.IP())
		return nil
	}
}

// parseUintParam tries to parse a decimal string into a uint.
func parseUintParam(s string, out *uint) error {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*out = uint(v)
	return nil
}
