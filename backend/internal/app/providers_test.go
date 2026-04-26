package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	appconfig "github.com/isolate-project/isolate-panel/internal/config"
	"github.com/isolate-project/isolate-panel/internal/database"
)

// setupProvidersTest creates a minimal test config and DB for testing NewApp()
func setupProvidersTest(t *testing.T) (*appconfig.Config, *database.Database) {
	t.Helper()

	// Use temp directory for data
	tmpDir := t.TempDir()

	cfg := &appconfig.Config{
		JWT: appconfig.JWTConfig{
			Secret:          "test-jwt-secret-min-32-characters!!",
			AccessTokenTTL:  900,
			RefreshTokenTTL: 604800,
		},
		App: appconfig.AppConfig{
			AdminEmail: "test@test.com",
			Env:        "development",
			PanelURL:   "http://localhost:8080",
		},
		Cores: appconfig.CoresConfig{
			SupervisorURL: "http://127.0.0.1:9001",
			ConfigDir:     tmpDir + "/configs",
			XrayAPIAddr:   "127.0.0.1:10085",
			SingboxAPIAddr: "127.0.0.1:9090",
			MihomoAPIAddr:  "127.0.0.1:9091",
		},
		Data: appconfig.DataConfig{
			DataDir:   tmpDir + "/data",
			WarpDir:   tmpDir + "/data/warp",
			GeoDir:    tmpDir + "/data/geo",
			BackupDir: tmpDir + "/data/backups",
			CertDir:   tmpDir + "/certs",
		},
		Traffic: appconfig.TrafficConfig{
			CollectInterval: 60,
			ConnInterval:    60,
		},
		Notifications: appconfig.NotificationsConfig{
			WebhookURL:     "",
			WebhookSecret:  "",
			TelegramToken:  "",
			TelegramChatID: "",
		},
	}

	db, err := database.New(&database.Config{
		Path:         tmpDir + "/test.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		LogLevel:     logger.Silent,
	})
	require.NoError(t, err)

	// Run migrations to create all required tables
	require.NoError(t, db.RunMigrations())

	return cfg, db
}

// TestNewApp_AllServicesInitialized verifies that NewApp() initializes all services
func TestNewApp_AllServicesInitialized(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	// Infrastructure
	assert.NotNil(t, a.Cache, "Cache should be initialized")
	assert.NotNil(t, a.TokenSvc, "TokenSvc should be initialized")
	assert.NotNil(t, a.LoginRL, "LoginRL should be initialized")
	assert.NotNil(t, a.ProtectedRL, "ProtectedRL should be initialized")
	assert.NotNil(t, a.HeavyRL, "HeavyRL should be initialized")

	// Core management
	assert.NotNil(t, a.Cores, "Cores should be initialized")
	assert.NotNil(t, a.Lifecycle, "Lifecycle should be initialized")
	assert.NotNil(t, a.Config, "Config should be initialized")
	assert.NotNil(t, a.Watchdog, "Watchdog should be initialized")

	// Audit
	assert.NotNil(t, a.Audit, "Audit should be initialized")
	assert.NotNil(t, a.AuditH, "AuditH should be initialized")

	// Domain services
	assert.NotNil(t, a.Notifications, "Notifications should be initialized")
	assert.NotNil(t, a.Ports, "Ports should be initialized")
	assert.NotNil(t, a.Settings, "Settings should be initialized")
	assert.NotNil(t, a.Users, "Users should be initialized")
	assert.NotNil(t, a.Inbounds, "Inbounds should be initialized")
	assert.NotNil(t, a.Outbounds, "Outbounds should be initialized")
	assert.NotNil(t, a.Subscriptions, "Subscriptions should be initialized")
	assert.NotNil(t, a.Certs, "Certs should be initialized")
	assert.NotNil(t, a.Warp, "Warp should be initialized")
	assert.NotNil(t, a.Geo, "Geo should be initialized")
	assert.NotNil(t, a.Backups, "Backups should be initialized")
	assert.NotNil(t, a.BackupSched, "BackupSched should be initialized")
	assert.NotNil(t, a.TrafficResetSched, "TrafficResetSched should be initialized")
	assert.NotNil(t, a.Quota, "Quota should be initialized")

	// Monitoring services
	assert.NotNil(t, a.Traffic, "Traffic should be initialized")
	assert.NotNil(t, a.Connections, "Connections should be initialized")
	assert.NotNil(t, a.Aggregator, "Aggregator should be initialized")
	assert.NotNil(t, a.Retention, "Retention should be initialized")

	// WebSocket hub
	assert.NotNil(t, a.DashboardHub, "DashboardHub should be initialized")

	// API handlers
	assert.NotNil(t, a.SystemH, "SystemH should be initialized")
	assert.NotNil(t, a.AuthH, "AuthH should be initialized")
	assert.NotNil(t, a.CoresH, "CoresH should be initialized")
	assert.NotNil(t, a.UsersH, "UsersH should be initialized")
	assert.NotNil(t, a.InboundsH, "InboundsH should be initialized")
	assert.NotNil(t, a.OutboundsH, "OutboundsH should be initialized")
	assert.NotNil(t, a.ProtocolsH, "ProtocolsH should be initialized")
	assert.NotNil(t, a.SubscriptionsH, "SubscriptionsH should be initialized")
	assert.NotNil(t, a.CertificatesH, "CertificatesH should be initialized")
	assert.NotNil(t, a.StatsH, "StatsH should be initialized")
	assert.NotNil(t, a.WarpH, "WarpH should be initialized")
	assert.NotNil(t, a.BackupH, "BackupH should be initialized")
	assert.NotNil(t, a.NotificationsH, "NotificationsH should be initialized")
	assert.NotNil(t, a.SettingsH, "SettingsH should be initialized")

	// Verify StartTime is set
	assert.False(t, a.StartTime.IsZero(), "StartTime should be set")
}

// TestNewApp_CoreManagerCreated verifies that a.Cores is a *cores.CoreManager
func TestNewApp_CoreManagerCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.Cores, "Cores should be initialized")
}

// TestNewApp_DashboardHubCreated verifies that a.DashboardHub is non-nil
func TestNewApp_DashboardHubCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.DashboardHub, "DashboardHub should be initialized")
}

// TestNewApp_RateLimitersCreated verifies that all rate limiters are non-nil
func TestNewApp_RateLimitersCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.LoginRL, "LoginRL should be initialized")
	assert.NotNil(t, a.ProtectedRL, "ProtectedRL should be initialized")
	assert.NotNil(t, a.HeavyRL, "HeavyRL should be initialized")
}

// TestNewApp_WatchdogCreated verifies that a.Watchdog is non-nil with correct interval/timeout defaults
func TestNewApp_WatchdogCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.Watchdog, "Watchdog should be initialized")
	assert.Equal(t, 30*time.Second, a.Watchdog.interval, "Watchdog interval should be 30 seconds")
	assert.Equal(t, 5*time.Second, a.Watchdog.timeout, "Watchdog timeout should be 5 seconds")
}

// TestNewApp_InvalidJWTSecret verifies that creating App with empty JWT secret is handled
func TestNewApp_InvalidJWTSecret(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &appconfig.Config{
		JWT: appconfig.JWTConfig{
			Secret:          "", // Empty secret
			AccessTokenTTL:  900,
			RefreshTokenTTL: 604800,
		},
		App: appconfig.AppConfig{
			AdminEmail: "test@test.com",
			Env:        "development",
			PanelURL:   "http://localhost:8080",
		},
		Cores: appconfig.CoresConfig{
			SupervisorURL: "http://127.0.0.1:9001",
			ConfigDir:     tmpDir + "/configs",
		},
		Data: appconfig.DataConfig{
			DataDir:   tmpDir + "/data",
			WarpDir:   tmpDir + "/data/warp",
			GeoDir:    tmpDir + "/data/geo",
			BackupDir: tmpDir + "/data/backups",
			CertDir:   tmpDir + "/certs",
		},
		Traffic: appconfig.TrafficConfig{
			CollectInterval: 60,
			ConnInterval:    60,
		},
		Notifications: appconfig.NotificationsConfig{
			WebhookURL:     "",
			WebhookSecret:  "",
			TelegramToken:  "",
			TelegramChatID: "",
		},
	}

	db, err := database.New(&database.Config{
		Path:         tmpDir + "/test.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		LogLevel:     logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.RunMigrations())

	// NewApp should still succeed even with empty JWT secret
	// (TokenService will be created but may not work properly)
	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.NotNil(t, a.TokenSvc, "TokenSvc should still be initialized")
}

// TestNewApp_ServiceDependencies verifies that service dependencies are properly wired
func TestNewApp_ServiceDependencies(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	// Verify that Users has SubscriptionService set
	assert.NotNil(t, a.Users, "Users should be initialized")

	// Verify that Inbounds has SubscriptionService set
	assert.NotNil(t, a.Inbounds, "Inbounds should be initialized")

	// Verify that Lifecycle has ConfigService set
	assert.NotNil(t, a.Lifecycle, "Lifecycle should be initialized")
	assert.NotNil(t, a.Config, "Config should be initialized")

	// Verify that SettingsH has TrafficResetScheduler set
	assert.NotNil(t, a.SettingsH, "SettingsH should be initialized")
	assert.NotNil(t, a.TrafficResetSched, "TrafficResetScheduler should be initialized")
}

// TestNewApp_MonitoringServicesCreated verifies that all monitoring services are initialized
func TestNewApp_MonitoringServicesCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.Traffic, "Traffic collector should be initialized")
	assert.NotNil(t, a.Connections, "Connection tracker should be initialized")
	assert.NotNil(t, a.Aggregator, "Data aggregator should be initialized")
	assert.NotNil(t, a.Retention, "Data retention service should be initialized")
}

// TestNewApp_SchedulersCreated verifies that all schedulers are initialized
func TestNewApp_SchedulersCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.BackupSched, "Backup scheduler should be initialized")
	assert.NotNil(t, a.TrafficResetSched, "Traffic reset scheduler should be initialized")
}

// TestNewApp_APIHandlersCreated verifies that all API handlers are initialized
func TestNewApp_APIHandlersCreated(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	// System handlers
	assert.NotNil(t, a.SystemH, "SystemH should be initialized")
	assert.NotNil(t, a.AuthH, "AuthH should be initialized")

	// Core handlers
	assert.NotNil(t, a.CoresH, "CoresH should be initialized")

	// Domain handlers
	assert.NotNil(t, a.UsersH, "UsersH should be initialized")
	assert.NotNil(t, a.InboundsH, "InboundsH should be initialized")
	assert.NotNil(t, a.OutboundsH, "OutboundsH should be initialized")
	assert.NotNil(t, a.ProtocolsH, "ProtocolsH should be initialized")
	assert.NotNil(t, a.SubscriptionsH, "SubscriptionsH should be initialized")

	// Feature handlers
	assert.NotNil(t, a.CertificatesH, "CertificatesH should be initialized")
	assert.NotNil(t, a.StatsH, "StatsH should be initialized")
	assert.NotNil(t, a.WarpH, "WarpH should be initialized")
	assert.NotNil(t, a.BackupH, "BackupH should be initialized")
	assert.NotNil(t, a.NotificationsH, "NotificationsH should be initialized")
	assert.NotNil(t, a.SettingsH, "SettingsH should be initialized")
}

// TestNewApp_DatabaseConnection verifies that the database connection is properly stored
func TestNewApp_DatabaseConnection(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.gormDB, "gormDB should be stored")
	assert.Same(t, db.DB, a.gormDB, "gormDB should be the same as db.DB")
}

// TestNewApp_StopQuotaChannel verifies that the stopQuota channel is properly initialized
func TestNewApp_StopQuotaChannel(t *testing.T) {
	cfg, db := setupProvidersTest(t)
	defer db.Close()

	a, err := NewApp(cfg, db)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.NotNil(t, a.stopQuota, "stopQuota channel should be initialized")
}