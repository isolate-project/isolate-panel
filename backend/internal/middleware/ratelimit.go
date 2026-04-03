package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Cleanup old entries every minute
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, timestamps := range rl.requests {
			// Remove timestamps older than window
			valid := make([]time.Time, 0)
			for _, ts := range timestamps {
				if now.Sub(ts) < rl.window {
					valid = append(valid, ts)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get existing timestamps for this key
	timestamps, exists := rl.requests[key]
	if !exists {
		timestamps = make([]time.Time, 0)
	}

	// Remove old timestamps
	valid := make([]time.Time, 0)
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			valid = append(valid, ts)
		}
	}

	// Check if limit exceeded
	if len(valid) >= rl.limit {
		return false
	}

	// Add current timestamp
	valid = append(valid, now)
	rl.requests[key] = valid

	return true
}

// LoginRateLimiter creates a rate limiter middleware for login attempts
func LoginRateLimiter(limiter *RateLimiter) fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.Get("X-Forwarded-For")
		if ip == "" {
			ip = c.IP()
		}

		if !limiter.Allow(ip) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again later.",
			})
		}

		return c.Next()
	}
}

// AuthRateLimiter creates a rate limiter middleware for authenticated endpoints.
// Key is the admin ID extracted from JWT context (set by AuthMiddleware).
// Falls back to IP when admin_id is not in context.
func AuthRateLimiter(limiter *RateLimiter) fiber.Handler {
	return func(c fiber.Ctx) error {
		var key string
		if adminID, ok := c.Locals("admin_id").(uint); ok && adminID > 0 {
			key = fmt.Sprintf("auth:%d", adminID)
		} else {
			key = "auth:" + c.IP()
		}
		if !limiter.Allow(key) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please slow down.",
			})
		}
		return c.Next()
	}
}

// SubscriptionRateLimiter creates a rate limiter for subscription endpoints
// Limits: 10 requests/hour per token, 30 requests/hour per IP
func SubscriptionRateLimiter() fiber.Handler {
	tokenLimiter := NewRateLimiter(10, 1*time.Hour)
	ipLimiter := NewRateLimiter(30, 1*time.Hour)

	return func(c fiber.Ctx) error {
		// Get token from URL
		token := c.Params("token")
		if token != "" {
			if !tokenLimiter.Allow("token:" + token) {
				return c.Status(fiber.StatusTooManyRequests).SendString("Subscription rate limit exceeded (token). Please try again later.")
			}
		}

		// Also check IP
		ip := c.IP()
		if !ipLimiter.Allow("ip:" + ip) {
			return c.Status(fiber.StatusTooManyRequests).SendString("Subscription rate limit exceeded (IP). Please try again later.")
		}

		return c.Next()
	}
}
