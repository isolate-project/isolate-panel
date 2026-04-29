package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/time/rate"
)

// RateLimiter uses a token-bucket algorithm for O(1) memory and CPU per key.
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	r := rate.Every(window / time.Duration(requests))
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    requests,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	lim, ok := rl.limiters[key]
	if !ok {
		lim = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = lim
	}
	rl.mu.Unlock()
	return lim.Allow()
}

func (rl *RateLimiter) Stop() {}

func LoginRateLimiter(limiter *RateLimiter) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !limiter.Allow(c.IP()) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again later.",
			})
		}
		return c.Next()
	}
}

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

type SubRateLimiterBundle struct {
	TokenLimiter *RateLimiter
	IPLimiter    *RateLimiter
	Handler      fiber.Handler
}

func SubscriptionRateLimiterWithStop() SubRateLimiterBundle {
	tokenLimiter := NewRateLimiter(10, time.Hour)
	ipLimiter := NewRateLimiter(30, time.Hour)

	handler := func(c fiber.Ctx) error {
		token := c.Params("token")
		if token != "" {
			if !tokenLimiter.Allow("token:" + token) {
				return c.Status(fiber.StatusTooManyRequests).SendString("Subscription rate limit exceeded (token). Please try again later.")
			}
		}
		ip := c.IP()
		if !ipLimiter.Allow("ip:" + ip) {
			return c.Status(fiber.StatusTooManyRequests).SendString("Subscription rate limit exceeded (IP). Please try again later.")
		}
		return c.Next()
	}

	return SubRateLimiterBundle{
		TokenLimiter: tokenLimiter,
		IPLimiter:    ipLimiter,
		Handler:      handler,
	}
}
