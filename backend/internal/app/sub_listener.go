package app

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	appconfig "github.com/isolate-project/isolate-panel/internal/config"
	applogger "github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
)

var (
	cachedCert   *tls.Certificate
	cachedCertMu sync.RWMutex
	cachedCertAt time.Time
)

const certCacheTTL = 60 * time.Second

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

	a.subApp = subApp

	subApp.Use(middleware.SecurityHeaders())

	subRateLimiter := middleware.SubscriptionRateLimiterWithStop()
	a.SubTokenRL = subRateLimiter.TokenLimiter
	a.SubIPRL = subRateLimiter.IPLimiter
	subApp.Use(subRateLimiter.Handler)

	subApp.Get("/sub/:token", a.SubscriptionsH.GetAutoDetectSubscription)
	subApp.Get("/sub/:token/clash", a.SubscriptionsH.GetClashSubscription)
	subApp.Get("/sub/:token/singbox", a.SubscriptionsH.GetSingboxSubscription)
	subApp.Get("/sub/:token/isolate", a.SubscriptionsH.GetIsolateSubscription)
	subApp.Get("/sub/:token/qr", a.SubscriptionsH.GetQRCode)
	subApp.Get("/s/:code", a.SubscriptionsH.RedirectShortURL)

	addr := fmt.Sprintf("%s:%d", cfg.Subscription.Host, cfg.Subscription.Port)

	go func() {
		if cfg.Subscription.AutoTLS {
			tlsCfg := &tls.Config{
				GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					return getCachedCertificate(a)
				},
			}
			log.Info().Int("port", cfg.Subscription.Port).Msg("Starting subscription listener with TLS")
			ln, err := tls.Listen("tcp", addr, tlsCfg)
			if err != nil {
				if cfg.App.Env == "production" && !cfg.Subscription.AllowHTTP {
					log.Error().Err(err).Msg("AutoTLS failed, HTTP not allowed in production")
					return
				}
				log.Warn().Err(err).Msg("AutoTLS failed, falling back to HTTP")
			} else {
				if err := subApp.Listener(ln); err != nil {
					log.Error().Err(err).Msg("Subscription TLS listener stopped")
				}
				return
			}
		}

		log.Info().Int("port", cfg.Subscription.Port).Msg("Starting subscription listener (plain HTTP)")
		if err := subApp.Listen(addr); err != nil {
			log.Error().Err(err).Msg("Subscription listener stopped")
		}
	}()
}

func StopSubscriptionListener(a *App) {
	if a.subApp == nil {
		return
	}

	log := applogger.Log
	log.Info().Msg("Stopping subscription listener...")

	if err := a.subApp.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Subscription listener shutdown error")
	}

	log.Info().Msg("Subscription listener stopped")
}

// getCachedCertificate returns a cached tls.Certificate, refreshing from DB every 60s.
func getCachedCertificate(a *App) (*tls.Certificate, error) {
	cachedCertMu.RLock()
	if cachedCert != nil && time.Since(cachedCertAt) < certCacheTTL {
		cert := cachedCert
		cachedCertMu.RUnlock()
		return cert, nil
	}
	cachedCertMu.RUnlock()

	cachedCertMu.Lock()
	defer cachedCertMu.Unlock()

	if cachedCert != nil && time.Since(cachedCertAt) < certCacheTTL {
		return cachedCert, nil
	}

	var cert models.Certificate
	if err := a.gormDB.Where("auto_renew = ? AND cert_path != '' AND key_path != ''", true).
		Order("not_after DESC").First(&cert).Error; err != nil {
		return nil, err
	}

	pair, err := tls.LoadX509KeyPair(cert.CertPath, cert.KeyPath)
	if err != nil {
		applogger.Log.Warn().Err(err).Str("domain", cert.Domain).Msg("Failed to load certificate")
		return nil, err
	}

	cachedCert = &pair
	cachedCertAt = time.Now()
	return &pair, nil
}

// InvalidateCertCache forces a reload on next TLS handshake (e.g., after cert renewal).
func InvalidateCertCache() {
	cachedCertMu.Lock()
	cachedCert = nil
	cachedCertAt = time.Time{}
	cachedCertMu.Unlock()
}
