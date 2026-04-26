package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

type notificationJob struct {
	notification *models.Notification
	cleanup      bool
}

// NotificationService manages system notifications
type NotificationService struct {
	db               *gorm.DB
	webhookNotifier  *WebhookNotifier
	telegramNotifier *TelegramNotifier
	settings         *models.NotificationSettings
	maxNotifications int
	jobChan          chan notificationJob
	wg               sync.WaitGroup
	quit             chan struct{}
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *gorm.DB, webhookURL, webhookSecret, telegramToken, telegramChatID string) *NotificationService {
	settings := models.DefaultNotificationSettings()
	if webhookURL != "" {
		settings.WebhookEnabled = true
		settings.WebhookURL = webhookURL
		settings.WebhookSecret = webhookSecret
	}
	if telegramToken != "" && telegramChatID != "" {
		settings.TelegramEnabled = true
		settings.TelegramBotToken = telegramToken
		settings.TelegramChatID = telegramChatID
	}

	return &NotificationService{
		db: db,
		webhookNotifier: &WebhookNotifier{
			url:    webhookURL,
			secret: webhookSecret,
		},
		telegramNotifier: &TelegramNotifier{
			botToken: telegramToken,
			chatID:   telegramChatID,
		},
		settings:         settings,
		maxNotifications: 100,
	}
}

// Initialize loads settings from database
func (s *NotificationService) Initialize() error {
	var settings models.NotificationSettings
	if err := s.db.First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default settings
			return s.db.Create(models.DefaultNotificationSettings()).Error
		}
		return err
	}

	s.settings = &settings
	s.webhookNotifier.url = settings.WebhookURL
	s.webhookNotifier.secret = settings.WebhookSecret
	s.webhookNotifier.enabled = settings.WebhookEnabled
	s.telegramNotifier.botToken = settings.TelegramBotToken
	s.telegramNotifier.chatID = settings.TelegramChatID
	s.telegramNotifier.enabled = settings.TelegramEnabled

	return nil
}

func (s *NotificationService) Start() {
	s.jobChan = make(chan notificationJob, 100)
	s.quit = make(chan struct{})
	s.wg.Add(1)
	go s.worker()
	s.wg.Add(1)
	go s.startRetryWorker()
}

func (s *NotificationService) worker() {
	defer s.wg.Done()
	for {
		select {
		case job := <-s.jobChan:
			s.processJob(job)
		case <-s.quit:
			for {
				select {
				case job := <-s.jobChan:
					s.processJob(job)
				default:
					return
				}
			}
		}
	}
}

func (s *NotificationService) startRetryWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			var pending []models.Notification
			s.db.Where("status = ? AND next_retry_at <= ?", "failed", time.Now()).Find(&pending)
			for _, n := range pending {
				select {
				case s.jobChan <- notificationJob{notification: &n, cleanup: false}:
					logger.Log.Debug().Uint("id", n.ID).Msg("Retrying notification")
				default:
				}
			}
		}
	}
}

func (s *NotificationService) processJob(job notificationJob) {
	if job.cleanup {
		s.cleanupOldNotifications()
	}
	s.sendNotification(job.notification)
}

func (s *NotificationService) Stop() {
	close(s.quit)
	s.wg.Wait()
}

// Send sends a notification
func (s *NotificationService) Send(eventType models.NotificationEventType, severity models.NotificationSeverity, title, message string, metadata map[string]interface{}) error {
	// Check if event type is enabled
	if !s.isEventTypeEnabled(eventType) {
		return nil
	}

	// Create notification record
	notification := &models.Notification{
		EventType:  eventType,
		Severity:   severity,
		Title:      title,
		Message:    message,
		Status:     models.NotificationStatusPending,
		MaxRetries: 3,
	}

	if metadata != nil {
		metaJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		notification.Metadata = string(metaJSON)
	}

	if err := s.db.Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	if s.jobChan != nil {
		select {
		case s.jobChan <- notificationJob{notification: notification, cleanup: true}:
		default:
			logger.Log.Warn().Str("type", string(notification.EventType)).Msg("Notification channel full, dropping job")
		}
	}

	return nil
}

// isEventTypeEnabled checks if event type is enabled in settings
func (s *NotificationService) isEventTypeEnabled(eventType models.NotificationEventType) bool {
	switch eventType {
	case models.EventTypeQuotaExceeded:
		return s.settings.NotifyQuotaExceeded
	case models.EventTypeExpiryWarning:
		return s.settings.NotifyExpiryWarning
	case models.EventTypeCertRenewed:
		return s.settings.NotifyCertRenewed
	case models.EventTypeCoreError:
		return s.settings.NotifyCoreError
	case models.EventTypeFailedLogin:
		return s.settings.NotifyFailedLogin
	case models.EventTypeUserCreated:
		return s.settings.NotifyUserCreated
	case models.EventTypeUserDeleted:
		return s.settings.NotifyUserDeleted
	default:
		return true
	}
}

// sendNotification sends notification via configured channels
func (s *NotificationService) sendNotification(notification *models.Notification) {
	var metadata models.NotificationMetadata
	if notification.Metadata != "" {
		json.Unmarshal([]byte(notification.Metadata), &metadata)
	}

	errors := make([]string, 0)

	// Send via webhook
	if s.settings.WebhookEnabled {
		if err := s.webhookNotifier.Send(notification); err != nil {
			errors = append(errors, fmt.Sprintf("webhook: %v", err))
		}
	}

	// Send via Telegram
	if s.settings.TelegramEnabled {
		if err := s.telegramNotifier.Send(notification); err != nil {
			errors = append(errors, fmt.Sprintf("telegram: %v", err))
		}
	}

	// Update notification status
	now := time.Now()
	if len(errors) == 0 {
		notification.Status = models.NotificationStatusSent
		notification.SentAt = &now
	} else {
		notification.Status = models.NotificationStatusFailed
		notification.Error = fmt.Sprintf("Failed: %s", errors)
		notification.RetryCount++
		if notification.RetryCount < notification.MaxRetries {
			nextRetry := now.Add(time.Duration(notification.RetryCount*5) * time.Minute)
			notification.NextRetryAt = &nextRetry
		}
	}

	s.db.Save(notification)
}

// cleanupOldNotifications removes old notifications keeping only maxNotifications
func (s *NotificationService) cleanupOldNotifications() {
	var count int64
	s.db.Model(&models.Notification{}).Count(&count)

	if count > int64(s.maxNotifications) {
		var oldNotifications []models.Notification
		s.db.Order("created_at ASC").
			Limit(int(count) - s.maxNotifications).
			Find(&oldNotifications)

		for _, n := range oldNotifications {
			s.db.Delete(&n)
		}
	}
}

// NotifyQuotaExceeded sends quota exceeded notification
func (s *NotificationService) NotifyQuotaExceeded(user *models.User) {
	metadata := map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
	}

	s.Send(
		models.EventTypeQuotaExceeded,
		models.NotificationSeverityWarning,
		"User quota exceeded",
		fmt.Sprintf("User %s exceeded their traffic quota", user.Username),
		metadata,
	)
}

// NotifyQuotaWarning sends a quota threshold warning notification (e.g., 80% or 90%)
func (s *NotificationService) NotifyQuotaWarning(user *models.User, percentUsed int) {
	usedStr := formatBytes(user.TrafficUsedBytes)
	limitStr := ""
	if user.TrafficLimitBytes != nil {
		limitStr = formatBytes(*user.TrafficLimitBytes)
	}

	metadata := map[string]interface{}{
		"user_id":      user.ID,
		"username":     user.Username,
		"percent_used": percentUsed,
		"used_bytes":   user.TrafficUsedBytes,
	}

	s.Send(
		models.EventTypeQuotaExceeded, // reuse same event type for filtering
		models.NotificationSeverityWarning,
		fmt.Sprintf("User quota warning: %d%%", percentUsed),
		fmt.Sprintf("User %s has used %d%% of traffic quota (%s / %s)", user.Username, percentUsed, usedStr, limitStr),
		metadata,
	)
}

// NotifyExpiryWarning sends expiry warning notification
func (s *NotificationService) NotifyExpiryWarning(user *models.User, daysLeft int) {
	expiryStr := ""
	if user.ExpiryDate != nil {
		expiryStr = user.ExpiryDate.Format("2006-01-02")
	}

	metadata := map[string]interface{}{
		"user_id":     user.ID,
		"username":    user.Username,
		"expiry_date": expiryStr,
		"days_left":   daysLeft,
	}

	s.Send(
		models.EventTypeExpiryWarning,
		models.NotificationSeverityWarning,
		fmt.Sprintf("User expires in %d days", daysLeft),
		fmt.Sprintf("User %s account expires in %d days", user.Username, daysLeft),
		metadata,
	)
}

// NotifyCertRenewed sends certificate renewed notification
func (s *NotificationService) NotifyCertRenewed(cert *models.Certificate) {
	metadata := map[string]interface{}{
		"cert_id":   cert.ID,
		"domain":    cert.Domain,
		"not_after": cert.NotAfter.Format("2006-01-02"),
	}

	s.Send(
		models.EventTypeCertRenewed,
		models.NotificationSeverityInfo,
		"Certificate renewed",
		fmt.Sprintf("TLS certificate for %s has been renewed", cert.Domain),
		metadata,
	)
}

// NotifyCoreError sends core error notification
func (s *NotificationService) NotifyCoreError(coreName string, err error) {
	metadata := map[string]interface{}{
		"core_name":     coreName,
		"error_message": err.Error(),
	}

	s.Send(
		models.EventTypeCoreError,
		models.NotificationSeverityCritical,
		fmt.Sprintf("Core error: %s", coreName),
		fmt.Sprintf("Core %s encountered an error: %v", coreName, err),
		metadata,
	)
}

// NotifyFailedLogin sends failed login notification
func (s *NotificationService) NotifyFailedLogin(ip, username string, attempts int) {
	metadata := map[string]interface{}{
		"ip_address":    ip,
		"username":      username,
		"attempt_count": attempts,
	}

	s.Send(
		models.EventTypeFailedLogin,
		models.NotificationSeverityWarning,
		"Multiple failed login attempts",
		fmt.Sprintf("Failed login attempts for user %s from IP %s: %d attempts", username, ip, attempts),
		metadata,
	)
}

// NotifyUserCreated sends user created notification
func (s *NotificationService) NotifyUserCreated(user *models.User) {
	metadata := map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
	}

	s.Send(
		models.EventTypeUserCreated,
		models.NotificationSeverityInfo,
		"User created",
		fmt.Sprintf("New user %s has been created", user.Username),
		metadata,
	)
}

// NotifyUserDeleted sends user deleted notification
func (s *NotificationService) NotifyUserDeleted(user *models.User) {
	metadata := map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
	}

	s.Send(
		models.EventTypeUserDeleted,
		models.NotificationSeverityInfo,
		"User deleted",
		fmt.Sprintf("User %s has been deleted", user.Username),
		metadata,
	)
}

// GetSettings returns current notification settings
func (s *NotificationService) GetSettings() (*models.NotificationSettings, error) {
	var settings models.NotificationSettings
	if err := s.db.First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateSettings updates notification settings
func (s *NotificationService) UpdateSettings(settings *models.NotificationSettings) error {
	if err := s.db.Save(settings).Error; err != nil {
		return err
	}

	s.settings = settings
	s.webhookNotifier.url = settings.WebhookURL
	s.webhookNotifier.secret = settings.WebhookSecret
	s.webhookNotifier.enabled = settings.WebhookEnabled
	s.telegramNotifier.botToken = settings.TelegramBotToken
	s.telegramNotifier.chatID = settings.TelegramChatID
	s.telegramNotifier.enabled = settings.TelegramEnabled

	return nil
}

// ListNotifications returns list of notifications
func (s *NotificationService) ListNotifications(limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	err := s.db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

// GetNotification returns a single notification
func (s *NotificationService) GetNotification(id uint) (*models.Notification, error) {
	var notification models.Notification
	err := s.db.First(&notification, id).Error
	return &notification, err
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(id uint) error {
	return s.db.Delete(&models.Notification{}, id).Error
}

// formatBytes formats bytes to human readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
