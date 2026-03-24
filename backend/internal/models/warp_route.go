package models

import "time"

// WarpRoute represents a WARP routing rule
type WarpRoute struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ResourceType  string    `gorm:"not null;index;size:20" json:"resource_type"` // "domain", "ip", "cidr"
	ResourceValue string    `gorm:"not null;size:255" json:"resource_value"`
	Description   string    `gorm:"type:text" json:"description"`
	CoreID        uint      `gorm:"not null;index" json:"core_id"`
	Priority      int       `gorm:"default:50;index" json:"priority"` // 1-100, higher = more priority
	IsEnabled     bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName returns the table name for WarpRoute
func (WarpRoute) TableName() string {
	return "warp_routes"
}
