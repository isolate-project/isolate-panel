package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/services"
)

// AuditAction returns a post-handler middleware that writes an audit entry
// for all operations (both successful and failed).
// action and resource are constants like "user.delete" / "user".
// The resource ID is read from the "id" route param when present.
func AuditAction(auditSvc *services.AuditService, action, resource string) fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()

		adminID, _ := c.Locals("admin_id").(uint)

		var resourceID *uint
		if raw := c.Params("id"); raw != "" {
			var id uint
			if parseUintParam(raw, &id) == nil {
				resourceID = &id
			}
		}

		// Log both successful and failed operations
		statusCode := c.Response().StatusCode()
		details := map[string]interface{}{
			"status_code": statusCode,
		}
		if err != nil {
			details["error"] = err.Error()
		}

		auditSvc.Log(adminID, action, resource, resourceID, details, c.IP())
		return err
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
