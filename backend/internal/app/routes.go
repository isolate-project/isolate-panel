package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/version"
)

// SetupRoutes registers all application routes on the Fiber app.
func SetupRoutes(fiberApp *fiber.App, a *App) {
	// Swagger UI — only enabled in development
	if os.Getenv("APP_ENV") == "development" {
		fiberApp.Get("/api/docs", func(c fiber.Ctx) error {
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.SendString(`<!DOCTYPE html>
<html>
<head>
  <title>Isolate Panel API</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url: "/api/docs/swagger.json",
  dom_id: "#swagger-ui",
  deepLinking: true,
  presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
  layout: "StandaloneLayout"
})
</script>
</body>
</html>`)
		})
		fiberApp.Get("/api/docs/swagger.json", func(c fiber.Ctx) error {
			return c.SendFile("docs/swagger/swagger.json")
		})
	}

	// Health check (must be before SPA static middleware)
	fiberApp.Get("/health", func(c fiber.Ctx) error {
		type HealthResponse struct {
			Status    string `json:"status"`
			Version   string `json:"version"`
			Uptime    string `json:"uptime"`
			Database  string `json:"database"`
			Timestamp string `json:"timestamp"`
		}
		resp := HealthResponse{
			Status:    "healthy",
			Version:   version.Version,
			Uptime:    time.Since(a.StartTime).String(),
			Timestamp: time.Now().Format(time.RFC3339),
			Database:  "connected",
		}
		sqlDB, err := a.gormDB.DB()
		if err != nil || sqlDB.Ping() != nil {
			resp.Database = "disconnected"
			resp.Status = "unhealthy"
		}
		code := fiber.StatusOK
		if resp.Status == "unhealthy" {
			code = fiber.StatusServiceUnavailable
		}
		return c.Status(code).JSON(resp)
	})

	// SPA static file serving (must be before /api, after /health)
	fiberApp.Use(func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") ||
			strings.HasPrefix(c.Path(), "/sub/") ||
			strings.HasPrefix(c.Path(), "/s/") {
			return c.Next()
		}
		reqPath := c.Path()
		if reqPath == "/" {
			reqPath = "/index.html"
		}
		filePath := filepath.Join("/var/www/html", filepath.Clean(reqPath))
		if !strings.HasPrefix(filePath, "/var/www/html/") {
			return c.Next() // path traversal attempt
		}
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			return c.SendFile(filePath)
		}
		if _, err := os.Stat("/var/www/html/index.html"); err == nil {
			return c.SendFile("/var/www/html/index.html")
		}
		return c.Next()
	})

	// Register API routes under both /api (backward compatible) and /api/v1 (versioned)
	registerV1Routes(fiberApp.Group("/api"), a)
	registerV1Routes(fiberApp.Group("/api/v1"), a)

	// Subscription public routes (rate limited, token-based auth; H-29: uses WithStop for cleanup)
	subsBundle := middleware.SubscriptionRateLimiterWithStop()
	a.SubTokenRL = subsBundle.TokenLimiter
	a.SubIPRL = subsBundle.IPLimiter
	subsGrp := fiberApp.Group("", subsBundle.Handler)
	subsGrp.Get("/sub/:token", middleware.SubscriptionSignatureValidator(a.SubscriptionSigner), a.SubscriptionsH.GetAutoDetectSubscription)
	subsGrp.Get("/sub/:token/clash", middleware.SubscriptionSignatureValidator(a.SubscriptionSigner), a.SubscriptionsH.GetClashSubscription)
	subsGrp.Get("/sub/:token/singbox", middleware.SubscriptionSignatureValidator(a.SubscriptionSigner), a.SubscriptionsH.GetSingboxSubscription)
	subsGrp.Get("/sub/:token/isolate", middleware.SubscriptionSignatureValidator(a.SubscriptionSigner), a.SubscriptionsH.GetIsolateSubscription)
	subsGrp.Get("/sub/:token/qr", middleware.SubscriptionSignatureValidator(a.SubscriptionSigner), a.SubscriptionsH.GetQRCode)
	subsGrp.Get("/s/:code", a.SubscriptionsH.RedirectShortURL)
}

// registerV1Routes registers all v1 API routes on the given router
func registerV1Routes(router fiber.Router, a *App) {
	// Public API info
	router.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Isolate Panel API",
			"version": version.Version,
			"docs":    "/api/docs",
		})
	})

	router.Get("/.well-known/jwks.json", func(c fiber.Ctx) error {
		return c.JSON(a.TokenSvc.GetJWKS())
	})

	authGrp := router.Group("/auth")
	authGrp.Post("/login", middleware.LoginRateLimiter(a.LoginRL), a.AuthH.Login)
	authGrp.Post("/refresh", middleware.LoginRateLimiter(a.RefreshLogoutRL), a.AuthH.Refresh)
	authGrp.Post("/logout", middleware.LoginRateLimiter(a.RefreshLogoutRL), a.AuthH.Logout)

	authGrp.Post("/session/login", middleware.LoginRateLimiter(a.LoginRL), a.AuthH.SessionLogin)
	authGrp.Post("/session/logout", middleware.LoginRateLimiter(a.RefreshLogoutRL), a.AuthH.SessionLogout)
	authGrp.Post("/session/refresh", middleware.LoginRateLimiter(a.RefreshLogoutRL), a.AuthH.SessionRefresh)

	// WebAuthn authentication (public - for login)
	authGrp.Post("/webauthn/authenticate/begin", middleware.LoginRateLimiter(a.LoginRL), a.AuthH.WebAuthnAuthenticateBegin)
	authGrp.Post("/webauthn/authenticate/finish", middleware.LoginRateLimiter(a.LoginRL), a.AuthH.WebAuthnAuthenticateFinish)

	// TOTP routes (protected)
	totpGrp := router.Group("/auth/totp",
		middleware.AuthMiddleware(a.TokenSvc, a.SessionManager),
		middleware.MustChangePasswordGuard(),
	)
	totpGrp.Get("/status", a.AuthH.TOTPStatus)
	totpGrp.Post("/setup", a.AuthH.TOTPSetup)
	totpGrp.Post("/verify", a.AuthH.TOTPVerify)
	totpGrp.Post("/disable", a.AuthH.TOTPDisable)

	// WebAuthn routes (protected - for credential management)
	webauthnGrp := router.Group("/auth/webauthn",
		middleware.AuthMiddleware(a.TokenSvc, a.SessionManager),
		middleware.MustChangePasswordGuard(),
	)
	webauthnGrp.Get("/status", a.AuthH.WebAuthnStatus)
	webauthnGrp.Get("/credentials", a.AuthH.WebAuthnListCredentials)
	webauthnGrp.Delete("/credentials/:id", a.AuthH.WebAuthnDeleteCredential)
	webauthnGrp.Post("/register/begin", a.AuthH.WebAuthnRegisterBegin)
	webauthnGrp.Post("/register/finish", a.AuthH.WebAuthnRegisterFinish)

	protected := router.Group("/",
		middleware.AuthMiddleware(a.TokenSvc, a.SessionManager),
		middleware.MustChangePasswordGuard(),
		middleware.AuthRateLimiter(a.ProtectedRL),
	)
	protected.Get("/me", a.AuthH.Me)
	protected.Post("/auth/change-password", a.AuthH.ChangePassword)

	// System
	systemGrp := protected.Group("/system")
	systemGrp.Get("/resources", middleware.RequirePermission(auth.PermViewDashboard), a.SystemH.GetResources)
	systemGrp.Get("/connections", middleware.RequirePermission(auth.PermViewDashboard), a.SystemH.GetConnections)
	systemGrp.Post("/emergency-cleanup", middleware.AuthRateLimiter(a.HeavyRL), middleware.RequirePermission(auth.PermManageCores), a.SystemH.EmergencyCleanup)

	// Cores
	coresGrp := protected.Group("/cores")
	coresGrp.Get("/", middleware.RequirePermission(auth.PermViewDashboard), a.CoresH.ListCores)
	coresGrp.Get("/:name", middleware.RequirePermission(auth.PermViewDashboard), a.CoresH.GetCore)
	coresGrp.Post("/:name/start", middleware.AuthRateLimiter(a.HeavyRL), middleware.AuditAction(a.Audit, "core.start", "core"), middleware.RequirePermission(auth.PermManageCores), a.CoresH.StartCore)
	coresGrp.Post("/:name/stop", middleware.AuthRateLimiter(a.HeavyRL), middleware.AuditAction(a.Audit, "core.stop", "core"), middleware.RequirePermission(auth.PermManageCores), a.CoresH.StopCore)
	coresGrp.Post("/:name/restart", middleware.AuthRateLimiter(a.HeavyRL), middleware.AuditAction(a.Audit, "core.restart", "core"), middleware.RequirePermission(auth.PermManageCores), a.CoresH.RestartCore)
	coresGrp.Get("/:name/status", middleware.RequirePermission(auth.PermViewDashboard), a.CoresH.GetCoreStatus)
	coresGrp.Get("/:name/logs", middleware.RequirePermission(auth.PermViewLogs), api.GetCoreLogs(a.Cores))

	// Users
	usersGrp := protected.Group("/users")
	usersGrp.Get("/", middleware.RequirePermission(auth.PermManageUsers), a.UsersH.ListUsers)
	usersGrp.Post("/", middleware.AuditAction(a.Audit, "user.create", "user"), middleware.RequirePermission(auth.PermManageUsers), a.UsersH.CreateUser)
	usersGrp.Get("/:id", middleware.RequirePermission(auth.PermManageUsers), a.UsersH.GetUser)
	usersGrp.Put("/:id", middleware.AuditAction(a.Audit, "user.update", "user"), middleware.RequirePermission(auth.PermManageUsers), a.UsersH.UpdateUser)
	usersGrp.Delete("/:id", middleware.AuditAction(a.Audit, "user.delete", "user"), middleware.RequirePermission(auth.PermManageUsers), a.UsersH.DeleteUser)
	usersGrp.Post("/:id/regenerate", middleware.AuditAction(a.Audit, "user.regenerate", "user"), middleware.RequirePermission(auth.PermManageUsers), a.UsersH.RegenerateCredentials)
	usersGrp.Get("/:id/inbounds", middleware.RequirePermission(auth.PermManageUsers), a.UsersH.GetUserInbounds)

	// Protocols
	protocolsGrp := protected.Group("/protocols")
	protocolsGrp.Get("/", middleware.RequirePermission(auth.PermViewDashboard), a.ProtocolsH.ListProtocols)
	protocolsGrp.Get("/:name", middleware.RequirePermission(auth.PermViewDashboard), a.ProtocolsH.GetProtocol)
	protocolsGrp.Get("/:name/defaults", middleware.RequirePermission(auth.PermViewDashboard), a.ProtocolsH.GetProtocolDefaults)

	// Inbounds
	inboundsGrp := protected.Group("/inbounds")
	inboundsGrp.Get("/", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.ListInbounds)
	inboundsGrp.Post("/", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.CreateInbound)
	inboundsGrp.Get("/core/:core_id", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.GetInboundsByCore)
	inboundsGrp.Get("/check-port", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.CheckPort)
	inboundsGrp.Post("/check-port", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.CheckPortAvailability)
	inboundsGrp.Post("/assign", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.AssignInboundToUser)
	inboundsGrp.Post("/unassign", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.UnassignInboundFromUser)
	inboundsGrp.Get("/:id", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.GetInbound)
	inboundsGrp.Put("/:id", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.UpdateInbound)
	inboundsGrp.Delete("/:id", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.DeleteInbound)
	inboundsGrp.Get("/:id/users", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.GetInboundUsers)
	inboundsGrp.Post("/:id/users/bulk", middleware.RequirePermission(auth.PermManageInbounds), a.InboundsH.BulkAssignUsers)

	// Outbounds
	outboundsGrp := protected.Group("/outbounds")
	outboundsGrp.Get("/", middleware.RequirePermission(auth.PermManageOutbounds), a.OutboundsH.ListOutbounds)
	outboundsGrp.Post("/", middleware.RequirePermission(auth.PermManageOutbounds), a.OutboundsH.CreateOutbound)
	outboundsGrp.Get("/:id", middleware.RequirePermission(auth.PermManageOutbounds), a.OutboundsH.GetOutbound)
	outboundsGrp.Put("/:id", middleware.RequirePermission(auth.PermManageOutbounds), a.OutboundsH.UpdateOutbound)
	outboundsGrp.Delete("/:id", middleware.RequirePermission(auth.PermManageOutbounds), a.OutboundsH.DeleteOutbound)

	// Subscription management (admin)
	protected.Get("/subscriptions/:user_id/short-url", middleware.RequirePermission(auth.PermManageUsers), a.SubscriptionsH.GetUserShortURL)
	protected.Get("/users/:id/subscription/stats", middleware.RequirePermission(auth.PermManageUsers), a.SubscriptionsH.GetAccessStats)
	protected.Post("/users/:id/subscription/regenerate", middleware.RequirePermission(auth.PermManageUsers), a.SubscriptionsH.RegenerateToken)

	// Certificates
	certsGrp := protected.Group("/certificates")
	certsGrp.Get("/", middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.ListCertificates)
	certsGrp.Get("/dropdown", middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.ListCertificatesDropdown)
	certsGrp.Post("/", middleware.AuthRateLimiter(a.HeavyRL), middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.RequestCertificate)
	certsGrp.Post("/upload", middleware.AuthRateLimiter(a.HeavyRL), middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.UploadCertificate)
	certsGrp.Get("/:id", middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.GetCertificate)
	certsGrp.Post("/:id/renew", middleware.AuthRateLimiter(a.HeavyRL), middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.RenewCertificate)
	certsGrp.Post("/:id/revoke", middleware.AuthRateLimiter(a.HeavyRL), middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.RevokeCertificate)
	certsGrp.Delete("/:id", middleware.RequirePermission(auth.PermManageCertificates), a.CertificatesH.DeleteCertificate)

	// Stats and monitoring
	statsGrp := protected.Group("/stats")
	statsGrp.Get("/dashboard", middleware.RequirePermission(auth.PermViewDashboard), a.StatsH.GetDashboardStats)
	statsGrp.Get("/user/:user_id/traffic", middleware.RequirePermission(auth.PermManageUsers), a.StatsH.GetUserTrafficStats)
	statsGrp.Get("/connections", middleware.RequirePermission(auth.PermViewDashboard), a.StatsH.GetActiveConnections)
	statsGrp.Post("/user/:user_id/disconnect", middleware.RequirePermission(auth.PermManageUsers), a.StatsH.DisconnectUser)
	statsGrp.Post("/user/:user_id/kick", middleware.RequirePermission(auth.PermManageUsers), a.StatsH.KickUser)
	statsGrp.Get("/traffic/overview", middleware.RequirePermission(auth.PermViewDashboard), a.StatsH.GetTrafficOverview)
	statsGrp.Get("/traffic/top-users", middleware.RequirePermission(auth.PermViewDashboard), a.StatsH.GetTopUsers)

	// WARP, Backup, Notifications (handlers register their own sub-routes)
	a.WarpH.RegisterRoutes(protected)
	a.BackupH.RegisterRoutes(protected)
	a.NotificationsH.RegisterRoutes(protected)

	protected.Get("/audit-logs", middleware.RequirePermission(auth.PermViewLogs), a.AuditH.ListAuditLogs)

	// Settings
	settingsGrp := protected.Group("/settings")
	settingsGrp.Get("/monitoring", middleware.RequirePermission(auth.PermManageSettings), a.SettingsH.GetMonitoring)
	settingsGrp.Put("/monitoring", middleware.RequirePermission(auth.PermManageSettings), a.SettingsH.UpdateMonitoring)
	settingsGrp.Get("/traffic-reset", middleware.RequirePermission(auth.PermManageSettings), a.SettingsH.GetTrafficResetSchedule)
	settingsGrp.Put("/traffic-reset", middleware.RequirePermission(auth.PermManageSettings), a.SettingsH.UpdateTrafficResetSchedule)
	settingsGrp.Get("/", middleware.RequirePermission(auth.PermManageSettings), a.SettingsH.GetAllSettings)
	settingsGrp.Put("/", middleware.RequirePermission(auth.PermManageSettings), a.SettingsH.UpdateSettings)

	// WebSocket ticket endpoint (protected — issues one-time ticket for WS auth)
	protected.Post("/ws/ticket", a.DashboardHub.IssueWSTicket)

	// WebSocket routes (auth via ?ticket= one-time token only)
	router.Get("/ws/dashboard", a.DashboardHub.DashboardWS)
}
