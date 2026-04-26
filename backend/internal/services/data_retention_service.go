package services

import (
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// Default retention periods (days)
const (
	DefaultRawRetentionDays    = 7
	DefaultHourlyRetentionDays = 90
	DefaultConnStaleMinutes    = 60
)

// DataRetentionService manages data retention policies.
// Periodically cleans up old traffic stats, stale connections, and expired notifications.
// Retention periods are configurable via SettingsService.
type DataRetentionService struct {
	db       *gorm.DB
	settings *SettingsService
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	log      zerolog.Logger
}

// NewDataRetentionService creates a new data retention service
func NewDataRetentionService(db *gorm.DB, interval time.Duration, settings ...*SettingsService) *DataRetentionService {
	if interval == 0 {
		interval = 24 * time.Hour // Default: run once per day
	}

	var settingsSvc *SettingsService
	if len(settings) > 0 {
		settingsSvc = settings[0]
	}

	return &DataRetentionService{
		db:       db,
		settings: settingsSvc,
		interval: interval,
		stopChan: make(chan struct{}),
		log:      logger.WithComponent("data_retention"),
	}
}

// Start starts the retention service
func (dr *DataRetentionService) Start() {
	dr.log.Info().Dur("interval", dr.interval).Msg("Starting data retention service")

	// Run immediately on start to clean up any accumulated data
	dr.cleanupOldData()

	dr.wg.Add(1)
	go func() {
		defer dr.wg.Done()
		ticker := time.NewTicker(dr.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				dr.cleanupOldData()
			case <-dr.stopChan:
				dr.log.Info().Msg("Data retention service stopped")
				return
			}
		}
	}()
}

// Stop stops the retention service
func (dr *DataRetentionService) Stop() {
	close(dr.stopChan)
	dr.wg.Wait()
}

// cleanupOldData removes old data according to retention policies
func (dr *DataRetentionService) cleanupOldData() {
	now := time.Now()

	rawDays := dr.getRetentionDays("retention_raw_days", DefaultRawRetentionDays)
	hourlyDays := dr.getRetentionDays("retention_hourly_days", DefaultHourlyRetentionDays)
	connStaleMinutes := dr.getRetentionDays("retention_conn_stale_minutes", DefaultConnStaleMinutes)

	dr.log.Debug().
		Int("raw_days", rawDays).
		Int("hourly_days", hourlyDays).
		Int("conn_stale_minutes", connStaleMinutes).
		Msg("Running data retention cleanup")

	totalDeleted := int64(0)

	err := dr.db.Transaction(func(tx *gorm.DB) error {
		// Raw stats: keep N days (default 7)
		rawCutoff := now.AddDate(0, 0, -rawDays)
		dr.log.Debug().Time("raw_cutoff", rawCutoff).Msg("Deleting raw stats")
		result := tx.Where("granularity = ? AND recorded_at < ?", "raw", rawCutoff).
			Delete(&models.TrafficStats{})
		if result.Error != nil {
			dr.log.Debug().Err(result.Error).Msg("Raw stats delete error")
			if !isTableNotFoundError(result.Error) {
				return result.Error
			}
		}
		if result.RowsAffected > 0 {
			dr.log.Info().Int64("deleted", result.RowsAffected).Int("older_than_days", rawDays).Msg("Cleaned up raw traffic stats")
			totalDeleted += result.RowsAffected
		}

		// Hourly stats: keep N days (default 90)
		hourlyCutoff := now.AddDate(0, 0, -hourlyDays)
		result = tx.Where("granularity = ? AND recorded_at < ?", "hourly", hourlyCutoff).
			Delete(&models.TrafficStats{})
		if result.Error != nil {
			dr.log.Debug().Err(result.Error).Msg("Hourly stats delete error")
			if !isTableNotFoundError(result.Error) {
				return result.Error
			}
		}
		if result.RowsAffected > 0 {
			dr.log.Info().Int64("deleted", result.RowsAffected).Int("older_than_days", hourlyDays).Msg("Cleaned up hourly traffic stats")
			totalDeleted += result.RowsAffected
		}

		// Daily stats: keep indefinitely (no cleanup)

		// Active connections: cleanup stale (default 60 minutes without activity)
		connCutoff := now.Add(-time.Duration(connStaleMinutes) * time.Minute)
		result = tx.Where("last_activity < ?", connCutoff).
			Delete(&models.ActiveConnection{})
		if result.Error != nil {
			dr.log.Debug().Err(result.Error).Msg("Active connections delete error")
			if !isTableNotFoundError(result.Error) {
				return result.Error
			}
		}
		if result.RowsAffected > 0 {
			dr.log.Info().Int64("deleted", result.RowsAffected).Int("older_than_minutes", connStaleMinutes).Msg("Cleaned up stale connections")
			totalDeleted += result.RowsAffected
		}

		// Notification logs: cleanup read/resolved notifications older than 30 days
		notifCutoff := now.AddDate(0, 0, -30)
		result = tx.Where("created_at < ? AND status = ?", notifCutoff, "read").
			Delete(&models.Notification{})
		if result.Error != nil {
			dr.log.Debug().Err(result.Error).Msg("Notification logs delete error")
			if !isTableNotFoundError(result.Error) {
				return result.Error
			}
		}
		if result.RowsAffected > 0 {
			dr.log.Info().Int64("deleted", result.RowsAffected).Msg("Cleaned up old notification logs")
			totalDeleted += result.RowsAffected
		}

		// Expired refresh tokens: cleanup tokens past their expiry
		result = tx.Where("expires_at < ?", now).Delete(&models.RefreshToken{})
		if result.Error != nil {
			dr.log.Debug().Err(result.Error).Msg("Refresh tokens delete error")
			if !isTableNotFoundError(result.Error) {
				return result.Error
			}
		}
		if result.RowsAffected > 0 {
			dr.log.Info().Int64("deleted", result.RowsAffected).Msg("Cleaned up expired refresh tokens")
			totalDeleted += result.RowsAffected
		}

		// Subscription access logs: keep 90 days
		subAccessCutoff := now.AddDate(0, 0, -90)
		result = tx.Where("created_at < ?", subAccessCutoff).Delete(&models.SubscriptionAccess{})
		if result.Error != nil {
			dr.log.Debug().Err(result.Error).Msg("Subscription access logs delete error")
			if !isTableNotFoundError(result.Error) {
				return result.Error
			}
		}
		if result.RowsAffected > 0 {
			dr.log.Info().Int64("deleted", result.RowsAffected).Msg("Cleaned up old subscription access logs")
			totalDeleted += result.RowsAffected
		}

		return nil
	})

	if err != nil {
		dr.log.Error().Err(err).Msg("Failed to cleanup expired data")
	}

	if totalDeleted > 0 {
		dr.log.Info().Int64("total_deleted", totalDeleted).Msg("Data retention cleanup completed")
	} else {
		dr.log.Debug().Msg("Data retention cleanup completed — nothing to delete")
	}
}

func isTableNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "no such table") || contains(errStr, "table") && contains(errStr, "does not exist") || contains(errStr, "no such column")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getRetentionDays reads a retention setting from SettingsService with a fallback default.
func (dr *DataRetentionService) getRetentionDays(key string, defaultValue int) int {
	if dr.settings == nil {
		return defaultValue
	}

	value, err := dr.settings.GetSettingValue(key)
	if err != nil {
		return defaultValue
	}

	days, err := strconv.Atoi(value)
	if err != nil || days <= 0 {
		return defaultValue
	}

	return days
}
