package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm/logger"

	"github.com/vovk4morkovk4/isolate-panel/internal/api"
	"github.com/vovk4morkovk4/isolate-panel/internal/auth"
	"github.com/vovk4morkovk4/isolate-panel/internal/database"
	"github.com/vovk4morkovk4/isolate-panel/internal/database/seeds"
	"github.com/vovk4morkovk4/isolate-panel/internal/middleware"
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

	// Initialize JWT token service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-this-in-production-use-env-var-at-least-64-chars-long"
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET environment variable in production!")
	}
	tokenService := auth.NewTokenService(jwtSecret, 15*time.Minute, 7*24*time.Hour)

	// Initialize handlers
	authHandler := api.NewAuthHandler(db.DB, tokenService)

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

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Isolate Panel on port %s", port)
	log.Println("✓ Authentication system enabled")
	log.Println("  - POST /api/auth/login - Login")
	log.Println("  - POST /api/auth/refresh - Refresh token")
	log.Println("  - POST /api/auth/logout - Logout")
	log.Println("  - GET /api/me - Get current admin info (protected)")

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
