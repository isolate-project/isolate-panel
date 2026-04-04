package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/version"
)

// SetupRoutes registers all application routes on the Fiber app.
func SetupRoutes(fiberApp *fiber.App, a *App) {
	// Swagger UI (before SPA middleware so /api/docs isn't caught by it)
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

	// API group
	apiGrp := fiberApp.Group("/api")

	// Public API info
	apiGrp.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Isolate Panel API",
			"version": version.Version,
			"docs":    "/api/docs",
		})
	})

	// Auth routes (public)
	authGrp := apiGrp.Group("/auth")
	authGrp.Post("/login", middleware.LoginRateLimiter(a.LoginRL), a.AuthH.Login)
	authGrp.Post("/refresh", a.AuthH.Refresh)
	authGrp.Post("/logout", a.AuthH.Logout)

	// TOTP routes (protected)
	totpGrp := apiGrp.Group("/auth/totp", middleware.AuthMiddleware(a.TokenSvc))
	totpGrp.Get("/status", a.AuthH.TOTPStatus)
	totpGrp.Post("/setup", a.AuthH.TOTPSetup)
	totpGrp.Post("/verify", a.AuthH.TOTPVerify)
	totpGrp.Post("/disable", a.AuthH.TOTPDisable)

	// Protected routes (JWT required + standard rate limit)
	protected := apiGrp.Group("/",
		middleware.AuthMiddleware(a.TokenSvc),
		middleware.AuthRateLimiter(a.ProtectedRL),
	)
	protected.Get("/me", a.AuthH.Me)

	// Cores
	coresGrp := protected.Group("/cores")
	coresGrp.Get("/", a.CoresH.ListCores)
	coresGrp.Get("/:name", a.CoresH.GetCore)
	coresGrp.Post("/:name/start", middleware.AuthRateLimiter(a.HeavyRL), middleware.AuditAction(a.Audit, "core.start", "core"), a.CoresH.StartCore)
	coresGrp.Post("/:name/stop", middleware.AuthRateLimiter(a.HeavyRL), middleware.AuditAction(a.Audit, "core.stop", "core"), a.CoresH.StopCore)
	coresGrp.Post("/:name/restart", middleware.AuthRateLimiter(a.HeavyRL), middleware.AuditAction(a.Audit, "core.restart", "core"), a.CoresH.RestartCore)
	coresGrp.Get("/:name/status", a.CoresH.GetCoreStatus)

	// Users
	usersGrp := protected.Group("/users")
	usersGrp.Get("/", a.UsersH.ListUsers)
	usersGrp.Post("/", middleware.AuditAction(a.Audit, "user.create", "user"), a.UsersH.CreateUser)
	usersGrp.Get("/:id", a.UsersH.GetUser)
	usersGrp.Put("/:id", middleware.AuditAction(a.Audit, "user.update", "user"), a.UsersH.UpdateUser)
	usersGrp.Delete("/:id", middleware.AuditAction(a.Audit, "user.delete", "user"), a.UsersH.DeleteUser)
	usersGrp.Post("/:id/regenerate", middleware.AuditAction(a.Audit, "user.regenerate", "user"), a.UsersH.RegenerateCredentials)
	usersGrp.Get("/:id/inbounds", a.UsersH.GetUserInbounds)

	// Protocols
	protocolsGrp := protected.Group("/protocols")
	protocolsGrp.Get("/", a.ProtocolsH.ListProtocols)
	protocolsGrp.Get("/:name", a.ProtocolsH.GetProtocol)
	protocolsGrp.Get("/:name/defaults", a.ProtocolsH.GetProtocolDefaults)

	// Inbounds
	inboundsGrp := protected.Group("/inbounds")
	inboundsGrp.Get("/", a.InboundsH.ListInbounds)
	inboundsGrp.Post("/", a.InboundsH.CreateInbound)
	inboundsGrp.Get("/:id", a.InboundsH.GetInbound)
	inboundsGrp.Put("/:id", a.InboundsH.UpdateInbound)
	inboundsGrp.Delete("/:id", a.InboundsH.DeleteInbound)
	inboundsGrp.Get("/core/:core_id", a.InboundsH.GetInboundsByCore)
	inboundsGrp.Post("/assign", a.InboundsH.AssignInboundToUser)
	inboundsGrp.Post("/unassign", a.InboundsH.UnassignInboundFromUser)
	inboundsGrp.Get("/:id/users", a.InboundsH.GetInboundUsers)
	inboundsGrp.Post("/:id/users/bulk", a.InboundsH.BulkAssignUsers)
	inboundsGrp.Get("/check-port", a.InboundsH.CheckPort)

	// Outbounds
	outboundsGrp := protected.Group("/outbounds")
	outboundsGrp.Get("/", a.OutboundsH.ListOutbounds)
	outboundsGrp.Post("/", a.OutboundsH.CreateOutbound)
	outboundsGrp.Get("/:id", a.OutboundsH.GetOutbound)
	outboundsGrp.Put("/:id", a.OutboundsH.UpdateOutbound)
	outboundsGrp.Delete("/:id", a.OutboundsH.DeleteOutbound)

	// Subscription management (admin)
	protected.Get("/subscriptions/:user_id/short-url", a.SubscriptionsH.GetUserShortURL)
	protected.Get("/users/:id/subscription/stats", a.SubscriptionsH.GetAccessStats)
	protected.Post("/users/:id/subscription/regenerate", a.SubscriptionsH.RegenerateToken)

	// Certificates
	certsGrp := protected.Group("/certificates")
	certsGrp.Get("/", a.CertificatesH.ListCertificates)
	certsGrp.Get("/dropdown", a.CertificatesH.ListCertificatesDropdown)
	certsGrp.Post("/", middleware.AuthRateLimiter(a.HeavyRL), a.CertificatesH.RequestCertificate)
	certsGrp.Post("/upload", middleware.AuthRateLimiter(a.HeavyRL), a.CertificatesH.UploadCertificate)
	certsGrp.Get("/:id", a.CertificatesH.GetCertificate)
	certsGrp.Post("/:id/renew", middleware.AuthRateLimiter(a.HeavyRL), a.CertificatesH.RenewCertificate)
	certsGrp.Post("/:id/revoke", middleware.AuthRateLimiter(a.HeavyRL), a.CertificatesH.RevokeCertificate)
	certsGrp.Delete("/:id", a.CertificatesH.DeleteCertificate)

	// Stats and monitoring
	statsGrp := protected.Group("/stats")
	statsGrp.Get("/dashboard", a.StatsH.GetDashboardStats)
	statsGrp.Get("/user/:user_id/traffic", a.StatsH.GetUserTrafficStats)
	statsGrp.Get("/connections", a.StatsH.GetActiveConnections)
	statsGrp.Post("/user/:user_id/disconnect", a.StatsH.DisconnectUser)
	statsGrp.Post("/user/:user_id/kick", a.StatsH.KickUser)
	statsGrp.Get("/traffic/overview", a.StatsH.GetTrafficOverview)
	statsGrp.Get("/traffic/top-users", a.StatsH.GetTopUsers)

	// WARP, Backup, Notifications (handlers register their own sub-routes)
	a.WarpH.RegisterRoutes(protected)
	a.BackupH.RegisterRoutes(protected)
	a.NotificationsH.RegisterRoutes(protected)

	// Audit logs (super-admin only)
	protected.Get("/audit-logs", middleware.RequireSuperAdmin(), a.AuditH.ListAuditLogs)

	// Settings
	settingsGrp := protected.Group("/settings")
	settingsGrp.Get("/monitoring", a.SettingsH.GetMonitoring)
	settingsGrp.Put("/monitoring", a.SettingsH.UpdateMonitoring)
	settingsGrp.Get("/traffic-reset", a.SettingsH.GetTrafficResetSchedule)
	settingsGrp.Put("/traffic-reset", a.SettingsH.UpdateTrafficResetSchedule)
	settingsGrp.Get("/", a.SettingsH.GetAllSettings)
	settingsGrp.Put("/", a.SettingsH.UpdateSettings)

	// WebSocket ticket endpoint (protected — issues one-time ticket for WS auth)
	protected.Post("/ws/ticket", a.DashboardHub.IssueWSTicket)

	// WebSocket routes (auth via ?ticket= one-time token, fallback to ?token= for compat)
	apiGrp.Get("/ws/dashboard", a.DashboardHub.DashboardWS)

	// Subscription public routes (rate limited, token-based auth)
	subsGrp := fiberApp.Group("", middleware.SubscriptionRateLimiter())
	subsGrp.Get("/sub/:token", a.SubscriptionsH.GetAutoDetectSubscription)
	subsGrp.Get("/sub/:token/clash", a.SubscriptionsH.GetClashSubscription)
	subsGrp.Get("/sub/:token/singbox", a.SubscriptionsH.GetSingboxSubscription)
	subsGrp.Get("/sub/:token/qr", a.SubscriptionsH.GetQRCode)
	subsGrp.Get("/s/:code", a.SubscriptionsH.RedirectShortURL)
}
