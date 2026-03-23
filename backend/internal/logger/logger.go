package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log zerolog.Logger

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string // "json" or "console"
	Output     string // "stdout", "file", or "both"
	FilePath   string
	MaxSize    int  // megabytes
	MaxBackups int  // number of backups
	MaxAge     int  // days
	Compress   bool // compress rotated files
}

// Init initializes the global logger
func Init(config *Config) error {
	// Parse log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	var writers []io.Writer

	// Console output
	if config.Output == "stdout" || config.Output == "both" {
		var consoleWriter io.Writer
		if config.Format == "console" {
			consoleWriter = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "2006-01-02 15:04:05",
			}
		} else {
			consoleWriter = os.Stdout
		}
		writers = append(writers, consoleWriter)
	}

	// File output with rotation
	if config.Output == "file" || config.Output == "both" {
		// Create log directory if not exists
		logDir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		fileWriter := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
		writers = append(writers, fileWriter)
	}

	// Create multi-writer
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else if len(writers) > 1 {
		writer = io.MultiWriter(writers...)
	} else {
		writer = os.Stdout
	}

	// Initialize global logger
	Log = zerolog.New(writer).With().Timestamp().Caller().Logger()

	return nil
}

// WithComponent creates a logger with component field
func WithComponent(component string) zerolog.Logger {
	return Log.With().Str("component", component).Logger()
}

// WithRequestID creates a logger with request ID field
func WithRequestID(requestID string) zerolog.Logger {
	return Log.With().Str("request_id", requestID).Logger()
}
