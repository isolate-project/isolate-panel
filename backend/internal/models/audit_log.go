package models

import "time"

// AuditLog records admin actions on critical resources.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey"        json:"id"`
	AdminID    uint      `gorm:"index;not null"    json:"admin_id"`
	Action     string    `gorm:"not null"          json:"action"`     // e.g. "user.create"
	Resource   string    `gorm:"not null"          json:"resource"`   // e.g. "user"
	ResourceID *uint     `                         json:"resource_id"`
	Details    string    `                         json:"details"`    // JSON payload
	IPAddress  string    `                         json:"ip_address"`
	CreatedAt  time.Time `gorm:"index"             json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
