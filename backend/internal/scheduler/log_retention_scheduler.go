package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
)

// RetentionPolicy defines cleanup rules for each data type.
type RetentionPolicy struct {
	Table       string
	DateField   string
	Retention   time.Duration
	ExtraWhere  string
	Description string
}

// DefaultRetentionPolicies are the built-in cleanup rules.
var DefaultRetentionPolicies = []RetentionPolicy{
	{
		Table:       "audit_logs",
		DateField:   "created_at",
		Retention:   90 * 24 * time.Hour,
		Description: "Admin audit logs",
	},
	{
		Table:       "login_attempts",
		DateField:   "attempted_at",
		Retention:   90 * 24 * time.Hour,
		Description: "Login attempts (success and failure)",
	},
	{
		Table:       "subscription_accesses",
		DateField:   "accessed_at",
		Retention:   90 * 24 * time.Hour,
		Description: "Subscription access logs",
	},
	{
		Table:       "traffic_stats",
		DateField:   "recorded_at",
		Retention:   90 * 24 * time.Hour,
		ExtraWhere:  "granularity IN ('raw', 'hourly')",
		Description: "Traffic stats (raw and hourly aggregates)",
	},
	{
		Table:       "traffic_stats",
		DateField:   "recorded_at",
		Retention:   365 * 24 * time.Hour,
		ExtraWhere:  "granularity = 'daily'",
		Description: "Traffic stats (daily aggregates kept longer)",
	},
	{
		Table:       "active_connections",
		DateField:   "last_activity",
		Retention:   24 * time.Hour,
		Description: "Stale active connections",
	},
	{
		Table:       "webauthn_session_data",
		DateField:   "created_at",
		Retention:   24 * time.Hour,
		Description: "WebAuthn temporary session data",
	},
	{
		Table:       "refresh_tokens",
		DateField:   "expires_at",
		Retention:   -1, // special: delete expired/revoked immediately
		ExtraWhere:  "revoked = 1 OR expires_at < ?",
		Description: "Revoked or expired refresh tokens",
	},
}

// LogRetentionScheduler cleans up old data on a schedule.
type LogRetentionScheduler struct {
	db         *gorm.DB
	cron       *cron.Cron
	mu         sync.Mutex
	jobEntry   cron.EntryID
	logPath    string
	policies   []RetentionPolicy
}

// NewLogRetentionScheduler creates a scheduler with default policies.
func NewLogRetentionScheduler(db *gorm.DB, logPath string) *LogRetentionScheduler {
	return &LogRetentionScheduler{
		db:       db,
		cron:     cron.New(),
		logPath:  logPath,
		policies: DefaultRetentionPolicies,
	}
}

// Initialize starts the daily cleanup job at 03:00.
func (s *LogRetentionScheduler) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, err := s.cron.AddFunc("0 3 * * *", s.runCleanup)
	if err != nil {
		return fmt.Errorf("failed to schedule log retention: %w", err)
	}
	s.jobEntry = entryID
	s.cron.Start()

	logger.Log.Info().Str("scheduler", "log_retention").Msg("Log retention scheduler initialized (daily at 03:00)")
	return nil
}

// runCleanup performs all retention cleanup tasks.
func (s *LogRetentionScheduler) runCleanup() {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error().Interface("panic", r).Msg("Log retention cleanup panicked, recovered")
		}
	}()

	start := time.Now()
	logger.Log.Info().Str("scheduler", "log_retention").Time("started", start).Msg("Starting log retention cleanup")

	var totalDeleted int64

	for _, policy := range s.policies {
		deleted := s.cleanupTable(policy)
		totalDeleted += deleted
	}

	// Cleanup old rotated log files on disk
	if s.logPath != "" {
		s.cleanupLogFiles()
	}

	logger.Log.Info().
		Str("scheduler", "log_retention").
		Int64("total_deleted_rows", totalDeleted).
		Dur("duration", time.Since(start)).
		Msg("Log retention cleanup completed")
}

// cleanupTable deletes rows older than the retention period.
func (s *LogRetentionScheduler) cleanupTable(policy RetentionPolicy) int64 {
	var cutoff interface{}
	if policy.Retention < 0 {
		// Special case: delete based on current time (e.g. expired tokens)
		cutoff = time.Now()
	} else {
		cutoff = time.Now().Add(-policy.Retention)
	}

	var query *gorm.DB
	if policy.ExtraWhere != "" {
		query = s.db.Where(policy.ExtraWhere, cutoff)
	} else {
		query = s.db.Where(policy.DateField+" < ?", cutoff)
	}

	result := query.Delete(&struct{}{})
	if result.Error != nil {
		logger.Log.Error().
			Err(result.Error).
			Str("table", policy.Table).
			Str("scheduler", "log_retention").
			Msg("Failed to cleanup table")
		return 0
	}

	if result.RowsAffected > 0 {
		logger.Log.Info().
			Str("scheduler", "log_retention").
			Str("table", policy.Table).
			Int64("deleted", result.RowsAffected).
			Str("description", policy.Description).
			Msg("Cleaned up old records")
	}
	return result.RowsAffected
}

// cleanupLogFiles removes old compressed log files (*.gz) older than 90 days.
func (s *LogRetentionScheduler) cleanupLogFiles() {
	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	logDir := filepath.Dir(s.logPath)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("path", logDir).
			Str("scheduler", "log_retention").
			Msg("Failed to read log directory")
		return
	}

	var removed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match rotated logs: isolate-panel.log.2024-01-01.gz or isolate-panel.log.1
		if !isRotatedLog(name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(logDir, name)
			if err := os.Remove(path); err == nil {
				removed++
				logger.Log.Info().
					Str("file", name).
					Str("scheduler", "log_retention").
					Msg("Removed old log file")
			}
		}
	}

	if removed > 0 {
		logger.Log.Info().
			Str("scheduler", "log_retention").
			Int("removed_files", removed).
			Msg("Cleaned up old log files")
	}
}

func isRotatedLog(name string) bool {
	// Matches: app.log.1, app.log.2024-01-01, app.log.2024-01-01.gz
	for _, suffix := range []string{".gz", ".1", ".2", ".3", ".4", ".5", ".6", ".7"} {
		if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

// RunCleanupNow triggers an immediate cleanup (for CLI/admin use).
func (s *LogRetentionScheduler) RunCleanupNow() (int64, error) {
	var totalDeleted int64
	for _, policy := range s.policies {
		totalDeleted += s.cleanupTable(policy)
	}
	if s.logPath != "" {
		s.cleanupLogFiles()
	}
	return totalDeleted, nil
}

// Stop stops the scheduler.
func (s *LogRetentionScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cron.Stop()
}

// AddPolicy adds a custom retention policy.
func (s *LogRetentionScheduler) AddPolicy(policy RetentionPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = append(s.policies, policy)
}
