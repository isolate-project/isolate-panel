package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORS returns a CORS middleware configured for the application
func CORS() fiber.Handler {
	origins := os.Getenv("CORS_ORIGINS")
	if origins == "" {
		if os.Getenv("APP_ENV") == "production" {
			origins = "" // No CORS in production (same-origin SPA)
		} else {
			origins = "http://localhost:5173,http://127.0.0.1:5173"
		}
	}

	originList := strings.Split(origins, ",")
	var filtered []string
	for _, o := range originList {
		if o = strings.TrimSpace(o); o != "" {
			filtered = append(filtered, o)
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     filtered,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
		ExposeHeaders:    []string{"Subscription-Userinfo", "Profile-Update-Interval"},
	})
}
