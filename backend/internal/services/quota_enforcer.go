package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// QuotaEnforcer enforces user quotas
// Uses graceful reload for all cores (simpler than per-core strategies)
type QuotaEnforcer struct {
	db                  *gorm.DB
	configService       *ConfigService
	notificationService *NotificationService
	mu                  sync.Mutex
}

// NewQuotaEnforcer creates a new quota enforcer
func NewQuotaEnforcer(db *gorm.DB, configService *ConfigService, notificationService *NotificationService) *QuotaEnforcer {
	return &QuotaEnforcer{
		db:                  db,
		configService:       configService,
		notificationService: notificationService,
	}
}

// CheckAndEnforce checks all users and enforces quotas
func (qe *QuotaEnforcer) CheckAndEnforce(ctx context.Context) {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	// Get all users with traffic limits
	var users []models.User
	if err := qe.db.Where("traffic_limit_bytes > 0 AND is_active = ?", true).Find(&users).Error; err != nil {
		return
	}

	for i := range users {
		user := &users[i]
		if user.TrafficLimitBytes != nil && user.TrafficUsedBytes >= *user.TrafficLimitBytes {
			qe.DisableUser(ctx, user)
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

	// Trigger config regeneration and graceful reload for all cores
	// This will apply the user disable across all inbounds
	cores := []string{"singbox", "xray", "mihomo"}
	for _, coreName := range cores {
		_ = qe.configService.RegenerateAndReload(coreName)
	}

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

	// Trigger config regeneration and graceful reload for all cores
	cores := []string{"singbox", "xray", "mihomo"}
	for _, coreName := range cores {
		_ = qe.configService.RegenerateAndReload(coreName)
	}

	return nil
}

// ResetUserTraffic resets user's traffic counter
func (qe *QuotaEnforcer) ResetUserTraffic(userID uint) error {
	var user models.User
	if err := qe.db.First(&user, userID).Error; err != nil {
		return err
	}

	user.TrafficUsedBytes = 0
	return qe.db.Save(&user).Error
}
