package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB     *gorm.DB
	SqlDB  *sql.DB
	Config *Config
}

type Config struct {
	Path         string
	MaxOpenConns int
	MaxIdleConns int
	LogLevel     logger.LogLevel
}

// New creates a new database connection with optimizations for SQLite
func New(config *Config) (*Database, error) {
	// Open SQLite with optimizations
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=ON", config.Path)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Database{
		DB:     db,
		SqlDB:  sqlDB,
		Config: config,
	}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.SqlDB.Close()
}

// Ping checks if the database is reachable
func (d *Database) Ping() error {
	return d.SqlDB.Ping()
}

// RunMigrations runs all pending migrations
func (d *Database) RunMigrations() error {
	mm, err := NewMigrationManager(d.SqlDB)
	if err != nil {
		return fmt.Errorf("failed to create migration manager: %w", err)
	}
	defer mm.Close()

	if err := mm.Up(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
