package middleware

import "github.com/gofiber/fiber/v3"

// SecurityHeaders adds common security headers to every response.
func SecurityHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Cross-Origin-Resource-Policy", "same-origin")
		c.Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
		// Add HSTS only for HTTPS (subscription listener serves TLS)
		if c.Scheme() == "https" {
			c.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; object-src 'none'; base-uri 'self'; form-action 'self'")
		return c.Next()
	}
}
