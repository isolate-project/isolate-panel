package app

import (
	"context"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

type Watchdog struct {
	db       *gorm.DB
	cores    *cores.CoreManager
	interval time.Duration
	timeout  time.Duration
	stopChan chan struct{}
	doneChan chan struct{}
}

func NewWatchdog(db *gorm.DB, cm *cores.CoreManager, interval, timeout time.Duration) *Watchdog {
	if interval == 0 {
		interval = 30 * time.Second
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Watchdog{
		db:       db,
		cores:    cm,
		interval: interval,
		timeout:  timeout,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

func (w *Watchdog) Start() {
	go w.run()
}

func (w *Watchdog) Stop() {
	close(w.stopChan)
	<-w.doneChan
}

func (w *Watchdog) run() {
	defer close(w.doneChan)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.checkAll()

	for {
		select {
		case <-ticker.C:
			w.checkAll()
		case <-w.stopChan:
			return
		}
	}
}

func (w *Watchdog) checkAll() {
	log := logger.Log
	var coreList []models.Core
	if err := w.db.Where("is_enabled = ?", true).Find(&coreList).Error; err != nil {
		log.Error().Err(err).Msg("Watchdog: failed to list cores")
		return
	}

	for _, core := range coreList {
		adapter, err := cores.GetCoreAdapter(core.Name)
		if err != nil {
			log.Warn().Str("core", core.Name).Msg("Watchdog: no adapter registered")
			continue
		}
		if setter, ok := adapter.(interface{ SetCoreConfig(*cores.CoreConfig) }); ok {
			setter.SetCoreConfig(w.cores.GetCoreConfig())
		}

		ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
		healthErr := adapter.CheckHealth(ctx, w.timeout)
		cancel()

		if healthErr != nil {
			log.Warn().Str("core", core.Name).Err(healthErr).Msg("Watchdog: core health check failed")
			core.HealthStatus = "unhealthy"
			core.LastError = healthErr.Error()
		} else {
			core.HealthStatus = "healthy"
			core.LastError = ""
		}

		if saveErr := w.db.Save(&core).Error; saveErr != nil {
			log.Error().Err(saveErr).Str("core", core.Name).Msg("Watchdog: failed to update core status")
		}
	}
}