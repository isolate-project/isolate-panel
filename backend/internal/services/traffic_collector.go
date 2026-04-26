package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	"github.com/isolate-project/isolate-panel/internal/cores/singbox"
	"github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/stats"
	"gorm.io/gorm"
)

// TrafficCollector collects traffic statistics from all cores
type TrafficCollector struct {
	db                 *gorm.DB
	settings           *SettingsService
	interval           time.Duration
	stopChan           chan struct{}
	wg                 sync.WaitGroup
	xrayClient         *xray.StatsClient
	singboxClient      *singbox.StatsClient
	mihomoClient       *mihomo.StatsClient
	mu                 sync.RWMutex
	reloadIntervalChan chan struct{}
}

// NewTrafficCollector creates a new traffic collector
func NewTrafficCollector(
	db *gorm.DB,
	settings *SettingsService,
	interval time.Duration,
	xrayAddr, singboxAddr, mihomoAddr string,
	singboxAPIKey, mihomoAPIKey string,
) *TrafficCollector {
	// Auto-detect interval based on monitoring_mode setting
	if interval == 0 {
		if settings != nil {
			interval, _ = settings.GetMonitoringInterval()
		} else {
			interval = 60 * time.Second // Default: 60 seconds (lite mode)
		}
	}

	tc := &TrafficCollector{
		db:                 db,
		settings:           settings,
		interval:           interval,
		stopChan:           make(chan struct{}),
		reloadIntervalChan: make(chan struct{}, 1),
	}

	// Initialize Xray client
	if xrayAddr != "" {
		client, err := xray.NewStatsClient(xrayAddr)
		if err != nil {
			// Log error but continue - Xray might not be running yet
		} else {
			tc.xrayClient = client
		}
	}

	// Initialize Sing-box client
	if singboxAddr != "" {
		tc.singboxClient = singbox.NewStatsClient(singboxAddr, singboxAPIKey)
	}

	// Initialize Mihomo client
	if mihomoAddr != "" {
		tc.mihomoClient = mihomo.NewStatsClient(mihomoAddr, mihomoAPIKey)
	}

	return tc
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
			case <-tc.reloadIntervalChan:
				// Reload interval from settings
				if tc.settings != nil {
					newInterval, err := tc.settings.GetMonitoringInterval()
					if err == nil && newInterval != tc.interval {
						tc.mu.Lock()
						tc.interval = newInterval
						tc.mu.Unlock()
						ticker.Reset(newInterval)
					}
				}
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

	// Close clients
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.xrayClient != nil {
		tc.xrayClient.Close()
	}
	if tc.singboxClient != nil {
		tc.singboxClient.Close()
	}
	if tc.mihomoClient != nil {
		tc.mihomoClient.Close()
	}
}

// ReloadInterval triggers a reload of the monitoring interval from settings
func (tc *TrafficCollector) ReloadInterval() {
	select {
	case tc.reloadIntervalChan <- struct{}{}:
		// Signal sent successfully
	default:
		// Reload already pending
	}
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

		// Get traffic stats for this core using appropriate provider
		samples, err := tc.getCoreStats(ctx, core)
		if err != nil {
			continue
		}

		// Batch insert traffic stats and collect per-user totals
		var statsToInsert []models.TrafficStats
		userTraffic := make(map[uint]uint64)

		for _, sample := range samples {
			if sample.UserID == 0 || sample.InboundID == 0 {
				continue
			}

			statsToInsert = append(statsToInsert, models.TrafficStats{
				UserID:      sample.UserID,
				InboundID:   sample.InboundID,
				CoreID:      sample.CoreID,
				Upload:      sample.Upload,
				Download:    sample.Download,
				Total:       sample.Upload + sample.Download,
				RecordedAt:  now,
				Granularity: "raw",
			})

			userTraffic[sample.UserID] += sample.Upload + sample.Download
		}

		if len(statsToInsert) > 0 {
			if err := tc.db.CreateInBatches(statsToInsert, 100).Error; err != nil {
				logger.Log.Error().Err(err).Str("core", core.Name).Int("count", len(statsToInsert)).
					Msg("traffic_collector: failed to insert traffic stats")
			}
		}

		// Atomic UPDATE without SELECT for each user
		for userID, bytes := range userTraffic {
			if err := tc.db.Model(&models.User{}).Where("id = ?", userID).
				Update("traffic_used_bytes", gorm.Expr("traffic_used_bytes + ?", int64(bytes))).Error; err != nil {
				logger.Log.Error().Err(err).Uint("user_id", userID).Uint64("bytes", bytes).
					Msg("traffic_collector: failed to update user traffic")
			}
		}
	}
}

// getCoreStats retrieves stats from a core using the appropriate provider
func (tc *TrafficCollector) getCoreStats(ctx context.Context, core models.Core) ([]stats.TrafficSample, error) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	switch core.Name {
	case "xray":
		if tc.xrayClient == nil {
			return []TrafficSample{}, fmt.Errorf("xray client not initialized")
		}
		return tc.xrayClient.GetTrafficStats(ctx, core.ID)

	case "singbox":
		if tc.singboxClient == nil {
			return []TrafficSample{}, fmt.Errorf("singbox client not initialized")
		}
		return tc.singboxClient.GetTrafficStats(ctx, core.ID)

	case "mihomo":
		if tc.mihomoClient == nil {
			return []TrafficSample{}, fmt.Errorf("mihomo client not initialized")
		}
		return tc.mihomoClient.GetTrafficStats(ctx, core.ID)

	default:
		return []TrafficSample{}, fmt.Errorf("unknown core: %s", core.Name)
	}
}

// GetXrayClient returns the Xray stats client for direct access
func (tc *TrafficCollector) GetXrayClient() *xray.StatsClient {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.xrayClient
}

// GetSingboxClient returns the Sing-box stats client for direct access
func (tc *TrafficCollector) GetSingboxClient() *singbox.StatsClient {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.singboxClient
}

// GetMihomoClient returns the Mihomo stats client for direct access
func (tc *TrafficCollector) GetMihomoClient() *mihomo.StatsClient {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.mihomoClient
}

// UpdateXrayClientAddress recreates the Xray stats client with a new address
func (tc *TrafficCollector) UpdateXrayClientAddress(newAddr string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.xrayClient != nil {
		tc.xrayClient.Close()
	}

	client, err := xray.NewStatsClient(newAddr)
	if err != nil {
		return fmt.Errorf("failed to create Xray stats client: %w", err)
	}

	tc.xrayClient = client
	return nil
}
