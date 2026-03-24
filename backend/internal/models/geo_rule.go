package models

import "time"

// GeoRule represents a GeoIP/GeoSite routing rule
type GeoRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CoreID      uint      `gorm:"not null;index" json:"core_id"`
	Type        string    `gorm:"not null;index;size:20" json:"type"` // "geoip" or "geosite"
	Code        string    `gorm:"not null;size:50" json:"code"`       // country code (US, CN) or category (google, netflix)
	Action      string    `gorm:"not null;size:20" json:"action"`     // "proxy", "direct", "block", "warp"
	Priority    int       `gorm:"default:50;index" json:"priority"`   // 1-100, higher = more priority
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName returns the table name for GeoRule
func (GeoRule) TableName() string {
	return "geo_rules"
}
