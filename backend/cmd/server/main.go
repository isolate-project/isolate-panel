package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm/logger"

	"github.com/vovk4morkovk4/isolate-panel/internal/api"
	"github.com/vovk4morkovk4/isolate-panel/internal/auth"
	"github.com/vovk4morkovk4/isolate-panel/internal/core"
	"github.com/vovk4morkovk4/isolate-panel/internal/database"
	"github.com/vovk4morkovk4/isolate-panel/internal/database/seeds"
	"github.com/vovk4morkovk4/isolate-panel/internal/middleware"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
)

func main() {
	// Database configuration
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/isolate-panel.db"
	}

	// Initialize database
	db, err := database.New(&database.Config{
		Path:         dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		LogLevel:     logger.Info,
	})
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Migrations completed")

	// Run seeders in development
	if os.Getenv("APP_ENV") == "development" {
		log.Println("Running database seeders...")
		seeder := seeds.NewSeeder(db.DB)
		if err := seeder.RunAll(); err != nil {
			log.Fatalf("Failed to run seeders: %v", err)
		}
	}

	// Initialize Core Manager
	supervisorURL := os.Getenv("SUPERVISOR_URL")
	if supervisorURL == "" {
		supervisorURL = "http://localhost:9001/RPC2"
	}
	coreManager := core.NewCoreManager(db.DB, supervisorURL)

	// Initialize Core Lifecycle Manager (lazy loading)
	lifecycleManager := services.NewCoreLifecycleManager(db.DB, coreManager)

	// Initialize Config Service
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		configDir = "./data/cores"
	}
	configService := services.NewConfigService(db.DB, coreManager, configDir)

	// Connect ConfigService to LifecycleManager
	lifecycleManager.SetConfigService(configService)

	// Initialize cores (lazy loading - only start if needed)
	log.Println("Initializing cores (lazy loading)...")
	if err := lifecycleManager.InitializeCores(); err != nil {
		log.Printf("Warning: Failed to initialize cores: %v", err)
		// Don't fail startup, cores can be started manually
	}

	// Initialize JWT token service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-this-in-production-use-env-var-at-least-64-chars-long"
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET environment variable in production!")
	}
	tokenService := auth.NewTokenService(jwtSecret, 15*time.Minute, 7*24*time.Hour)

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
		AppName: "Isolate Panel v0.1.0",
	})

	// Health check endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":   "ok",
			"message":  "Isolate Panel is running",
			"database": "connected",
		})
	})

	// API routes
	apiGroup := app.Group("/api")

	// Public routes (no auth required)
	apiGroup.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Isolate Panel API",
			"version": "0.1.0",
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

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Isolate Panel on port %s", port)
	log.Println("✓ Authentication system enabled")
	log.Println("✓ Core management enabled")
	log.Println("✓ User management enabled")
	log.Println("✓ Inbound management enabled")
	log.Println("")
	log.Println("API Endpoints:")
	log.Println("  Auth:")
	log.Println("    POST /api/auth/login")
	log.Println("    POST /api/auth/refresh")
	log.Println("    POST /api/auth/logout")
	log.Println("  Admin:")
	log.Println("    GET  /api/me")
	log.Println("  Cores:")
	log.Println("    GET  /api/cores")
	log.Println("    POST /api/cores/:name/start")
	log.Println("    POST /api/cores/:name/stop")
	log.Println("    POST /api/cores/:name/restart")
	log.Println("  Users:")
	log.Println("    GET  /api/users")
	log.Println("    POST /api/users")
	log.Println("    GET  /api/users/:id")
	log.Println("    PUT  /api/users/:id")
	log.Println("    DELETE /api/users/:id")
	log.Println("    POST /api/users/:id/regenerate")
	log.Println("    GET  /api/users/:id/inbounds")
	log.Println("  Inbounds:")
	log.Println("    GET  /api/inbounds")
	log.Println("    POST /api/inbounds")
	log.Println("    GET  /api/inbounds/:id")
	log.Println("    PUT  /api/inbounds/:id")
	log.Println("    DELETE /api/inbounds/:id")

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
