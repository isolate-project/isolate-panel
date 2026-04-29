package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
)

// AuthMiddleware validates authentication via BFF session cookie first,
// then falls back to Authorization header for legacy/mobile clients.
func AuthMiddleware(tokenService *auth.TokenService, sessionManager *auth.BFFSessionManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		sessionID := c.Cookies("__Host-session")
		if sessionID != "" {
			session := sessionManager.GetSession(sessionID)
			if session != nil {
				claims, err := tokenService.ValidateAccessToken(session.AccessToken)
				if err == nil {
				c.Locals("admin_id", claims.AdminID)
				c.Locals("username", claims.Username)
				c.Locals("is_super_admin", claims.IsSuperAdmin)
				c.Locals("must_change_password", claims.MustChangePassword)
				c.Locals("session_id", session.SessionID)
				c.Locals("permissions", permissionsFromClaims(claims))
				return c.Next()
				}

				if time.Now().After(session.AccessExpiry) {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Session expired",
					})
				}
			}
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		claims, err := tokenService.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		c.Locals("admin_id", claims.AdminID)
		c.Locals("username", claims.Username)
		c.Locals("is_super_admin", claims.IsSuperAdmin)
		c.Locals("must_change_password", claims.MustChangePassword)
		c.Locals("permissions", permissionsFromClaims(claims))

		return c.Next()
	}
}

func permissionsFromClaims(claims *auth.Claims) auth.Permissions {
	if claims.Permissions != 0 {
		return auth.Permissions(claims.Permissions)
	}
	if claims.IsSuperAdmin {
		return auth.NewPermissions(
			auth.PermViewDashboard,
			auth.PermManageUsers,
			auth.PermManageInbounds,
			auth.PermManageOutbounds,
			auth.PermManageCores,
			auth.PermManageSettings,
			auth.PermViewLogs,
			auth.PermManageCertificates,
			auth.PermManageBackups,
			auth.PermSuperAdmin,
			auth.PermManageWarp,
			auth.PermManageGeo,
			auth.PermManageNotifications,
		)
	}
	return auth.NewPermissions(auth.PermViewDashboard)
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
			"/auth/session/refresh",
			"/auth/session/logout",
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
