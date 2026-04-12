package app

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	appconfig "github.com/isolate-project/isolate-panel/internal/config"
	applogger "github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// StartSubscriptionListener starts a dedicated listener for subscription endpoints.
// This listener can serve on a public-facing port (e.g. 443) with auto-TLS from
// the panel's certificate database, while the main panel stays on localhost only.
func StartSubscriptionListener(a *App, cfg *appconfig.Config) {
	if !cfg.Subscription.Enabled || cfg.Subscription.Port <= 0 {
		applogger.Log.Info().Msg("Subscription listener disabled")
		return
	}

	log := applogger.Log

	subApp := fiber.New(fiber.Config{
		AppName:      "Isolate Subscription Server",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorHandler: middleware.ErrorHandler,
	})

	// Minimal middleware — only security headers + subscription rate limiter
	subApp.Use(middleware.SecurityHeaders())

	// Rate limiters stored in App for graceful shutdown
	subRateLimiter := middleware.SubscriptionRateLimiterWithStop()
	a.SubTokenRL = subRateLimiter.TokenLimiter
	a.SubIPRL = subRateLimiter.IPLimiter
	subApp.Use(subRateLimiter.Handler)

	// Register subscription routes
	subApp.Get("/sub/:token", a.SubscriptionsH.GetAutoDetectSubscription)
	subApp.Get("/sub/:token/clash", a.SubscriptionsH.GetClashSubscription)
	subApp.Get("/sub/:token/singbox", a.SubscriptionsH.GetSingboxSubscription)
	subApp.Get("/sub/:token/qr", a.SubscriptionsH.GetQRCode)
	subApp.Get("/s/:code", a.SubscriptionsH.RedirectShortURL)

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Subscription.Port)

	go func() {
		if cfg.Subscription.AutoTLS {
			// Try to find a certificate from the database
			tlsCfg := findActiveCertificate(a)
			if tlsCfg != nil {
				log.Info().Int("port", cfg.Subscription.Port).Msg("Starting subscription listener with TLS")
				ln, err := tls.Listen("tcp", addr, tlsCfg)
				if err != nil {
					log.Error().Err(err).Msg("Failed to start TLS subscription listener")
					return
				}
				if err := subApp.Listener(ln); err != nil {
					log.Error().Err(err).Msg("Subscription TLS listener stopped")
				}
				return
			}
			log.Warn().Msg("No TLS certificate found for subscription listener, falling back to plain HTTP")
		}

		log.Info().Int("port", cfg.Subscription.Port).Msg("Starting subscription listener (plain HTTP)")
		if err := subApp.Listen(addr); err != nil {
			log.Error().Err(err).Msg("Subscription listener stopped")
		}
	}()
}

// findActiveCertificate searches for a usable TLS certificate in the panel's DB.
func findActiveCertificate(a *App) *tls.Config {
	var cert models.Certificate
	if err := a.gormDB.Where("auto_renew = ? AND cert_path != '' AND key_path != ''", true).
		Order("expires_at DESC").First(&cert).Error; err != nil {
		return nil
	}

	pair, err := tls.LoadX509KeyPair(cert.CertPath, cert.KeyPath)
	if err != nil {
		applogger.Log.Warn().Err(err).Str("domain", cert.Domain).Msg("Failed to load certificate")
		return nil
	}

	return &tls.Config{
		Certificates: []tls.Certificate{pair},
		MinVersion:   tls.VersionTLS12,
	}
}
