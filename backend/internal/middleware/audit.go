package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/services"
)

// AuditAction returns a post-handler middleware that writes an audit entry
// for all operations (both successful and failed).
func AuditAction(auditSvc *services.AuditService, action, resource string) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		adminID, _ := c.Locals("admin_id").(uint)
		username, _ := c.Locals("username").(string)

		var resourceID *uint
		if raw := c.Params("id"); raw != "" {
			var id uint
			if parseUintParam(raw, &id) == nil {
				resourceID = &id
			}
		}

		statusCode := c.Response().StatusCode()
		details := map[string]interface{}{
			"status_code":   statusCode,
			"path":          c.Path(),
			"method":        c.Method(),
			"user_agent":    c.Get("User-Agent"),
			"duration_ms":   time.Since(start).Milliseconds(),
			"username":      username,
		}
		if err != nil {
			details["error"] = err.Error()
		}

		auditSvc.Log(adminID, action, resource, resourceID, details, c.IP())
		return err
	}
}

func parseUintParam(s string, out *uint) error {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*out = uint(v)
	return nil
}
