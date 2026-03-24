package models

import (
	"time"
)

// TrafficStats stores traffic statistics per user per inbound
type TrafficStats struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	UserID    uint `gorm:"not null;index" json:"user_id"`
	InboundID uint `gorm:"not null;index" json:"inbound_id"`
	CoreID    uint `gorm:"not null;index" json:"core_id"`

	// Traffic in bytes
	Upload   uint64 `gorm:"default:0" json:"upload"`
	Download uint64 `gorm:"default:0" json:"download"`
	Total    uint64 `gorm:"default:0" json:"total"`

	// Timestamp
	RecordedAt time.Time `gorm:"not null;index" json:"recorded_at"`

	// Granularity: "raw" (per-minute), "hourly", "daily"
	Granularity string `gorm:"default:'raw';index" json:"granularity"`

	CreatedAt time.Time `json:"created_at"`
}

func (TrafficStats) TableName() string {
	return "traffic_stats"
}

// ActiveConnection represents an active user connection
type ActiveConnection struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	InboundID uint   `gorm:"not null;index" json:"inbound_id"`
	CoreID    uint   `gorm:"not null;index" json:"core_id"`
	CoreName  string `gorm:"not null" json:"core_name"`

	// Connection details
	SourceIP        string `json:"source_ip"`
	SourcePort      int    `json:"source_port"`
	DestinationIP   string `json:"destination_ip"`
	DestinationPort int    `json:"destination_port"`

	// Timing
	StartedAt    time.Time `gorm:"not null;index" json:"started_at"`
	LastActivity time.Time `json:"last_activity"`

	// Traffic for this connection (bytes)
	Upload   uint64 `gorm:"default:0" json:"upload"`
	Download uint64 `gorm:"default:0" json:"download"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ActiveConnection) TableName() string {
	return "active_connections"
}
