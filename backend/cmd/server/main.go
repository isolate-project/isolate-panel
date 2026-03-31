package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm/logger"

	"github.com/vovk4morkovk4/isolate-panel/internal/api"
	"github.com/vovk4morkovk4/isolate-panel/internal/auth"
	"github.com/vovk4morkovk4/isolate-panel/internal/cache"
	appconfig "github.com/vovk4morkovk4/isolate-panel/internal/config"
	"github.com/vovk4morkovk4/isolate-panel/internal/cores"
	"github.com/vovk4morkovk4/isolate-panel/internal/database"
	"github.com/vovk4morkovk4/isolate-panel/internal/database/seeds"
	applogger "github.com/vovk4morkovk4/isolate-panel/internal/logger"
	"github.com/vovk4morkovk4/isolate-panel/internal/middleware"
	"github.com/vovk4morkovk4/isolate-panel/internal/scheduler"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := appconfig.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := applogger.Init(&applogger.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		Output:     cfg.Logging.Output,
		FilePath:   cfg.Logging.FilePath,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log := applogger.Log
	log.Info().Msg("Starting Isolate Panel")
	log.Info().Str("env", cfg.App.Env).Msg("Environment")

	// Validate configuration (only in production)
	if cfg.IsProduction() {
		if err := cfg.Validate(); err != nil {
			log.Fatal().Err(err).Msg("Configuration validation failed")
		}
	} else {
		log.Warn().Msg("Running in development mode - some validations are skipped")
		if cfg.JWT.Secret == "change-this-in-production-use-env-var" {
			log.Warn().Msg("Using default JWT secret - set JWT_SECRET environment variable in production!")
		}
	}

	// Initialize database
	db, err := database.New(&database.Config{
		Path:         cfg.Database.Path,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		LogLevel:     logger.Info,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	// Note: Don't close database - it needs to stay open for the server lifetime

	// Run migrations (commented out - run manually with ./bin/migrate)
	// Migration manager closes the database connection, so we can't use it here
	// log.Info().Msg("Running database migrations")
	// if err := db.RunMigrations(); err != nil {
	// 	log.Fatal().Err(err).Msg("Failed to run migrations")
	// }
	// log.Info().Msg("Migrations completed")

	// Run seeders in development
	// Note: Seeders use the same database connection, so they work fine
	if cfg.IsDevelopment() {
		log.Info().Msg("Running database seeders")
		seeder := seeds.NewSeeder(db.DB)
		if err := seeder.RunAll(os.Getenv("ADMIN_PASSWORD")); err != nil {
			log.Fatal().Err(err).Msg("Failed to run seeders")
		}
	}

	// Initialize Cache Manager
	cacheManager, err := cache.NewCacheManager()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize cache manager")
	}

	// Initialize Core Manager
	coreManager := cores.NewCoreManager(db.DB, cfg.Cores.SupervisorURL)

	// Initialize Core Lifecycle Manager (lazy loading)
	lifecycleManager := services.NewCoreLifecycleManager(db.DB, coreManager)

	// Initialize Config Service
	configService := services.NewConfigService(db.DB, coreManager, cfg.Cores.ConfigDir, cacheManager)

	// Connect ConfigService to LifecycleManager
	lifecycleManager.SetConfigService(configService)

	// Initialize cores (lazy loading - only start if needed)
	log.Info().Msg("Initializing cores (lazy loading)")
	if err := lifecycleManager.InitializeCores(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize cores - cores can be started manually")
	}

	// Initialize JWT token service
	tokenService := auth.NewTokenService(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTokenTTL)*time.Second,
		time.Duration(cfg.JWT.RefreshTokenTTL)*time.Second,
	)

	// Initialize Notification service (before other services that depend on it)
	notificationService := services.NewNotificationService(db.DB, "", "", "", "")
	if err := notificationService.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Notification service")
	}

	// Initialize port manager
	portManager := services.NewPortManager(db.DB)

	// Initialize services
	settingsService := services.NewSettingsService(db.DB, cacheManager)
	userService := services.NewUserService(db.DB, notificationService, cacheManager)
	inboundService := services.NewInboundService(db.DB, lifecycleManager, portManager)
	outboundService := services.NewOutboundService(db.DB, configService)
	subscriptionService := services.NewSubscriptionService(db.DB, "", cacheManager)

	// Initialize traffic collector (interval based on monitoring_mode setting)
	trafficCollector := services.NewTrafficCollector(
		db.DB,
		settingsService,
		0, // auto-detect interval from settings
		cfg.Cores.XrayAPIAddr,
		cfg.Cores.SingboxAPIAddr,
		cfg.Cores.MihomoAPIAddr,
		cfg.Cores.SingboxAPIKey,
		cfg.Cores.MihomoAPIKey,
	)
	trafficCollector.Start()

	// Initialize connection tracker (default 10s interval for real-time feel)
	connectionTracker := services.NewConnectionTracker(
		db.DB,
		0, // auto-detect interval
		cfg.Cores.XrayAPIAddr,
		cfg.Cores.SingboxAPIAddr,
		cfg.Cores.MihomoAPIAddr,
		cfg.Cores.SingboxAPIKey,
		cfg.Cores.MihomoAPIKey,
	)
	connectionTracker.Start()

	// Initialize quota enforcer (for automatic quota enforcement)
	quotaEnforcer := services.NewQuotaEnforcer(db.DB, configService, notificationService)

	// Initialize data aggregator (hourly and daily aggregation)
	dataAggregator := services.NewDataAggregator(db.DB, 0)
	dataAggregator.Start()

	// Initialize data retention service (cleanup old data)
	dataRetention := services.NewDataRetentionService(db.DB, 0, settingsService)
	dataRetention.Start()

	// Initialize WARP service
	warpService := services.NewWARPService(db.DB, "/app/data/warp")
	if err := warpService.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize WARP service")
	}

	// Initialize Geo service
	geoService := services.NewGeoService(db.DB, "/app/data/geo")
	if err := geoService.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Geo service")
	}

	// Initialize Backup service
	backupService := services.NewBackupService(db.DB, "/app/data/backups", "/app/data")
	if err := backupService.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Backup service")
	}

	// Initialize Backup Scheduler
	backupScheduler := scheduler.NewBackupScheduler(db.DB, backupService)
	if err := backupScheduler.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Backup Scheduler")
	}

	// Start quota enforcement loop (check every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			quotaEnforcer.CheckAndEnforce(context.Background())
			// Also check for expiring users
			userService.CheckExpiringUsers()
		}
	}()

	// Build Cloudflare credentials (only include non-empty values)
	cfCredentials := make(map[string]string)
	if v := os.Getenv("CLOUDFLARE_API_KEY"); v != "" {
		cfCredentials["api_key"] = v
	}
	if v := os.Getenv("CLOUDFLARE_EMAIL"); v != "" {
		cfCredentials["email"] = v
	}
	if v := os.Getenv("CLOUDFLARE_API_TOKEN"); v != "" {
		cfCredentials["api_token"] = v
	}

	// Determine DNS provider (only set if credentials are present)
	dnsProvider := ""
	if len(cfCredentials) > 0 {
		dnsProvider = "cloudflare"
	}

	// Initialize certificate service (with optional Cloudflare DNS-01 challenge)
	certService, err := services.NewCertificateService(db.DB, services.CertificateServiceConfig{
		CertDir:     "/etc/isolate-panel/certs",
		Email:       cfg.App.AdminEmail, // From config
		DNSProvider: dnsProvider,
		Credentials: cfCredentials,
		Staging:     cfg.App.Env == "development", // Use staging in dev
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize certificate service - ACME features disabled")
	}

	// Set notification service for certificate and core lifecycle (if available)
	if notificationService != nil {
		if certService != nil {
			certService.SetNotificationService(notificationService)
		}
		lifecycleManager.SetNotificationService(notificationService)
	}

	// Initialize handlers
	authHandler := api.NewAuthHandler(db.DB, tokenService, notificationService)
	coresHandler := api.NewCoresHandler(coreManager)
	usersHandler := api.NewUsersHandler(userService)
	inboundsHandler := api.NewInboundsHandler(inboundService, portManager)
	outboundsHandler := api.NewOutboundsHandler(outboundService)
	protocolsHandler := api.NewProtocolsHandler()
	subscriptionsHandler := api.NewSubscriptionsHandler(subscriptionService)
	certificatesHandler := api.NewCertificatesHandler(certService, db.DB)
	statsHandler := api.NewStatsHandler(db.DB, trafficCollector, connectionTracker)
	warpHandler := api.NewWarpHandler(warpService, geoService)
	backupHandler := api.NewBackupHandler(backupService, backupScheduler)
	notificationHandler := api.NewNotificationHandler(notificationService)
	settingsHandler := api.NewSettingsHandler(settingsService, trafficCollector)

	// Initialize rate limiter for login (5 attempts per minute per IP)
	loginLimiter := middleware.NewRateLimiter(5, 1*time.Minute)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      fmt.Sprintf("%s v0.1.0", cfg.App.Name),
		ErrorHandler: middleware.ErrorHandler,
	})

	// Global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.CORS())
	app.Use(middleware.RequestLogger())

	// Health check endpoint (must be before static middleware)
	var startTime = time.Now()
	app.Get("/health", func(c fiber.Ctx) error {
		type HealthResponse struct {
			Status    string `json:"status"` // healthy, unhealthy
			Version   string `json:"version"`
			Uptime    string `json:"uptime"`
			Database  string `json:"database"` // connected, disconnected
			Timestamp string `json:"timestamp"`
		}

		response := HealthResponse{
			Status:    "healthy",
			Version:   "0.1.0",
			Uptime:    time.Since(startTime).String(),
			Timestamp: time.Now().Format(time.RFC3339),
			Database:  "connected",
		}

		// Check database connection
		sqlDB, err := db.DB.DB()
		if err != nil {
			response.Database = "disconnected"
			response.Status = "unhealthy"
		} else if err := sqlDB.Ping(); err != nil {
			response.Database = "disconnected"
			response.Status = "unhealthy"
		}

		statusCode := fiber.StatusOK
		if response.Status == "unhealthy" {
			statusCode = fiber.StatusServiceUnavailable
		}

		return c.Status(statusCode).JSON(response)
	})

	// Serve static files from /var/www/html (must be after /health but before /api)
	app.Use(func(c fiber.Ctx) error {
		// Skip API routes and subscription routes
		if strings.HasPrefix(c.Path(), "/api") || strings.HasPrefix(c.Path(), "/sub/") || strings.HasPrefix(c.Path(), "/s/") {
			return c.Next()
		}

		// Try to serve static file
		filePath := "/var/www/html" + c.Path()
		if c.Path() == "/" {
			filePath = "/var/www/html/index.html"
		}

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			return c.SendFile(filePath)
		}

		// For SPA - return index.html for unknown routes
		if _, err := os.Stat("/var/www/html/index.html"); err == nil {
			return c.SendFile("/var/www/html/index.html")
		}

		return c.Next()
	})

	// API routes
	apiGroup := app.Group("/api")

	// Public routes (no auth required)
	apiGroup.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Isolate Panel API",
			"version": "0.1.0",
			"docs":    "/api/docs",
		})
	})

	// Auth routes
	authGroup := apiGroup.Group("/auth")
	authGroup.Post("/login", middleware.LoginRateLimiter(loginLimiter), authHandler.Login)
	authGroup.Post("/refresh", authHandler.Refresh)
	authGroup.Post("/logout", authHandler.Logout)

	// Protected routes (auth required)
	protectedGroup := apiGroup.Group("/", middleware.AuthMiddleware(tokenService))
	protectedGroup.Get("/me", authHandler.Me)

	// Core management routes (protected)
	coresGroup := protectedGroup.Group("/cores")
	coresGroup.Get("/", coresHandler.ListCores)
	coresGroup.Get("/:name", coresHandler.GetCore)
	coresGroup.Post("/:name/start", coresHandler.StartCore)
	coresGroup.Post("/:name/stop", coresHandler.StopCore)
	coresGroup.Post("/:name/restart", coresHandler.RestartCore)
	coresGroup.Get("/:name/status", coresHandler.GetCoreStatus)

	// User management routes (protected)
	usersGroup := protectedGroup.Group("/users")
	usersGroup.Get("/", usersHandler.ListUsers)
	usersGroup.Post("/", usersHandler.CreateUser)
	usersGroup.Get("/:id", usersHandler.GetUser)
	usersGroup.Put("/:id", usersHandler.UpdateUser)
	usersGroup.Delete("/:id", usersHandler.DeleteUser)
	usersGroup.Post("/:id/regenerate", usersHandler.RegenerateCredentials)
	usersGroup.Get("/:id/inbounds", usersHandler.GetUserInbounds)

	// Protocol registry routes (protected)
	protocolsGroup := protectedGroup.Group("/protocols")
	protocolsGroup.Get("/", protocolsHandler.ListProtocols)
	protocolsGroup.Get("/:name", protocolsHandler.GetProtocol)
	protocolsGroup.Get("/:name/defaults", protocolsHandler.GetProtocolDefaults)

	// Inbound management routes (protected)
	inboundsGroup := protectedGroup.Group("/inbounds")
	inboundsGroup.Get("/", inboundsHandler.ListInbounds)
	inboundsGroup.Post("/", inboundsHandler.CreateInbound)
	inboundsGroup.Get("/:id", inboundsHandler.GetInbound)
	inboundsGroup.Put("/:id", inboundsHandler.UpdateInbound)
	inboundsGroup.Delete("/:id", inboundsHandler.DeleteInbound)
	inboundsGroup.Get("/core/:core_id", inboundsHandler.GetInboundsByCore)
	inboundsGroup.Post("/assign", inboundsHandler.AssignInboundToUser)
	inboundsGroup.Post("/unassign", inboundsHandler.UnassignInboundFromUser)
	inboundsGroup.Get("/:id/users", inboundsHandler.GetInboundUsers)
	inboundsGroup.Post("/:id/users/bulk", inboundsHandler.BulkAssignUsers)
	inboundsGroup.Get("/check-port", inboundsHandler.CheckPort)

	// Outbound management routes (protected)
	outboundsGroup := protectedGroup.Group("/outbounds")
	outboundsGroup.Get("/", outboundsHandler.ListOutbounds)
	outboundsGroup.Post("/", outboundsHandler.CreateOutbound)
	outboundsGroup.Get("/:id", outboundsHandler.GetOutbound)
	outboundsGroup.Put("/:id", outboundsHandler.UpdateOutbound)
	outboundsGroup.Delete("/:id", outboundsHandler.DeleteOutbound)

	// Subscription short URL management (protected, admin)
	protectedGroup.Get("/subscriptions/:user_id/short-url", subscriptionsHandler.GetUserShortURL)
	protectedGroup.Get("/users/:id/subscription/stats", subscriptionsHandler.GetAccessStats)
	protectedGroup.Post("/users/:id/subscription/regenerate", subscriptionsHandler.RegenerateToken)

	// Certificate management routes (protected)
	certsGroup := protectedGroup.Group("/certificates")
	certsGroup.Get("/", certificatesHandler.ListCertificates)
	certsGroup.Get("/dropdown", certificatesHandler.ListCertificatesDropdown)
	certsGroup.Post("/", certificatesHandler.RequestCertificate)
	certsGroup.Post("/upload", certificatesHandler.UploadCertificate)
	certsGroup.Get("/:id", certificatesHandler.GetCertificate)
	certsGroup.Post("/:id/renew", certificatesHandler.RenewCertificate)
	certsGroup.Post("/:id/revoke", certificatesHandler.RevokeCertificate)
	certsGroup.Delete("/:id", certificatesHandler.DeleteCertificate)

	// Stats and monitoring routes (protected)
	statsGroup := protectedGroup.Group("/stats")
	statsGroup.Get("/dashboard", statsHandler.GetDashboardStats)
	statsGroup.Get("/user/:user_id/traffic", statsHandler.GetUserTrafficStats)
	statsGroup.Get("/connections", statsHandler.GetActiveConnections)
	statsGroup.Post("/user/:user_id/disconnect", statsHandler.DisconnectUser)
	statsGroup.Post("/user/:user_id/kick", statsHandler.KickUser)
	statsGroup.Get("/traffic/overview", statsHandler.GetTrafficOverview)
	statsGroup.Get("/traffic/top-users", statsHandler.GetTopUsers)

	// WARP and Geo routes (protected)
	warpHandler.RegisterRoutes(protectedGroup)

	// Backup routes (protected)
	backupHandler.RegisterRoutes(protectedGroup)

	// Notification routes (protected)
	notificationHandler.RegisterRoutes(protectedGroup)

	// Settings routes (protected)
	settingsGroup := protectedGroup.Group("/settings")
	settingsGroup.Get("/monitoring", settingsHandler.GetMonitoring)
	settingsGroup.Put("/monitoring", settingsHandler.UpdateMonitoring)
	settingsGroup.Get("/", settingsHandler.GetAllSettings)
	settingsGroup.Put("/", settingsHandler.UpdateSettings)

	// Subscription routes (public, token-based auth, rate limited)
	subscriptionRoutes := app.Group("", middleware.SubscriptionRateLimiter())
	subscriptionRoutes.Get("/sub/:token", subscriptionsHandler.GetAutoDetectSubscription)
	subscriptionRoutes.Get("/sub/:token/clash", subscriptionsHandler.GetClashSubscription)
	subscriptionRoutes.Get("/sub/:token/singbox", subscriptionsHandler.GetSingboxSubscription)
	subscriptionRoutes.Get("/sub/:token/qr", subscriptionsHandler.GetQRCode)
	subscriptionRoutes.Get("/s/:code", subscriptionsHandler.RedirectShortURL)

	// Log startup information
	log.Info().
		Str("host", cfg.App.Host).
		Int("port", cfg.App.Port).Str("env_port", os.Getenv("PORT")).
		Msg("Starting HTTP server")

	log.Info().Msg("✓ Authentication system enabled")
	log.Info().Msg("✓ Core management enabled")
	log.Info().Msg("✓ User management enabled")
	log.Info().Msg("✓ Inbound management enabled")
	log.Info().Msg("✓ Outbound management enabled")
	log.Info().Msg("✓ Protocol registry enabled")
	log.Info().Msg("✓ Subscription service enabled")
	log.Info().Msg("✓ Structured logging enabled")
	log.Info().Msg("✓ Configuration management enabled")
	log.Info().Msg("✓ Settings management enabled")
	log.Info().Msg("✓ Backup system enabled")

	// Start server in a separate goroutine
	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	go func() {
		if err := app.Listen(addr); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Gracefully shutting down server...")

	// 1. Stop processing new requests
	log.Info().Msg("Stopping HTTP server...")
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	// 2. Stop background services
	log.Info().Msg("Stopping background services...")
	dataAggregator.Stop()
	dataRetention.Stop()
	connectionTracker.Stop()
	trafficCollector.Stop()
	backupScheduler.Stop()
	if certService != nil {
		certService.Stop()
	}
	if cacheManager != nil {
		cacheManager.Close()
	}

	// 3. Close database connection
	log.Info().Msg("Closing database connection...")
	if sqlDB, err := db.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database securely")
		}
	}

	log.Info().Msg("Server stopped securely")
}
