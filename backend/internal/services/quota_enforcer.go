package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// QuotaEnforcer enforces user quotas
// Uses graceful reload for affected cores only (targeted approach)
type QuotaEnforcer struct {
	db                  *gorm.DB
	configService       *ConfigService
	notificationService *NotificationService
	mu                  sync.Mutex
	log                 zerolog.Logger

	// Track which users have already been warned at each threshold to avoid spam
	warned80 map[uint]bool
	warned90 map[uint]bool
}

// NewQuotaEnforcer creates a new quota enforcer
func NewQuotaEnforcer(db *gorm.DB, configService *ConfigService, notificationService *NotificationService) *QuotaEnforcer {
	return &QuotaEnforcer{
		db:                  db,
		configService:       configService,
		notificationService: notificationService,
		log:                 logger.WithComponent("quota_enforcer"),
		warned80:            make(map[uint]bool),
		warned90:            make(map[uint]bool),
	}
}

// CheckAndEnforce checks all users and enforces quotas
func (qe *QuotaEnforcer) CheckAndEnforce(ctx context.Context) {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	// Get all users with traffic limits
	var users []models.User
	if err := qe.db.Where("traffic_limit_bytes > 0 AND is_active = ?", true).Find(&users).Error; err != nil {
		qe.log.Error().Err(err).Msg("Failed to query users for quota enforcement")
		return
	}

	for i := range users {
		user := &users[i]
		if user.TrafficLimitBytes == nil {
			continue
		}

		limit := *user.TrafficLimitBytes
		if limit <= 0 {
			continue
		}

		percentUsed := int(float64(user.TrafficUsedBytes) / float64(limit) * 100)

		// Check thresholds in order: 100% → 90% → 80%
		if user.TrafficUsedBytes >= limit {
			// 100% — disable user
			if err := qe.DisableUser(ctx, user); err != nil {
				qe.log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to disable user")
			}
			// Clear warning state (will be reset on re-enable)
			delete(qe.warned80, user.ID)
			delete(qe.warned90, user.ID)
		} else if percentUsed >= 90 && !qe.warned90[user.ID] {
			// 90% threshold warning
			qe.warned90[user.ID] = true
			qe.log.Info().Uint("user_id", user.ID).Str("username", user.Username).Int("percent", percentUsed).Msg("User approaching quota limit (90%)")
			if qe.notificationService != nil {
				qe.notificationService.NotifyQuotaWarning(user, percentUsed)
			}
		} else if percentUsed >= 80 && !qe.warned80[user.ID] {
			// 80% threshold warning
			qe.warned80[user.ID] = true
			qe.log.Info().Uint("user_id", user.ID).Str("username", user.Username).Int("percent", percentUsed).Msg("User approaching quota limit (80%)")
			if qe.notificationService != nil {
				qe.notificationService.NotifyQuotaWarning(user, percentUsed)
			}
		}
	}
}

// DisableUser disables a user who exceeded quota
func (qe *QuotaEnforcer) DisableUser(ctx context.Context, user *models.User) error {
	// Mark as inactive in DB
	user.IsActive = false
	if err := qe.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to mark user as inactive: %w", err)
	}

	qe.log.Warn().Uint("user_id", user.ID).Str("username", user.Username).Msg("User disabled due to quota exceeded")

	// Targeted reload: only regenerate config for cores that have this user's inbounds
	qe.reloadAffectedCores(user.ID)

	// Send notification
	if qe.notificationService != nil {
		qe.notificationService.NotifyQuotaExceeded(user)
	}

	return nil
}

// EnableUser re-enables a user
func (qe *QuotaEnforcer) EnableUser(ctx context.Context, user *models.User) error {
	user.IsActive = true
	if err := qe.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to enable user: %w", err)
	}

	qe.log.Info().Uint("user_id", user.ID).Str("username", user.Username).Msg("User re-enabled")

	// Reset warning state
	delete(qe.warned80, user.ID)
	delete(qe.warned90, user.ID)

	// Targeted reload: regenerate for affected cores
	qe.reloadAffectedCores(user.ID)

	return nil
}

// ResetUserTraffic resets user's traffic counter
func (qe *QuotaEnforcer) ResetUserTraffic(userID uint) error {
	var user models.User
	if err := qe.db.First(&user, userID).Error; err != nil {
		return err
	}

	user.TrafficUsedBytes = 0
	if err := qe.db.Save(&user).Error; err != nil {
		return err
	}

	// Reset warning state
	delete(qe.warned80, userID)
	delete(qe.warned90, userID)

	qe.log.Info().Uint("user_id", userID).Msg("User traffic counter reset")

	return nil
}

// ResetAllTraffic resets traffic counters for all users (used by scheduled reset)
func (qe *QuotaEnforcer) ResetAllTraffic() error {
	err := qe.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("UPDATE users SET traffic_used_bytes = 0")
		if result.Error != nil {
			return fmt.Errorf("failed to reset traffic: %w", result.Error)
		}
		qe.log.Info().Int64("affected", result.RowsAffected).Msg("Scheduled traffic reset: all users reset")
		return nil
	})
	if err != nil {
		return err
	}
	// Clear warning state after successful commit
	qe.warned80 = make(map[uint]bool)
	qe.warned90 = make(map[uint]bool)
	return nil
}

// reloadAffectedCores regenerates and reloads only the cores that have inbounds
// assigned to the given user.
func (qe *QuotaEnforcer) reloadAffectedCores(userID uint) {
	if qe.configService == nil {
		qe.log.Debug().Uint("user_id", userID).Msg("ConfigService not available, skipping core reload")
		return
	}

	// Find which cores are affected by this user's inbounds
	type CoreNameResult struct {
		Name string
	}
	var coreNames []CoreNameResult

	err := qe.db.Table("cores").
		Select("DISTINCT cores.name").
		Joins("JOIN inbounds ON inbounds.core_id = cores.id").
		Joins("JOIN user_inbound_mapping ON user_inbound_mapping.inbound_id = inbounds.id").
		Where("user_inbound_mapping.user_id = ?", userID).
		Scan(&coreNames).Error

	if err != nil {
		qe.log.Error().Err(err).Uint("user_id", userID).Msg("Failed to determine affected cores, falling back to full reload")
		// Fallback: reload all cores
		for _, name := range []string{"singbox", "xray", "mihomo"} {
			_ = qe.configService.RegenerateAndReload(name)
		}
		return
	}

	if len(coreNames) == 0 {
		qe.log.Debug().Uint("user_id", userID).Msg("No affected cores found for user")
		return
	}

	for _, cn := range coreNames {
		qe.log.Debug().Str("core", cn.Name).Uint("user_id", userID).Msg("Reloading affected core")
		if err := qe.configService.RegenerateAndReload(cn.Name); err != nil {
			qe.log.Error().Err(err).Str("core", cn.Name).Msg("Failed to reload core")
		}
	}
}
