// Package main is the entry point for Isolate Panel.
//
// @title           Isolate Panel API
// @version         1.0.0
// @description     Lightweight proxy core management panel for Xray, Sing-box, and Mihomo.
//
// @contact.name   Isolate Panel
// @contact.url    https://github.com/isolate-project/isolate-panel/issues
//
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
//
// @host      localhost:8080
// @BasePath  /api
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT Bearer token. Format: "Bearer {token}"
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm/logger"

	_ "github.com/isolate-project/isolate-panel/docs/swagger" // swagger docs
	isolateapp "github.com/isolate-project/isolate-panel/internal/app"
	appconfig "github.com/isolate-project/isolate-panel/internal/config"
	"github.com/isolate-project/isolate-panel/internal/database"
	"github.com/isolate-project/isolate-panel/internal/database/seeds"
	applogger "github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/version"
)

func main() {
	// Load configuration
	cfg, err := appconfig.Load(os.Getenv("CONFIG_PATH"))
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
	log.Info().Str("version", version.Version).Msg("Starting Isolate Panel")
	log.Info().Str("env", cfg.App.Env).Msg("Environment")

	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Configuration validation failed")
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

	// Run seeders in development
	if cfg.IsDevelopment() {
		log.Info().Msg("Running database seeders")
		seeder := seeds.NewSeeder(db.DB)
		if err := seeder.RunAll(os.Getenv("ADMIN_PASSWORD")); err != nil {
			log.Fatal().Err(err).Msg("Failed to run seeders")
		}
	}

	// Initialize all application dependencies
	application, err := isolateapp.NewApp(cfg, db)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize application")
	}

	// Initialize Fiber app
	fiberApp := fiber.New(fiber.Config{
		AppName:      fmt.Sprintf("%s %s", cfg.App.Name, version.Version),
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    cfg.App.BodyLimit * 1024, // Convert KB to bytes
	})
	fiberApp.Use(middleware.SecurityHeaders())
	fiberApp.Use(middleware.Recovery())
	fiberApp.Use(middleware.CORS())
	fiberApp.Use(middleware.RequestLogger())

	// Register routes and start background workers
	isolateapp.SetupRoutes(fiberApp, application)
	isolateapp.StartWorkers(application)
	isolateapp.StartSubscriptionListener(application, cfg)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	log.Info().Str("addr", addr).Msg("Starting HTTP server")
	go func() {
		if err := fiberApp.Listen(addr); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Gracefully shutting down server...")

	// Wrap shutdown in context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		// 1. Stop accepting new requests
		log.Info().Msg("Stopping HTTP server...")
		if err := fiberApp.Shutdown(); err != nil {
			log.Error().Err(err).Msg("Server shutdown error")
		}

		// 2. Stop background services
		isolateapp.StopWorkers(application)
		isolateapp.StopSubscriptionListener(application)

		// 3. Close database connection
		log.Info().Msg("Closing database connection...")
		if sqlDB, err := db.DB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close database")
			}
		}

		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Server stopped gracefully")
	case <-ctx.Done():
		log.Error().Msg("Shutdown timeout exceeded, forcing exit")
		os.Exit(0)
	}
}
