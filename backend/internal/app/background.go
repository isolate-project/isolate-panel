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
	a.Watchdog.Start()

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

	if a.Watchdog != nil {
		a.Watchdog.Stop()
	}
	close(a.stopQuota)
	if a.DashboardHub != nil {
		a.DashboardHub.Stop()
	}
	if a.Aggregator != nil {
		a.Aggregator.Stop()
	}
	if a.Retention != nil {
		a.Retention.Stop()
	}
	if a.Connections != nil {
		a.Connections.Stop()
	}
	if a.Traffic != nil {
		a.Traffic.Stop()
	}
	if a.BackupSched != nil {
		a.BackupSched.Stop()
	}
	if a.TrafficResetSched != nil {
		a.TrafficResetSched.Stop()
	}
	if a.LogRetentionSched != nil {
		a.LogRetentionSched.Stop()
	}
	if a.EventBus != nil {
		a.EventBus.Close()
	}
	if a.Warp != nil {
		a.Warp.StopAutoRefresh()
	}
	if a.LogRetentionSched != nil {
		a.LogRetentionSched.Stop()
	}
	if a.Warp != nil {
		a.Warp.StopAutoRefresh()
	}
	if a.Geo != nil {
		a.Geo.StopAutoUpdate()
	}
	if a.Certs != nil {
		a.Certs.Stop()
	}
	if a.Cache != nil {
		a.Cache.Close()
	}
	if a.Notifications != nil {
		a.Notifications.Stop()
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
