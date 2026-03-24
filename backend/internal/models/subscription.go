package models

import (
	"time"
)

// SubscriptionShortURL model for short URLs
type SubscriptionShortURL struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	ShortCode string    `gorm:"uniqueIndex;not null" json:"short_code"`
	FullURL   string    `gorm:"not null" json:"full_url"`
	CreatedAt time.Time `json:"created_at"`
}

func (SubscriptionShortURL) TableName() string {
	return "subscription_short_urls"
}

// SubscriptionAccess model for access logging
type SubscriptionAccess struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null" json:"user_id"`
	IPAddress      string    `gorm:"not null" json:"ip_address"`
	UserAgent      string    `json:"user_agent"`
	Country        string    `json:"country"`
	Format         string    `json:"format"`
	IsSuspicious   bool      `gorm:"default:false" json:"is_suspicious"`
	ResponseTimeMs int       `gorm:"default:0" json:"response_time_ms"`
	AccessedAt     time.Time `json:"accessed_at"`
}

func (SubscriptionAccess) TableName() string {
	return "subscription_accesses"
}
