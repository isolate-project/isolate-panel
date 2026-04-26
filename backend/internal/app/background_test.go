package app

import (
	"testing"

	"github.com/isolate-project/isolate-panel/internal/api"
	"github.com/isolate-project/isolate-panel/internal/scheduler"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
)

// TestStopWorkers_ClosesQuotaChannel verifies that StopWorkers closes the stopQuota channel.
func TestStopWorkers_ClosesQuotaChannel(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
	}

	// Verify channel is open before StopWorkers
	select {
	case <-a.stopQuota:
		t.Fatal("channel should not be closed yet")
	default:
	}

	StopWorkers(a)

	// Verify channel is closed after StopWorkers
	select {
	case <-a.stopQuota:
	default:
		t.Fatal("channel should be closed after StopWorkers")
	}
}

// TestStopWorkers_NilOptionalServices verifies that StopWorkers doesn't panic when optional services are nil.
func TestStopWorkers_NilOptionalServices(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
	}

	assert.NotPanics(t, func() {
		StopWorkers(a)
	})
}

// TestStopWorkers_NilRateLimiters verifies that StopWorkers handles nil rate limiters gracefully.
func TestStopWorkers_NilRateLimiters(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
		LoginRL:     nil,
		ProtectedRL: nil,
		HeavyRL:     nil,
		SubTokenRL:  nil,
		SubIPRL:     nil,
	}

	assert.NotPanics(t, func() {
		StopWorkers(a)
	})
}

// TestStopWorkers_NilCertsAndCache verifies that StopWorkers handles nil Certs and Cache gracefully.
func TestStopWorkers_NilCertsAndCache(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
		Certs: nil,
		Cache: nil,
	}

	assert.NotPanics(t, func() {
		StopWorkers(a)
	})
}

// TestStopWorkers_NilWatchdog verifies that StopWorkers handles nil Watchdog gracefully.
func TestStopWorkers_NilWatchdog(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
		Watchdog: nil,
	}

	assert.NotPanics(t, func() {
		StopWorkers(a)
	})
}

// TestStopWorkers_DoubleCall verifies that calling StopWorkers twice panics on the second call.
func TestStopWorkers_DoubleCall(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
	}

	// First call should succeed
	StopWorkers(a)

	// Second call should panic because stopQuota is already closed
	assert.Panics(t, func() {
		StopWorkers(a)
	})
}

// TestStopWorkers_ClosedChannel verifies that StopWorkers panics when stopQuota is already closed.
func TestStopWorkers_ClosedChannel(t *testing.T) {
	a := &App{
		stopQuota: make(chan struct{}),
		DashboardHub: api.NewDashboardHub(nil, nil, nil),
		Aggregator:    services.NewDataAggregator(nil, 0),
		Retention:     services.NewDataRetentionService(nil, 0),
		Connections:   services.NewConnectionTracker(nil, 0, "", "", "", "", ""),
		Traffic:       services.NewTrafficCollector(nil, nil, 0, "", "", "", "", ""),
		BackupSched:   scheduler.NewBackupScheduler(nil, nil),
		TrafficResetSched: scheduler.NewTrafficResetScheduler(nil, nil),
		Warp:          services.NewWARPService(nil, ""),
		Geo:           services.NewGeoService(nil, ""),
	}

	// Close the channel manually
	close(a.stopQuota)

	// StopWorkers should panic when trying to close an already-closed channel
	assert.Panics(t, func() {
		StopWorkers(a)
	})
}