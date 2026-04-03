package services

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// AuditService records admin actions for compliance and debugging.
type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Log writes a single audit entry.  details may be any JSON-serialisable value
// (struct, map, nil).
func (s *AuditService) Log(adminID uint, action, resource string, resourceID *uint, details any, ip string) {
	entry := models.AuditLog{
		AdminID:    adminID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ip,
		CreatedAt:  time.Now(),
	}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			entry.Details = string(b)
		}
	}
	// Fire-and-forget: audit failure must not block the request
	s.db.Create(&entry)
}

type AuditListOptions struct {
	Action   string
	AdminID  uint
	Page     int
	PageSize int
}

type AuditListResult struct {
	Logs  []models.AuditLog `json:"logs"`
	Total int64             `json:"total"`
}

// List returns paginated audit log entries.
func (s *AuditService) List(opts AuditListOptions) (AuditListResult, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 || opts.PageSize > 200 {
		opts.PageSize = 50
	}

	q := s.db.Model(&models.AuditLog{})
	if opts.Action != "" {
		q = q.Where("action = ?", opts.Action)
	}
	if opts.AdminID > 0 {
		q = q.Where("admin_id = ?", opts.AdminID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return AuditListResult{}, err
	}

	var logs []models.AuditLog
	offset := (opts.Page - 1) * opts.PageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(opts.PageSize).Find(&logs).Error; err != nil {
		return AuditListResult{}, err
	}

	return AuditListResult{Logs: logs, Total: total}, nil
}

// Purge deletes audit entries older than retentionDays.  Called by DataRetentionService.
func (s *AuditService) Purge(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return s.db.Where("created_at < ?", cutoff).Delete(&models.AuditLog{}).Error
}
