package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm/logger"

	"github.com/vovk4morkovk4/isolate-panel/internal/api"
	"github.com/vovk4morkovk4/isolate-panel/internal/auth"
	appconfig "github.com/vovk4morkovk4/isolate-panel/internal/config"
	"github.com/vovk4morkovk4/isolate-panel/internal/core"
	"github.com/vovk4morkovk4/isolate-panel/internal/database"
	"github.com/vovk4morkovk4/isolate-panel/internal/database/seeds"
	applogger "github.com/vovk4morkovk4/isolate-panel/internal/logger"
	"github.com/vovk4morkovk4/isolate-panel/internal/middleware"
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
	defer db.Close()

	// Run migrations
	log.Info().Msg("Running database migrations")
	if err := db.RunMigrations(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	log.Info().Msg("Migrations completed")

	// Run seeders in development
	if cfg.IsDevelopment() {
		log.Info().Msg("Running database seeders")
		seeder := seeds.NewSeeder(db.DB)
		if err := seeder.RunAll(); err != nil {
			log.Fatal().Err(err).Msg("Failed to run seeders")
		}
	}

	// Initialize Core Manager
	coreManager := core.NewCoreManager(db.DB, cfg.Cores.SupervisorURL)

	// Initialize Core Lifecycle Manager (lazy loading)
	lifecycleManager := services.NewCoreLifecycleManager(db.DB, coreManager)

	// Initialize Config Service
	configService := services.NewConfigService(db.DB, coreManager, cfg.Cores.ConfigDir)

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

	// Initialize services
	userService := services.NewUserService(db.DB)
	inboundService := services.NewInboundService(db.DB, lifecycleManager)

	// Initialize handlers
	authHandler := api.NewAuthHandler(db.DB, tokenService)
	coresHandler := api.NewCoresHandler(coreManager)
	usersHandler := api.NewUsersHandler(userService)
	inboundsHandler := api.NewInboundsHandler(inboundService)

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

	// 404 handler
	app.Use(middleware.NotFoundHandler)

	// Health check endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":   "ok",
			"message":  "Isolate Panel is running",
			"database": "connected",
			"version":  "0.1.0",
		})
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

	// Log startup information
	log.Info().
		Str("host", cfg.App.Host).
		Int("port", cfg.App.Port).
		Msg("Starting HTTP server")

	log.Info().Msg("✓ Authentication system enabled")
	log.Info().Msg("✓ Core management enabled")
	log.Info().Msg("✓ User management enabled")
	log.Info().Msg("✓ Inbound management enabled")
	log.Info().Msg("✓ Structured logging enabled")
	log.Info().Msg("✓ Configuration management enabled")

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	if err := app.Listen(addr); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
