package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
)

func RequirePermission(perm auth.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		perms, ok := c.Locals("permissions").(auth.Permissions)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing permissions"})
		}
		if !perms.Has(perm) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fmt.Sprintf("permission %s required", perm),
			})
		}
		return c.Next()
	}
}

func RequireAnyPermission(perms ...auth.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := c.Locals("permissions").(auth.Permissions)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing permissions"})
		}
		if !p.HasAny(perms...) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "required permission missing",
			})
		}
		return c.Next()
	}
}

func RequireAllPermissions(perms ...auth.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := c.Locals("permissions").(auth.Permissions)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing permissions"})
		}
		if !p.HasAll(perms...) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "all required permissions missing",
			})
		}
		return c.Next()
	}
}
