package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
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
		c.Locals("must_change_password", claims.MustChangePassword)

		return c.Next()
	}
}

// MustChangePasswordGuard blocks access when the admin must change their password.
// Only allows /auth/change-password, /auth/refresh, /auth/logout, and /me endpoints.
func MustChangePasswordGuard() fiber.Handler {
	return func(c fiber.Ctx) error {
		mustChange, ok := c.Locals("must_change_password").(bool)
		if !ok || !mustChange {
			return c.Next()
		}

		path := c.Path()
		allowedSuffixes := []string{
			"/auth/change-password",
			"/auth/refresh",
			"/auth/logout",
			"/me",
		}
		for _, suffix := range allowedSuffixes {
			if strings.HasSuffix(path, suffix) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":             "Password change required",
			"must_change_password": true,
		})
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
