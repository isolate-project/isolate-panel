package models

import (
	"time"

	"gorm.io/gorm"
)

// NotificationSeverity represents the severity level of a notification
type NotificationSeverity string

const (
	NotificationSeverityInfo     NotificationSeverity = "info"
	NotificationSeverityWarning  NotificationSeverity = "warning"
	NotificationSeverityError    NotificationSeverity = "error"
	NotificationSeverityCritical NotificationSeverity = "critical"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

// NotificationEventType represents the type of notification event
type NotificationEventType string

const (
	EventTypeQuotaExceeded NotificationEventType = "quota_exceeded"
	EventTypeExpiryWarning NotificationEventType = "expiry_warning"
	EventTypeCertRenewed   NotificationEventType = "cert_renewed"
	EventTypeCoreError     NotificationEventType = "core_error"
	EventTypeFailedLogin   NotificationEventType = "failed_login"
	EventTypeUserCreated   NotificationEventType = "user_created"
	EventTypeUserDeleted   NotificationEventType = "user_deleted"
)

// Notification represents a system notification
type Notification struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	EventType   NotificationEventType `gorm:"not null;index;size:50" json:"event_type"`
	Severity    NotificationSeverity  `gorm:"not null;size:20" json:"severity"`
	Title       string                `gorm:"not null;size:255" json:"title"`
	Message     string                `gorm:"not null;type:text" json:"message"`
	Status      NotificationStatus    `gorm:"not null;index;size:20;default:'pending'" json:"status"`
	RetryCount  int                   `gorm:"default:0" json:"retry_count"`
	MaxRetries  int                   `gorm:"default:3" json:"max_retries"`
	NextRetryAt *time.Time            `json:"next_retry_at"`
	SentAt      *time.Time            `json:"sent_at"`
	Error       string                `gorm:"type:text" json:"error"`
	Metadata    string                `gorm:"type:text" json:"metadata"` // JSON
	CreatedAt   time.Time             `gorm:"index" json:"created_at"`
}

// TableName returns the table name for Notification
func (Notification) TableName() string {
	return "notifications"
}

// NotificationMetadata represents metadata for a notification
type NotificationMetadata struct {
	UserID        uint   `json:"user_id,omitempty"`
	Username      string `json:"username,omitempty"`
	QuotaBytes    int64  `json:"quota_bytes,omitempty"`
	UsedBytes     int64  `json:"used_bytes,omitempty"`
	ExpiryDate    string `json:"expiry_date,omitempty"`
	DaysLeft      int    `json:"days_left,omitempty"`
	CertID        uint   `json:"cert_id,omitempty"`
	Domain        string `json:"domain,omitempty"`
	CertExpiresAt string `json:"cert_expires_at,omitempty"`
	CoreName      string `json:"core_name,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	IPAddress     string `json:"ip_address,omitempty"`
	AttemptCount  int    `json:"attempt_count,omitempty"`
}

// NotificationSettings represents notification settings
type NotificationSettings struct {
	ID               uint   `gorm:"primaryKey;uniqueIndex" json:"id"`
	WebhookEnabled   bool   `json:"webhook_enabled"`
	WebhookURL       string `gorm:"size:255" json:"webhook_url"`
	WebhookSecret    string `gorm:"size:255" json:"-"`
	TelegramEnabled  bool   `json:"telegram_enabled"`
	TelegramBotToken string `gorm:"size:255" json:"-"`
	TelegramChatID   string `gorm:"size:100" json:"telegram_chat_id"`
	// Masked fields for display (computed, not stored)
	WebhookSecretMasked    string `gorm:"-" json:"webhook_secret_masked"`
	TelegramBotTokenMasked string `gorm:"-" json:"telegram_bot_token_masked"`
	// Event toggles
	NotifyQuotaExceeded bool      `json:"notify_quota_exceeded"`
	NotifyExpiryWarning bool      `json:"notify_expiry_warning"`
	NotifyCertRenewed   bool      `json:"notify_cert_renewed"`
	NotifyCoreError     bool      `json:"notify_core_error"`
	NotifyFailedLogin   bool      `json:"notify_failed_login"`
	NotifyUserCreated   bool      `json:"notify_user_created"`
	NotifyUserDeleted   bool      `json:"notify_user_deleted"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// TableName returns the table name for NotificationSettings
func (NotificationSettings) TableName() string {
	return "notification_settings"
}

// AfterFind is a GORM hook that computes masked values after loading from database
func (ns *NotificationSettings) AfterFind(tx *gorm.DB) error {
	if ns.WebhookSecret != "" {
		if len(ns.WebhookSecret) > 4 {
			ns.WebhookSecretMasked = ns.WebhookSecret[:4] + "****"
		} else {
			ns.WebhookSecretMasked = "****"
		}
	}
	if ns.TelegramBotToken != "" {
		if len(ns.TelegramBotToken) > 4 {
			ns.TelegramBotTokenMasked = ns.TelegramBotToken[:4] + "****"
		} else {
			ns.TelegramBotTokenMasked = "****"
		}
	}
	return nil
}

// DefaultNotificationSettings returns default notification settings
func DefaultNotificationSettings() *NotificationSettings {
	return &NotificationSettings{
		WebhookEnabled:      false,
		TelegramEnabled:     false,
		NotifyQuotaExceeded: true,
		NotifyExpiryWarning: true,
		NotifyCertRenewed:   true,
		NotifyCoreError:     true,
		NotifyFailedLogin:   true,
		NotifyUserCreated:   true,
		NotifyUserDeleted:   false,
	}
}
