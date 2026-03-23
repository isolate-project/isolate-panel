package models

import (
	"time"
)

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Email    string `json:"email"`

	// Universal Credentials
	UUID              string  `gorm:"uniqueIndex;not null" json:"uuid"`
	Password          string  `gorm:"not null" json:"password"` // plaintext in MVP
	Token             *string `gorm:"uniqueIndex" json:"token"`
	SubscriptionToken string  `gorm:"uniqueIndex;not null" json:"subscription_token"`

	// Quotas
	TrafficLimitBytes *int64     `json:"traffic_limit_bytes"` // NULL = unlimited
	TrafficUsedBytes  int64      `gorm:"default:0" json:"traffic_used_bytes"`
	ExpiryDate        *time.Time `json:"expiry_date"` // NULL = no expiry

	// Status
	IsActive bool `gorm:"default:true" json:"is_active"`
	IsOnline bool `gorm:"default:false" json:"is_online"`

	// Metadata
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastConnectedAt  *time.Time `json:"last_connected_at"`
	CreatedByAdminID *uint      `json:"created_by_admin_id"`

	// Relations
	CreatedByAdmin *Admin `gorm:"foreignKey:CreatedByAdminID" json:"created_by_admin,omitempty"`
}

func (User) TableName() string {
	return "users"
}

type UserInboundMapping struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	InboundID uint      `gorm:"not null" json:"inbound_id"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Inbound *Inbound `gorm:"foreignKey:InboundID" json:"inbound,omitempty"`
}

func (UserInboundMapping) TableName() string {
	return "user_inbound_mapping"
}
