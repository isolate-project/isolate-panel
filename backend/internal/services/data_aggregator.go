package services

import (
	"sync"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// DataAggregator aggregates raw traffic stats into hourly and daily summaries
type DataAggregator struct {
	db       *gorm.DB
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewDataAggregator creates a new data aggregator
func NewDataAggregator(db *gorm.DB, interval time.Duration) *DataAggregator {
	if interval == 0 {
		interval = 1 * time.Hour // Default: aggregate every hour
	}

	return &DataAggregator{
		db:       db,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start starts the aggregation loop
func (da *DataAggregator) Start() {
	da.wg.Add(1)
	go func() {
		defer da.wg.Done()
		ticker := time.NewTicker(da.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				da.aggregateHourly()
				da.aggregateDaily()
			case <-da.stopChan:
				return
			}
		}
	}()
}

// Stop stops the aggregation loop
func (da *DataAggregator) Stop() {
	close(da.stopChan)
	da.wg.Wait()
}

// aggregateHourly aggregates raw stats into hourly stats
func (da *DataAggregator) aggregateHourly() {
	now := time.Now()

	// Get the start of the previous hour
	hourStart := now.Truncate(time.Hour).Add(-1 * time.Hour)
	hourEnd := hourStart.Add(time.Hour)

	// Aggregate by user, inbound, core, and hour
	type Result struct {
		UserID    uint
		InboundID uint
		CoreID    uint
		Upload    uint64
		Download  uint64
	}

	var results []Result

	// Query raw stats from the previous hour
	err := da.db.Table("traffic_stats").
		Select("user_id, inbound_id, core_id, SUM(upload) as upload, SUM(download) as download").
		Where("granularity = ?", "raw").
		Where("recorded_at >= ? AND recorded_at < ?", hourStart, hourEnd).
		Group("user_id, inbound_id, core_id").
		Scan(&results).Error

	if err != nil {
		return
	}

	// Insert aggregated hourly stats
	for _, r := range results {
		stat := &models.TrafficStats{
			UserID:      r.UserID,
			InboundID:   r.InboundID,
			CoreID:      r.CoreID,
			Upload:      r.Upload,
			Download:    r.Download,
			Total:       r.Upload + r.Download,
			RecordedAt:  hourStart,
			Granularity: "hourly",
		}
		da.db.Create(stat)
	}

	// Delete raw stats older than 7 days
	cutoff := now.AddDate(0, 0, -7)
	da.db.Where("granularity = ? AND recorded_at < ?", "raw", cutoff).
		Delete(&models.TrafficStats{})
}

// aggregateDaily aggregates hourly stats into daily stats
func (da *DataAggregator) aggregateDaily() {
	now := time.Now()

	// Get the start of the previous day
	dayStart := now.Truncate(24*time.Hour).AddDate(0, 0, -1)
	dayEnd := dayStart.Add(24 * time.Hour)

	type Result struct {
		UserID    uint
		InboundID uint
		CoreID    uint
		Upload    uint64
		Download  uint64
	}

	var results []Result

	// Query hourly stats from the previous day
	err := da.db.Table("traffic_stats").
		Select("user_id, inbound_id, core_id, SUM(upload) as upload, SUM(download) as download").
		Where("granularity = ?", "hourly").
		Where("recorded_at >= ? AND recorded_at < ?", dayStart, dayEnd).
		Group("user_id, inbound_id, core_id").
		Scan(&results).Error

	if err != nil {
		return
	}

	// Insert aggregated daily stats
	for _, r := range results {
		stat := &models.TrafficStats{
			UserID:      r.UserID,
			InboundID:   r.InboundID,
			CoreID:      r.CoreID,
			Upload:      r.Upload,
			Download:    r.Download,
			Total:       r.Upload + r.Download,
			RecordedAt:  dayStart,
			Granularity: "daily",
		}
		da.db.Create(stat)
	}

	// Delete hourly stats older than 90 days
	cutoff := now.AddDate(0, 0, -90)
	da.db.Where("granularity = ? AND recorded_at < ?", "hourly", cutoff).
		Delete(&models.TrafficStats{})
}


