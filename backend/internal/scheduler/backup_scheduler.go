package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"gorm.io/gorm"
)

// BackupScheduler manages scheduled backup operations
type BackupScheduler struct {
	db            *gorm.DB
	backupService *services.BackupService
	cron          *cron.Cron
	mu            sync.RWMutex
	jobEntry      cron.EntryID
}

// NewBackupScheduler creates a new backup scheduler
func NewBackupScheduler(db *gorm.DB, backupService *services.BackupService) *BackupScheduler {
	return &BackupScheduler{
		db:            db,
		backupService: backupService,
		cron:          cron.New(cron.WithSeconds()),
	}
}

// Initialize loads and starts the scheduled backup job
func (s *BackupScheduler) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current schedule from database
	schedule, err := s.backupService.GetSchedule()
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule != "" {
		if err := s.scheduleBackup(schedule); err != nil {
			return fmt.Errorf("failed to schedule backup: %w", err)
		}
	}

	s.cron.Start()
	return nil
}

// scheduleBackup schedules a backup job with the given cron expression
func (s *BackupScheduler) scheduleBackup(cronExpr string) error {
	// Remove existing job if any
	if s.jobEntry != 0 {
		s.cron.Remove(s.jobEntry)
	}

	// Parse and validate cron expression
	_, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Schedule new job
	entryID, err := s.cron.AddFunc(cronExpr, s.runScheduledBackup)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobEntry = entryID
	return nil
}

// runScheduledBackup executes a scheduled backup
func (s *BackupScheduler) runScheduledBackup() {
	// Create backup request
	req := services.BackupRequest{
		Type:              models.BackupTypeScheduled,
		EncryptionEnabled: true,
		IncludeCores:      true,
		IncludeCerts:      true,
		IncludeWARP:       true,
		IncludeGeo:        false, // Geo databases can be re-downloaded
	}

	// Create backup
	backup, err := s.backupService.CreateBackup(req)
	if err != nil {
		// Log error - in production this should use proper logging
		fmt.Printf("Scheduled backup failed: %v\n", err)
		return
	}

	fmt.Printf("Scheduled backup created: %s (ID: %d)\n", backup.Filename, backup.ID)
}

// UpdateSchedule updates the backup schedule
func (s *BackupScheduler) UpdateSchedule(cronExpr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cronExpr == "" {
		// Remove existing schedule
		if s.jobEntry != 0 {
			s.cron.Remove(s.jobEntry)
			s.jobEntry = 0
		}
		return s.backupService.SetSchedule("")
	}

	// Validate cron expression
	_, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Update schedule in database
	if err := s.backupService.SetSchedule(cronExpr); err != nil {
		return fmt.Errorf("failed to save schedule: %w", err)
	}

	// Update cron job
	return s.scheduleBackup(cronExpr)
}

// GetSchedule returns the current backup schedule
func (s *BackupScheduler) GetSchedule() (string, error) {
	return s.backupService.GetSchedule()
}

// GetNextRun returns the next scheduled run time
func (s *BackupScheduler) GetNextRun() (*time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.jobEntry == 0 {
		return nil, nil
	}

	entry := s.cron.Entry(s.jobEntry)
	if entry.ID == 0 {
		return nil, nil
	}

	next := entry.Next
	return &next, nil
}

// Stop stops the scheduler
func (s *BackupScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron.Stop()
}

// Close gracefully shuts down the scheduler
func (s *BackupScheduler) Close() {
	s.Stop()
}
