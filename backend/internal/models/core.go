package models

import (
	"time"
)

type Core struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Name       string `gorm:"uniqueIndex;not null" json:"name"` // singbox, xray, mihomo
	Version    string `gorm:"not null" json:"version"`
	IsEnabled  bool   `gorm:"default:true" json:"is_enabled"`
	IsRunning  bool   `gorm:"default:false" json:"is_running"`
	PID        *int   `json:"pid"`
	ConfigPath string `json:"config_path"`
	LogPath    string `json:"log_path"`

	// Statistics
	UptimeSeconds int    `gorm:"default:0" json:"uptime_seconds"`
	RestartCount  int    `gorm:"default:0" json:"restart_count"`
	LastError     string `json:"last_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Core) TableName() string {
	return "cores"
}
