package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/isolate-project/isolate-panel/internal/database"
	"github.com/isolate-project/isolate-panel/internal/database/seeds"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	var (
		dbPath  = flag.String("db", "./data/isolate-panel.db", "Database path")
		command = flag.String("cmd", "up", "Command: up, down, steps, version, force, setup")
		steps   = flag.Int("steps", 1, "Number of steps for 'steps' command")
		version = flag.Int("version", 0, "Version for 'force' command")
	)
	flag.Parse()

	// Open database
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create migration manager
	mm, err := database.NewMigrationManager(db)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}
	defer mm.Close()

	// Execute command
	switch *command {
	case "up":
		fmt.Println("Running migrations...")
		if err := mm.Up(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("✓ Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back last migration...")
		if err := mm.Down(); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Println("✓ Rollback completed successfully")

	case "steps":
		fmt.Printf("Running %d migration steps...\n", *steps)
		if err := mm.Steps(*steps); err != nil {
			log.Fatalf("Failed to run steps: %v", err)
		}
		fmt.Println("✓ Steps completed successfully")

	case "version":
		v, dirty, err := mm.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", v)
		if dirty {
			fmt.Println("WARNING: Database is in dirty state!")
		}

	case "force":
		fmt.Printf("Forcing version to %d...\n", *version)
		if err := mm.Force(*version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		fmt.Println("✓ Version forced successfully")

	case "setup":
		fmt.Println("Running initialization setup (admin, settings, cores)...")
		gormDB, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to open GORM database: %v", err)
		}
		seeder := seeds.NewSeeder(gormDB)
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			fmt.Println("WARNING: ADMIN_PASSWORD environment variable not set. Using default.")
		}
		if err := seeder.RunAll(adminPassword); err != nil {
			log.Fatalf("Failed to run setup/seeders: %v", err)
		}
		fmt.Println("✓ Initial setup completed successfully")

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		fmt.Fprintf(os.Stderr, "Available commands: up, down, steps, version, force, setup\n")
		os.Exit(1)
	}
}
