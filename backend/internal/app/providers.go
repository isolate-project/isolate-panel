package app

import (
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/auth"
	"github.com/isolate-project/isolate-panel/internal/cache"
	appconfig "github.com/isolate-project/isolate-panel/internal/config"
	"github.com/isolate-project/isolate-panel/internal/cores"
	_ "github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	_ "github.com/isolate-project/isolate-panel/internal/cores/singbox"
	_ "github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/database"
	"github.com/isolate-project/isolate-panel/internal/haproxy"
	applogger "github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/middleware"
	"github.com/isolate-project/isolate-panel/internal/scheduler"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// App holds all initialized application dependencies wired together.
type App struct {
	StartTime time.Time
	stopQuota chan struct{}
	gormDB    *gorm.DB

	// Infrastructure
	Cache       *cache.CacheManager
	TokenSvc    *auth.TokenService
	LoginRL     *middleware.RateLimiter
	ProtectedRL *middleware.RateLimiter // 60 req/min per admin (standard)
	HeavyRL     *middleware.RateLimiter // 10 req/min per admin (expensive ops)
	SubTokenRL  *middleware.RateLimiter // subscription token rate limiter
	SubIPRL     *middleware.RateLimiter // subscription IP rate limiter

	// Core management
	Cores     *cores.CoreManager
	Lifecycle *services.CoreLifecycleManager
	Config    *services.ConfigService

	// Audit
	Audit  *services.AuditService
	AuditH *api.AuditHandler

	// Domain services
	Notifications *services.NotificationService
	Ports         *services.PortManager
	Settings      *services.SettingsService
	Users         *services.UserService
	Inbounds      *services.InboundService
	Outbounds     *services.OutboundService
	Subscriptions *services.SubscriptionService
	Certs         *services.CertificateService
	Warp          *services.WARPService
	Geo           *services.GeoService
	Backups             *services.BackupService
	BackupSched         *scheduler.BackupScheduler
	TrafficResetSched   *scheduler.TrafficResetScheduler
	Quota               *services.QuotaEnforcer

	// Monitoring services
	Traffic     *services.TrafficCollector
	Connections *services.ConnectionTracker
	Aggregator  *services.DataAggregator
	Retention   *services.DataRetentionService

	// WebSocket hub
	DashboardHub *api.DashboardHub

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

	var err error

	// Cache
	a.Cache, err = cache.NewCacheManager()
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
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
	a.Cores = cores.NewCoreManager(db.DB, cfg.Cores.SupervisorURL, coreCfg)
	a.Lifecycle = services.NewCoreLifecycleManager(db.DB, a.Cores)
	coreAPISecret := cfg.Cores.SingboxAPIKey
	if coreAPISecret == "" {
		coreAPISecret = cfg.Cores.MihomoAPIKey
	}
	a.Config = services.NewConfigService(db.DB, a.Cores, cfg.Cores.ConfigDir, coreAPISecret)
	a.Config.SetCoreConfig(coreCfg)
	a.Lifecycle.SetConfigService(a.Config)
	if err := a.Lifecycle.InitializeCores(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize cores - cores can be started manually")
	}

	// Auth
	a.TokenSvc = auth.NewTokenService(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTokenTTL)*time.Second,
		time.Duration(cfg.JWT.RefreshTokenTTL)*time.Second,
	)
	a.LoginRL = middleware.NewRateLimiter(5, time.Minute)
	a.ProtectedRL = middleware.NewRateLimiter(600, time.Minute)
	a.HeavyRL = middleware.NewRateLimiter(60, time.Minute)

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

	// Domain services
	a.Ports = services.NewPortManager(db.DB)
	a.Settings = services.NewSettingsService(db.DB, a.Cache)
	a.Users = services.NewUserService(db.DB, a.Notifications)
	a.Inbounds = services.NewInboundService(db.DB, a.Lifecycle, a.Ports)
	a.Outbounds = services.NewOutboundService(db.DB, a.Config)
	a.Subscriptions = services.NewSubscriptionService(db.DB, cfg.App.PanelURL, a.Cache)
	a.Users.SetSubscriptionService(a.Subscriptions) // cache invalidation on user changes
	a.Inbounds.SetSubscriptionService(a.Subscriptions) // cache invalidation on inbound changes

	// Monitoring services
	a.Traffic = services.NewTrafficCollector(
		db.DB, a.Settings, time.Duration(cfg.Traffic.CollectInterval)*time.Second,
		cfg.Cores.XrayAPIAddr, cfg.Cores.SingboxAPIAddr, cfg.Cores.MihomoAPIAddr,
		cfg.Cores.SingboxAPIKey, cfg.Cores.MihomoAPIKey,
	)
	a.Connections = services.NewConnectionTracker(
		db.DB, time.Duration(cfg.Traffic.ConnInterval)*time.Second,
		cfg.Cores.XrayAPIAddr, cfg.Cores.SingboxAPIAddr, cfg.Cores.MihomoAPIAddr,
		cfg.Cores.SingboxAPIKey, cfg.Cores.MihomoAPIKey,
	)
	a.Quota = services.NewQuotaEnforcer(db.DB, a.Config, a.Notifications)
	a.Aggregator = services.NewDataAggregator(db.DB, 0)
	a.Retention = services.NewDataRetentionService(db.DB, 0, a.Settings)

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
	}
	a.Lifecycle.SetNotificationService(a.Notifications)

	// API handlers
	a.SystemH = api.NewSystemHandler(a.Connections, a.Cores)
	a.AuthH = api.NewAuthHandler(db.DB, a.TokenSvc, a.Notifications)
	a.CoresH = api.NewCoresHandler(a.Cores)
	a.UsersH = api.NewUsersHandler(a.Users)
	a.InboundsH = api.NewInboundsHandler(a.Inbounds, a.Ports, haproxy.NewPortValidator(db.DB), db.DB)
	a.OutboundsH = api.NewOutboundsHandler(a.Outbounds)
	a.ProtocolsH = api.NewProtocolsHandler()
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
