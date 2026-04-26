package app

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/config"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Certificate{}))
	return db
}

func TestStartSubscriptionListener_Disabled(t *testing.T) {
	a := &App{gormDB: nil}
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{Enabled: false, Port: 0},
	}
	assert.NotPanics(t, func() {
		StartSubscriptionListener(a, cfg)
	})
}

func TestStartSubscriptionListener_ZeroPort(t *testing.T) {
	a := &App{gormDB: nil}
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{Enabled: true, Port: 0},
	}
	assert.NotPanics(t, func() {
		StartSubscriptionListener(a, cfg)
	})
}

func TestStartSubscriptionListener_NegativePort(t *testing.T) {
	a := &App{gormDB: nil}
	cfg := &config.Config{
		Subscription: config.SubscriptionConfig{Enabled: true, Port: -1},
	}
	assert.NotPanics(t, func() {
		StartSubscriptionListener(a, cfg)
	})
}

func TestGetCachedCertificate_NoCertificates(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)
	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetCachedCertificate_EmptyPaths(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)
	cert := models.Certificate{
		Domain:    "test.com",
		AutoRenew: true,
		CertPath:  "",
		KeyPath:   "",
	}
	require.NoError(t, db.Create(&cert).Error)
	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetCachedCertificate_NonExistentPaths(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)
	cert := models.Certificate{
		Domain:    "test.com",
		AutoRenew: true,
		CertPath:  "/nonexistent/cert.pem",
		KeyPath:   "/nonexistent/key.pem",
	}
	require.NoError(t, db.Create(&cert).Error)
	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetCachedCertificate_AutoRenewFalse(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)
	cert := models.Certificate{
		Domain:    "test.com",
		AutoRenew: false,
		CertPath:  "/some/cert.pem",
		KeyPath:   "/some/key.pem",
	}
	require.NoError(t, db.Create(&cert).Error)
	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetCachedCertificate_MultipleCertificates(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)

	cert1 := models.Certificate{
		Domain:    "old.com",
		AutoRenew: true,
		CertPath:  "/nonexistent/old.pem",
		KeyPath:   "/nonexistent/old-key.pem",
		NotAfter:  parseSubTime(t, "2025-01-01"),
	}
	cert2 := models.Certificate{
		Domain:    "new.com",
		AutoRenew: true,
		CertPath:  "/nonexistent/new.pem",
		KeyPath:   "/nonexistent/new-key.pem",
		NotAfter:  parseSubTime(t, "2026-01-01"),
	}
	require.NoError(t, db.Create(&cert1).Error)
	require.NoError(t, db.Create(&cert2).Error)

	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetCachedCertificate_OnlyCertPathEmpty(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)
	cert := models.Certificate{
		Domain:    "test.com",
		AutoRenew: true,
		CertPath:  "",
		KeyPath:   "/some/key.pem",
	}
	require.NoError(t, db.Create(&cert).Error)
	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetCachedCertificate_OnlyKeyPathEmpty(t *testing.T) {
	InvalidateCertCache()
	db := setupSubDB(t)
	cert := models.Certificate{
		Domain:    "test.com",
		AutoRenew: true,
		CertPath:  "/some/cert.pem",
		KeyPath:   "",
	}
	require.NoError(t, db.Create(&cert).Error)
	a := &App{gormDB: db}
	result, err := getCachedCertificate(a)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestInvalidateCertCache(t *testing.T) {
	cachedCert = &tls.Certificate{}
	cachedCertAt = time.Now()
	InvalidateCertCache()
	assert.Nil(t, cachedCert)
	assert.True(t, cachedCertAt.IsZero())
}

func parseSubTime(t *testing.T, s string) time.Time {
	t.Helper()
	if s == "2025-01-01" {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if s == "2026-01-01" {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Time{}
}
