package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/cache"
	appconfig "github.com/isolate-project/isolate-panel/internal/config"
	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	"github.com/isolate-project/isolate-panel/internal/cores/singbox"
	"github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/eventbus"
	"github.com/isolate-project/isolate-panel/internal/protocol"
	"github.com/isolate-project/isolate-panel/internal/database"
	"github.com/isolate-project/isolate-panel/internal/haproxy"
	applogger "github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/scheduler"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// App holds all initialized application dependencies wired together.
type App struct {
	StartTime time.Time
	stopQuota chan struct{}
	gormDB    *gorm.DB
	subApp    *fiber.App

	// Infrastructure
	Cache              *cache.CacheManager
	TokenSvc           *auth.TokenService
	SessionManager     *auth.BFFSessionManager
	SubscriptionSigner *auth.SubscriptionSigner
	WebAuthnSvc        *auth.WebAuthnService
	LoginRL            *middleware.RateLimiter
	RefreshLogoutRL *middleware.RateLimiter // 10 req/min per IP (refresh/logout)
	ProtectedRL     *middleware.RateLimiter // 60 req/min per admin (standard)
	HeavyRL         *middleware.RateLimiter // 10 req/min per admin (expensive ops)
	SubTokenRL      *middleware.RateLimiter // subscription token rate limiter
	SubIPRL         *middleware.RateLimiter // subscription IP rate limiter

	// Event Bus
	EventBus *eventbus.Registry

	// Core management
	Cores     *cores.CoreManager
	Lifecycle *services.CoreLifecycleManager
	Config    *services.ConfigService
	Watchdog  *Watchdog

	// Audit
	Audit  *services.AuditService
	AuditH *api.AuditHandler

	// Domain services
	Notifications     *services.NotificationService
	Ports             *services.PortManager
	Settings          *services.SettingsService
	Users             *services.UserService
	Inbounds          *services.InboundService
	Outbounds         *services.OutboundService
	Subscriptions     *services.SubscriptionService
	Certs             *services.CertificateService
	Warp              *services.WARPService
	Geo               *services.GeoService
	Backups           *services.BackupService
	BackupSched       *scheduler.BackupScheduler
	TrafficResetSched *scheduler.TrafficResetScheduler
	LogRetentionSched *scheduler.LogRetentionScheduler
	Quota             *services.QuotaEnforcer

	// Phase 5.4: Per-node API key management
	NodeAuth *services.NodeAuthService

	// Monitoring services
	Traffic     *services.TrafficCollector
	Connections *services.ConnectionTracker
	Aggregator  *services.DataAggregator
	Retention   *services.DataRetentionService

	// WebSocket hub
	DashboardHub *api.DashboardHub

	// Protocol registry
	ProtocolRegistry protocol.Registry

	// API handlers
	SystemH        *api.SystemHandler
	AuthH          *api.AuthHandler
	CoresH         *api.CoresHandler
	UsersH         *api.UsersHandler
	InboundsH      *api.InboundsHandler
	OutboundsH     *api.OutboundsHandler
	ProtocolsH     *api.ProtocolsHandler
	SubscriptionsH *api.SubscriptionsHandler
	CertificatesH  *api.CertificatesHandler
	StatsH         *api.StatsHandler
	WarpH          *api.WarpHandler
	BackupH        *api.BackupHandler
	NotificationsH *api.NotificationHandler
	SettingsH      *api.SettingsHandler
}

// NewApp creates and wires all application dependencies.
func NewApp(cfg *appconfig.Config, db *database.Database) (*App, error) {
	log := applogger.Log
	a := &App{
		StartTime: time.Now(),
		stopQuota: make(chan struct{}),
		gormDB:    db.DB,
	}

	// Explicit registration — replaces implicit init() side-effects
	protocol.RegisterAllProtocols()
	xray.Register()
	singbox.Register()
	mihomo.Register()
	middleware.SetupValidation()

	var err error

	// Event Bus
	a.EventBus = eventbus.NewRegistry()
	a.ProtocolRegistry = protocol.NewRegistryAdapter()

	// Cache
	a.Cache, err = cache.NewCacheManager()
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}

	// Phase 5.5: Database Field Encryption Plugin
	// Automatically encrypts/decrypts sensitive database fields using AES-256-GCM.
	// Gracefully degrades to unencrypted if no key is configured (dev environments).
	if encPlugin, err := database.NewEncryptionPluginFromEnv(); err != nil {
		log.Info().Err(err).Msg("Field encryption key not configured - running without encryption")
	} else {
		encPlugin.RegisterDefaultFields()
		if err := db.DB.Use(encPlugin); err != nil {
			log.Warn().Err(err).Msg("Failed to register encryption plugin - field encryption disabled")
		} else {
			log.Info().Msg("Database field encryption enabled")
		}
	}

	// Core management
	coreCfg := &cores.CoreConfig{
		APIPort:       cfg.Cores.APIPort,
		LogDirectory:  cfg.Cores.LogDirectory,
		ClashAPIPort:  cfg.Cores.ClashAPIPort,
		MihomoAPIPort: cfg.Cores.MihomoAPIPort,
		V2RayAPIPort:  cfg.Cores.V2RayAPIPort,
	}
	coreCfg.ApplyDefaults()

	// Resolve data directories with defaults
	dataDir := cfg.Data.DataDir
	if dataDir == "" {
		dataDir = "/app/data"
	}
	warpDir := cfg.Data.WarpDir
	if warpDir == "" {
		warpDir = dataDir + "/warp"
	}
	geoDir := cfg.Data.GeoDir
	if geoDir == "" {
		geoDir = dataDir + "/geo"
	}

	// Phase 5.4: Per-node API key management
	// Create NodeAuthService early for per-core API key lookup
	a.NodeAuth = services.NewNodeAuthService(db.DB)

	// Create per-core API key lookup callback with fallback to global config
	getCoreAPISecret := func(coreID uint) (string, error) {
		key, err := a.NodeAuth.GetCoreAPIKey(coreID)
		if err != nil {
			// Fallback to global config secret for backwards compatibility
			// when no per-core key has been generated yet
			globalSecret := cfg.Cores.SingboxAPIKey
			if globalSecret == "" {
				globalSecret = cfg.Cores.MihomoAPIKey
			}
			if globalSecret != "" {
				return globalSecret, nil
			}
			return "", err
		}
		return key, nil
	}

	// Create callbacks for stats clients that lookup by core name
	getSingboxAPIKey := func() string {
		key, err := a.NodeAuth.GetCoreAPIKeyByName("singbox")
		if err != nil {
			return cfg.Cores.SingboxAPIKey
		}
		return key
	}
	getMihomoAPIKey := func() string {
		key, err := a.NodeAuth.GetCoreAPIKeyByName("mihomo")
		if err != nil {
			return cfg.Cores.MihomoAPIKey
		}
		return key
	}

	// Resolve V2Ray API listen address
	v2rayAPIListenAddr := ""
	if cfg.Cores.V2RayAPIPort > 0 {
		v2rayAPIListenAddr = fmt.Sprintf("127.0.0.1:%d", cfg.Cores.V2RayAPIPort)
	}

	a.Cores = cores.NewCoreManager(db.DB, cfg.Cores.SupervisorURL, coreCfg, cfg.Cores.ConfigDir, warpDir, geoDir, getCoreAPISecret, v2rayAPIListenAddr)
	a.Lifecycle = services.NewCoreLifecycleManager(db.DB, a.Cores)
	a.Watchdog = NewWatchdog(db.DB, a.Cores, 30*time.Second, 5*time.Second)
	a.Config = services.NewConfigService(db.DB, a.Cores, cfg.Cores.ConfigDir, getCoreAPISecret)
	a.Config.SetCoreConfig(coreCfg)
	a.Lifecycle.SetConfigService(a.Config)
	if err := a.Lifecycle.InitializeCores(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize cores - cores can be started manually")
	}

	// Auth
	auth.SetPepper(cfg.Security.PasswordPepper)

	adminValidator := func(adminID uint) (isActive bool, isSuperAdmin bool, mustChangePassword bool, err error) {
		var admin models.Admin
		if err := db.DB.Select("is_active, is_super_admin, must_change_password").First(&admin, adminID).Error; err != nil {
			return false, false, false, err
		}
		return admin.IsActive, admin.IsSuperAdmin, admin.MustChangePassword, nil
	}
	a.TokenSvc, err = auth.NewTokenService(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTokenTTL)*time.Second,
		time.Duration(cfg.JWT.RefreshTokenTTL)*time.Second,
		adminValidator,
		db.DB,
	)
	if err != nil {
		return nil, fmt.Errorf("init token service: %w", err)
	}
	a.SessionManager = auth.NewBFFSessionManager(time.Duration(cfg.JWT.RefreshTokenTTL) * time.Second)
	a.SubscriptionSigner = auth.NewSubscriptionSigner(cfg.JWT.Secret)
	a.LoginRL = middleware.NewRateLimiter(5, time.Minute)
	a.RefreshLogoutRL = middleware.NewRateLimiter(10, time.Minute)
	a.ProtectedRL = middleware.NewRateLimiter(600, time.Minute)
	a.HeavyRL = middleware.NewRateLimiter(60, time.Minute)

	// WebAuthn
	rpID := "localhost"
	rpOrigin := cfg.App.PanelURL
	if rpOrigin == "" {
		rpOrigin = "http://localhost:8080"
	}
	// Extract RP ID from origin (hostname only)
	if len(rpOrigin) > 0 {
		// Simple extraction - remove protocol and port
		if idx := strings.Index(rpOrigin, "://"); idx != -1 {
			rpID = rpOrigin[idx+3:]
			if idx := strings.Index(rpID, ":"); idx != -1 {
				rpID = rpID[:idx]
			}
			if idx := strings.Index(rpID, "/"); idx != -1 {
				rpID = rpID[:idx]
			}
		}
	}
	a.WebAuthnSvc, err = auth.NewWebAuthnService(db.DB, rpID, rpOrigin, "Isolate Panel")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize WebAuthn service - WebAuthn features disabled")
		a.WebAuthnSvc = nil
	}

	// Audit
	a.Audit = services.NewAuditService(db.DB)
	a.AuditH = api.NewAuditHandler(a.Audit)

	// Notifications (early — other services depend on it)
	a.Notifications = services.NewNotificationService(db.DB,
		cfg.Notifications.WebhookURL, cfg.Notifications.WebhookSecret,
		cfg.Notifications.TelegramToken, cfg.Notifications.TelegramChatID,
	)
	if err := a.Notifications.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Notification service")
	}
	a.Notifications.Start()

	// Domain services
	a.Ports = services.NewPortManager(db.DB)
	a.Settings = services.NewSettingsService(db.DB, a.Cache)
	a.Users = services.NewUserService(db.DB, a.Notifications)
	a.Inbounds = services.NewInboundService(db.DB, a.Lifecycle, a.Ports, a.ProtocolRegistry)
	a.Outbounds = services.NewOutboundService(db.DB, a.Config, a.ProtocolRegistry)
	a.Subscriptions = services.NewSubscriptionService(db.DB, cfg.App.PanelURL, a.Cache)
	a.Users.SetSubscriptionService(a.Subscriptions)    // cache invalidation on user changes
	a.Inbounds.SetSubscriptionService(a.Subscriptions) // cache invalidation on inbound changes

	// Monitoring services (using per-core API key callbacks)
	a.Traffic = services.NewTrafficCollector(
		db.DB, a.Settings, time.Duration(cfg.Traffic.CollectInterval)*time.Second,
		cfg.Cores.XrayAPIAddr, cfg.Cores.SingboxAPIAddr, cfg.Cores.MihomoAPIAddr,
		getSingboxAPIKey, getMihomoAPIKey,
	)
	a.Config.SetTrafficCollector(a.Traffic)
	a.Connections = services.NewConnectionTracker(
		db.DB, time.Duration(cfg.Traffic.ConnInterval)*time.Second,
		cfg.Cores.XrayAPIAddr, cfg.Cores.SingboxAPIAddr, cfg.Cores.MihomoAPIAddr,
		getSingboxAPIKey, getMihomoAPIKey,
	)
	a.Quota = services.NewQuotaEnforcer(db.DB, a.Config, a.Notifications)
	a.Aggregator = services.NewDataAggregator(db.DB, 0)
	a.Retention = services.NewDataRetentionService(db.DB, 0, a.Settings)

	// Resolve additional data directories with defaults
	backupDir := cfg.Data.BackupDir
	if backupDir == "" {
		backupDir = dataDir + "/backups"
	}
	certDir := cfg.Data.CertDir
	if certDir == "" {
		certDir = "/etc/isolate-panel/certs"
	}

	// WARP and Geo
	a.Warp = services.NewWARPService(db.DB, warpDir)
	if err := a.Warp.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize WARP service")
	}
	a.Geo = services.NewGeoService(db.DB, geoDir)
	if err := a.Geo.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Geo service")
	}

	// Backup
	a.Backups = services.NewBackupService(db.DB, a.Settings, backupDir, dataDir)
	if err := a.Backups.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Backup service")
	}
	a.BackupSched = scheduler.NewBackupScheduler(db.DB, a.Backups)
	if err := a.BackupSched.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Backup Scheduler")
	}
	a.TrafficResetSched = scheduler.NewTrafficResetScheduler(a.Settings, a.Quota)
	if err := a.TrafficResetSched.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Traffic Reset Scheduler")
	}
	logFilePath := ""
	if cfg.Logging.Output == "file" || cfg.Logging.Output == "both" {
		logFilePath = cfg.Logging.FilePath
	}
	a.LogRetentionSched = scheduler.NewLogRetentionScheduler(db.DB, logFilePath)
	if err := a.LogRetentionSched.Initialize(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Log Retention Scheduler")
	}

	// Certificates (optional — graceful degradation if ACME unavailable)
	cfCreds := buildCloudflareCredentials()
	dnsProvider := ""
	if len(cfCreds) > 0 {
		dnsProvider = "cloudflare"
	}
	a.Certs, err = services.NewCertificateService(db.DB, services.CertificateServiceConfig{
		CertDir:     certDir,
		Email:       cfg.App.AdminEmail,
		DNSProvider: dnsProvider,
		Credentials: cfCreds,
		Staging:     cfg.App.Env == "development",
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize certificate service - ACME features disabled")
	}

	// Wire notification service to dependents
	if a.Certs != nil {
		a.Certs.SetNotificationService(a.Notifications)
		a.Certs.OnCertChange = InvalidateCertCache
	}
	a.Lifecycle.SetNotificationService(a.Notifications)

	// API handlers
	a.SystemH = api.NewSystemHandler(a.Connections, a.Cores)
	a.AuthH = api.NewAuthHandler(db.DB, a.TokenSvc, a.SessionManager, a.Notifications, a.WebAuthnSvc)
	a.CoresH = api.NewCoresHandler(a.Cores)
	a.UsersH = api.NewUsersHandler(a.Users)
	a.InboundsH = api.NewInboundsHandler(a.Inbounds, a.Ports, haproxy.NewPortValidator(db.DB), db.DB)
	a.OutboundsH = api.NewOutboundsHandler(a.Outbounds)
	a.ProtocolsH = api.NewProtocolsHandler(a.ProtocolRegistry)
	a.SubscriptionsH = api.NewSubscriptionsHandler(a.Subscriptions)
	a.CertificatesH = api.NewCertificatesHandler(a.Certs, db.DB)
	a.StatsH = api.NewStatsHandler(db.DB, a.Traffic, a.Connections)
	a.WarpH = api.NewWarpHandler(a.Warp, a.Geo, a.Config)
	a.BackupH = api.NewBackupHandler(a.Backups, a.BackupSched)
	a.NotificationsH = api.NewNotificationHandler(a.Notifications)
	a.SettingsH = api.NewSettingsHandler(a.Settings, a.Traffic)
	a.SettingsH.SetTrafficResetScheduler(a.TrafficResetSched)

	a.DashboardHub = api.NewDashboardHub(db.DB, a.Connections, a.TokenSvc)

	return a, nil
}

func buildCloudflareCredentials() map[string]string {
	creds := make(map[string]string)
	if v := os.Getenv("CLOUDFLARE_API_KEY"); v != "" {
		creds["api_key"] = v
	}
	if v := os.Getenv("CLOUDFLARE_EMAIL"); v != "" {
		creds["email"] = v
	}
	if v := os.Getenv("CLOUDFLARE_API_TOKEN"); v != "" {
		creds["api_token"] = v
	}
	return creds
}
