package app

import (
	"context"
	"time"

	applogger "github.com/isolate-project/isolate-panel/internal/logger"
)

// StartWorkers starts all background services and periodic goroutines.
func StartWorkers(a *App) {
	go a.DashboardHub.Run()
	a.Traffic.Start()
	a.Connections.Start()
	a.Aggregator.Start()
	a.Retention.Start()
	a.Warp.StartAutoRefresh(24 * time.Hour)
	a.Geo.StartAutoUpdate(7 * 24 * time.Hour)

	// Quota enforcement + expiry check loop (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.Quota.CheckAndEnforce(context.Background())
				a.Users.CheckExpiringUsers()
			case <-a.stopQuota:
				return
			}
		}
	}()
}

// StopWorkers gracefully stops all background services.
func StopWorkers(a *App) {
	log := applogger.Log
	log.Info().Msg("Stopping background services...")

	close(a.stopQuota)
	a.DashboardHub.Stop()
	a.Aggregator.Stop()
	a.Retention.Stop()
	a.Connections.Stop()
	a.Traffic.Stop()
	a.BackupSched.Stop()
	a.TrafficResetSched.Stop()
	a.Warp.StopAutoRefresh()
	a.Geo.StopAutoUpdate()
	if a.Certs != nil {
		a.Certs.Stop()
	}
	if a.Cache != nil {
		a.Cache.Close()
	}

	// Stop rate limiter cleanup goroutines
	if a.LoginRL != nil {
		a.LoginRL.Stop()
	}
	if a.ProtectedRL != nil {
		a.ProtectedRL.Stop()
	}
	if a.HeavyRL != nil {
		a.HeavyRL.Stop()
	}
	if a.SubTokenRL != nil {
		a.SubTokenRL.Stop()
	}
	if a.SubIPRL != nil {
		a.SubIPRL.Stop()
	}
}
