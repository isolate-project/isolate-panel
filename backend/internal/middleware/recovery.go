package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/logger"
)

// Recovery returns a middleware that recovers from panics
func Recovery() fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Get logger from context or use global
				log := GetLogger(c)

				// Log panic with stack trace
				log.Error().
					Interface("panic", r).
					Str("stack", string(debug.Stack())).
					Msg("Panic recovered")

				// Return internal server error
				err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
				if err != nil {
					logger.Log.Error().Err(err).Msg("Failed to send error response")
				}
			}
		}()

		return c.Next()
	}
}

// ErrorHandler is a custom error handler for Fiber
func ErrorHandler(c fiber.Ctx, err error) error {
	// Get logger from context
	log := GetLogger(c)

	// Default to 500 Internal Server Error
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log error
	log.Error().
		Err(err).
		Int("status", code).
		Str("message", message).
		Msg("Request error")

	// Send error response
	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(c fiber.Ctx) error {
	log := GetLogger(c)
	log.Warn().
		Str("path", c.Path()).
		Msg("Route not found")

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": fmt.Sprintf("Route %s not found", c.Path()),
	})
}
