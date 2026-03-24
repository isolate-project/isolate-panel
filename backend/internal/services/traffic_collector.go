package services

import (
	"context"
	"sync"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// TrafficCollector collects traffic statistics from all cores
type TrafficCollector struct {
	db       *gorm.DB
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewTrafficCollector creates a new traffic collector
func NewTrafficCollector(
	db *gorm.DB,
	interval time.Duration,
) *TrafficCollector {
	// Auto-detect interval based on system load (simplified)
	// In production, this could be based on CPU/memory usage
	if interval == 0 {
		interval = 30 * time.Second // Default: 30 seconds
	}

	return &TrafficCollector{
		db:       db,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start starts the traffic collection loop
func (tc *TrafficCollector) Start() {
	tc.wg.Add(1)
	go func() {
		defer tc.wg.Done()
		ticker := time.NewTicker(tc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				tc.collectStats()
			case <-tc.stopChan:
				return
			}
		}
	}()
}

// Stop stops the traffic collection loop
func (tc *TrafficCollector) Stop() {
	close(tc.stopChan)
	tc.wg.Wait()
}

// collectStats collects traffic statistics from all cores
func (tc *TrafficCollector) collectStats() {
	ctx := context.Background()
	now := time.Now()

	// Get all cores
	var cores []models.Core
	if err := tc.db.Find(&cores).Error; err != nil {
		return
	}

	for _, core := range cores {
		if !core.IsRunning {
			continue
		}

		// Get traffic stats for this core
		samples, err := tc.getCoreStats(ctx, core.ID)
		if err != nil {
			continue
		}

		// Save to database
		for _, sample := range samples {
			stat := &models.TrafficStats{
				UserID:      sample.UserID,
				InboundID:   sample.InboundID,
				CoreID:      sample.CoreID,
				Upload:      sample.Upload,
				Download:    sample.Download,
				Total:       sample.Upload + sample.Download,
				RecordedAt:  now,
				Granularity: "raw",
			}

			tc.db.Create(stat)

			// Update user's total traffic used
			tc.updateUserTraffic(sample.UserID, sample.Upload+sample.Download)
		}
	}
}

// getCoreStats retrieves stats from a core (placeholder)
func (tc *TrafficCollector) getCoreStats(ctx context.Context, coreID uint) ([]TrafficSample, error) {
	// TODO: Implement per-core stats providers
	// For now, return empty slice
	return []TrafficSample{}, nil
}

// updateUserTraffic updates user's total traffic used
func (tc *TrafficCollector) updateUserTraffic(userID uint, bytes uint64) {
	var user models.User
	if err := tc.db.First(&user, userID).Error; err != nil {
		return
	}

	// Increment traffic used (convert to int64 for model compatibility)
	user.TrafficUsedBytes += int64(bytes)
	tc.db.Save(&user)
}
