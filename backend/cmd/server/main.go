package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm/logger"

	"github.com/vovk4morkovk4/isolate-panel/internal/database"
	"github.com/vovk4morkovk4/isolate-panel/internal/database/seeds"
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

	// API routes placeholder
	api := app.Group("/api")
	api.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Isolate Panel API",
			"version": "0.1.0",
		})
	})

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Isolate Panel on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

	log.Printf("Starting Isolate Panel on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
