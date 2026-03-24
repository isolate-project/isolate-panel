package models

import "time"

// BackupStatus represents the status of a backup operation
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusRestoring BackupStatus = "restoring"
)

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull      BackupType = "full"
	BackupTypeManual    BackupType = "manual"
	BackupTypeScheduled BackupType = "scheduled"
)

// BackupDestination represents the destination of a backup
type BackupDestination string

const (
	BackupDestinationLocal BackupDestination = "local"
)

// Backup represents a backup operation
type Backup struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	Filename       string `gorm:"not null;size:255" json:"filename"`
	FilePath       string `gorm:"not null;size:255" json:"file_path"`
	FileSizeBytes  int64  `json:"file_size_bytes"`
	ChecksumSHA256 string `gorm:"size:64" json:"checksum_sha256"`

	// Type and destination
	BackupType  BackupType        `gorm:"not null;size:20" json:"backup_type"`
	Destination BackupDestination `gorm:"not null;size:20" json:"destination"`

	// Status
	Status       BackupStatus `gorm:"not null;index;size:20;default:'pending'" json:"status"`
	ErrorMessage string       `gorm:"type:text" json:"error_message"`

	// Scheduling
	ScheduleCron string `gorm:"size:50" json:"schedule_cron"` // e.g., "0 3 * * *"

	// Encryption
	EncryptionEnabled bool `gorm:"default:true" json:"encryption_enabled"`

	// Metadata
	BackupSource string `gorm:"type:text" json:"backup_source"` // JSON: what was included
	Metadata     string `gorm:"type:text" json:"metadata"`      // JSON: backup metadata

	// Timing
	DurationMs  int        `json:"duration_ms"`
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName returns the table name for Backup
func (Backup) TableName() string {
	return "backups"
}

// BackupMetadata represents metadata stored in the backup
type BackupMetadata struct {
	Version             string   `json:"version"`               // Backup format version
	IsolatePanelVersion string   `json:"isolate_panel_version"` // Panel version
	DatabaseMigration   string   `json:"database_migration"`    // Latest migration version
	CoresIncluded       []string `json:"cores_included"`        // ["xray", "singbox", "mihomo"]
	Hostname            string   `json:"hostname"`              // Server hostname
	CreatedAt           string   `json:"created_at"`            // ISO 8601 timestamp
}

// BackupSource represents what was included in the backup
type BackupSource struct {
	IncludeDatabase bool `json:"include_database"`
	IncludeCores    bool `json:"include_cores"`
	IncludeCerts    bool `json:"include_certs"`
	IncludeWARP     bool `json:"include_warp"`
	IncludeGeo      bool `json:"include_geo"`
}
