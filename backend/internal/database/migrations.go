package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type MigrationManager struct {
	db      *sql.DB
	migrate *migrate.Migrate
}

func NewMigrationManager(db *sql.DB) (*MigrationManager, error) {
	// Create source from embedded FS
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &MigrationManager{
		db:      db,
		migrate: m,
	}, nil
}

// Up runs all pending migrations
func (mm *MigrationManager) Up() error {
	if err := mm.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// Down rolls back the last migration
func (mm *MigrationManager) Down() error {
	if err := mm.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}
	return nil
}

// Steps runs n migrations (positive = up, negative = down)
func (mm *MigrationManager) Steps(n int) error {
	if err := mm.migrate.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d steps: %w", n, err)
	}
	return nil
}

// Version returns current migration version
func (mm *MigrationManager) Version() (uint, bool, error) {
	version, dirty, err := mm.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (mm *MigrationManager) Force(version int) error {
	if err := mm.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}
	return nil
}

// Close closes the migration manager
func (mm *MigrationManager) Close() error {
	sourceErr, dbErr := mm.migrate.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return dbErr
}
