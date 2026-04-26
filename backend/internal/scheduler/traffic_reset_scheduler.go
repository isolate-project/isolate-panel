package scheduler

import (
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// trafficResetCrons maps schedule names to standard 5-field cron expressions.
var trafficResetCrons = map[string]string{
	"weekly":  "0 0 * * 1", // Every Monday at midnight
	"monthly": "0 0 1 * *", // 1st of every month at midnight
}

// TrafficResetScheduler runs automatic traffic counter resets on a cron schedule.
type TrafficResetScheduler struct {
	settingsService *services.SettingsService
	quotaEnforcer   *services.QuotaEnforcer
	cron            *cron.Cron
	mu              sync.Mutex
	jobEntry        cron.EntryID
}

// NewTrafficResetScheduler creates a new TrafficResetScheduler.
func NewTrafficResetScheduler(
	settingsService *services.SettingsService,
	quotaEnforcer *services.QuotaEnforcer,
) *TrafficResetScheduler {
	return &TrafficResetScheduler{
		settingsService: settingsService,
		quotaEnforcer:   quotaEnforcer,
		cron:            cron.New(),
	}
}

// Initialize loads the schedule from settings and starts the cron runner.
func (s *TrafficResetScheduler) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, err := s.settingsService.GetTrafficResetSchedule()
	if err != nil {
		return fmt.Errorf("traffic reset scheduler: failed to get schedule: %w", err)
	}

	if schedule != "disabled" && schedule != "" {
		if err := s.scheduleReset(schedule); err != nil {
			return err
		}
	}

	s.cron.Start()
	return nil
}

// scheduleReset replaces the existing cron job with a new one for the given schedule.
// Must be called with s.mu held.
func (s *TrafficResetScheduler) scheduleReset(schedule string) error {
	if s.jobEntry != 0 {
		s.cron.Remove(s.jobEntry)
		s.jobEntry = 0
	}

	cronExpr, ok := trafficResetCrons[schedule]
	if !ok {
		return fmt.Errorf("unknown traffic reset schedule: %s", schedule)
	}

	entryID, err := s.cron.AddFunc(cronExpr, s.runReset)
	if err != nil {
		return fmt.Errorf("failed to add traffic reset cron job: %w", err)
	}

	s.jobEntry = entryID
	return nil
}

// runReset is called by the cron runner and performs the actual reset.
func (s *TrafficResetScheduler) runReset() {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error().Interface("panic", r).Msg("Scheduled traffic reset panicked, recovered")
		}
	}()

	if err := s.quotaEnforcer.ResetAllTraffic(); err != nil {
		logger.Log.Error().Err(err).Str("scheduler", "traffic_reset").Msg("Scheduled traffic reset failed")
	}
}

// GetSchedule returns the currently stored schedule name.
func (s *TrafficResetScheduler) GetSchedule() (string, error) {
	return s.settingsService.GetTrafficResetSchedule()
}

// UpdateSchedule persists the new schedule and replaces the cron job.
func (s *TrafficResetScheduler) UpdateSchedule(schedule string) error {
	if err := s.settingsService.SetTrafficResetSchedule(schedule); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if schedule == "disabled" {
		if s.jobEntry != 0 {
			s.cron.Remove(s.jobEntry)
			s.jobEntry = 0
		}
		return nil
	}

	return s.scheduleReset(schedule)
}

// Stop gracefully stops the cron runner.
func (s *TrafficResetScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cron.Stop()
}
