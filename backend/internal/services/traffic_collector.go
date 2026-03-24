package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/cores/mihomo"
	"github.com/vovk4morkovk4/isolate-panel/internal/cores/singbox"
	"github.com/vovk4morkovk4/isolate-panel/internal/cores/xray"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/stats"
	"gorm.io/gorm"
)

// TrafficCollector collects traffic statistics from all cores
type TrafficCollector struct {
	db            *gorm.DB
	interval      time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	xrayClient    *xray.StatsClient
	singboxClient *singbox.StatsClient
	mihomoClient  *mihomo.StatsClient
	mu            sync.RWMutex
}

// NewTrafficCollector creates a new traffic collector
func NewTrafficCollector(
	db *gorm.DB,
	interval time.Duration,
	xrayAddr, singboxAddr, mihomoAddr string,
	singboxAPIKey, mihomoAPIKey string,
) *TrafficCollector {
	// Auto-detect interval based on system load (simplified)
	if interval == 0 {
		interval = 30 * time.Second // Default: 30 seconds
	}

	tc := &TrafficCollector{
		db:       db,
		interval: interval,
		stopChan: make(chan struct{}),
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

		// Save to database
		for _, sample := range samples {
			if sample.UserID == 0 || sample.InboundID == 0 {
				// Skip samples without proper user/inbound mapping
				continue
			}

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
