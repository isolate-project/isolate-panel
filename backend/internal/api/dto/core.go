package dto

import (
	"time"
)

// CoreResponse represents the response for a core
type CoreResponse struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	IsEnabled     bool      `json:"is_enabled"`
	IsRunning     bool      `json:"is_running"`
	PID           *int      `json:"pid,omitempty"`
	ConfigPath    string    `json:"config_path,omitempty"`
	LogPath       string    `json:"log_path,omitempty"`
	APIPort       int       `json:"api_port"`
	APIKeyHint    string    `json:"api_key_hint,omitempty"`
	UptimeSeconds int       `json:"uptime_seconds"`
	RestartCount  int       `json:"restart_count"`
	LastError     string    `json:"last_error,omitempty"`
	HealthStatus  string    `json:"health_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CoreStatusResponse represents the detailed runtime status of a core
type CoreStatusResponse struct {
	Name           string `json:"name"`
	IsRunning      bool   `json:"is_running"`
	IsEnabled      bool   `json:"is_enabled"`
	PID            *int   `json:"pid,omitempty"`
	UptimeSeconds  int    `json:"uptime_seconds"`
	RestartCount   int    `json:"restart_count"`
	LastError      string `json:"last_error,omitempty"`
	HealthStatus   string `json:"health_status"`
}
