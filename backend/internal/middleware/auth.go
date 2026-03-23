package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/vovk4morkovk4/isolate-panel/internal/auth"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(tokenService *auth.TokenService) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		// Validate token
		claims, err := tokenService.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store claims in context
		c.Locals("admin_id", claims.AdminID)
		c.Locals("username", claims.Username)
		c.Locals("is_super_admin", claims.IsSuperAdmin)

		return c.Next()
	}
}

// RequireSuperAdmin checks if the authenticated user is a super admin
func RequireSuperAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		isSuperAdmin, ok := c.Locals("is_super_admin").(bool)
		if !ok || !isSuperAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Super admin access required",
			})
		}
		return c.Next()
	}
}
