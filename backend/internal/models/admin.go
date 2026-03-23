package models

import (
	"time"
)

type Admin struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"not null" json:"-"`
	Email        string     `json:"email"`
	TOTPSecret   string     `json:"-"`
	TOTPEnabled  bool       `gorm:"default:false" json:"totp_enabled"`
	IsSuperAdmin bool       `gorm:"default:false" json:"is_super_admin"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
}

func (Admin) TableName() string {
	return "admins"
}

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AdminID   uint      `gorm:"not null" json:"admin_id"`
	TokenHash string    `gorm:"not null" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`

	Admin Admin `gorm:"foreignKey:AdminID" json:"-"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

type LoginAttempt struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	IPAddress   string    `gorm:"not null" json:"ip_address"`
	Username    string    `json:"username"`
	Success     bool      `gorm:"default:false" json:"success"`
	AttemptedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"attempted_at"`
	UserAgent   string    `json:"user_agent"`
}

func (LoginAttempt) TableName() string {
	return "login_attempts"
}
