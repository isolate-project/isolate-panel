package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/vovk4morkovk4/isolate-panel/internal/logger"
)

// RequestLogger creates a middleware for logging HTTP requests
func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Generate request ID
		requestID := uuid.New().String()
		c.Locals("request_id", requestID)

		// Start timer
		start := time.Now()

		// Create request logger
		reqLogger := logger.Log.With().
			Str("request_id", requestID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Logger()

		// Store logger in context
		c.Locals("logger", reqLogger)

		// Log request
		reqLogger.Info().Msg("Request started")

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log response
		logEvent := reqLogger.Info()
		if err != nil {
			logEvent = reqLogger.Error().Err(err)
		}

		logEvent.
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Int("size", len(c.Response().Body())).
			Msg("Request completed")

		return err
	}
}

// GetLogger retrieves the logger from context
func GetLogger(c fiber.Ctx) zerolog.Logger {
	if log, ok := c.Locals("logger").(zerolog.Logger); ok {
		return log
	}
	return logger.Log
}
