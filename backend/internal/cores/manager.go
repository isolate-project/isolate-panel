package cores

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// CoreManager manages proxy cores (Xray, Sing-box, Mihomo)
type CoreManager struct {
	db         *gorm.DB
	supervisor *SupervisorClient
}

// NewCoreManager creates a new core manager
func NewCoreManager(db *gorm.DB, supervisorURL string) *CoreManager {
	return &CoreManager{
		db:         db,
		supervisor: NewSupervisorClient(supervisorURL),
	}
}

// StartCore starts a core by name
func (cm *CoreManager) StartCore(name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	if !core.IsEnabled {
		return fmt.Errorf("core %s is disabled", name)
	}

	// Check if already running
	running, err := cm.supervisor.IsProcessRunning(name)
	if err != nil {
		return fmt.Errorf("failed to check if core is running: %w", err)
	}

	if running {
		return fmt.Errorf("core %s is already running", name)
	}

	// Start via supervisord
	if err := cm.supervisor.StartProcess(name); err != nil {
		// Update last error
		core.LastError = err.Error()
		cm.db.Save(&core)
		return err
	}

	// Wait for process to start with retry backoff
	if err := cm.waitForProcess(name, 5); err != nil {
		return err
	}

	// Get process info
	info, err := cm.supervisor.GetProcessInfo(name)
	if err != nil {
		return fmt.Errorf("failed to get process info: %w", err)
	}

	// Update database
	core.IsRunning = true
	core.PID = &info.PID
	core.RestartCount++
	core.LastError = ""

	if err := cm.db.Save(&core).Error; err != nil {
		return fmt.Errorf("failed to update core status: %w", err)
	}

	return nil
}

// StopCore stops a core by name
func (cm *CoreManager) StopCore(name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	// Check if running
	running, err := cm.supervisor.IsProcessRunning(name)
	if err != nil {
		return fmt.Errorf("failed to check if core is running: %w", err)
	}

	if !running {
		return fmt.Errorf("core %s is not running", name)
	}

	// Stop via supervisord
	if err := cm.supervisor.StopProcess(name); err != nil {
		core.LastError = err.Error()
		cm.db.Save(&core)
		return err
	}

	// Update database
	core.IsRunning = false
	core.PID = nil
	core.LastError = ""

	if err := cm.db.Save(&core).Error; err != nil {
		return fmt.Errorf("failed to update core status: %w", err)
	}

	return nil
}

// RestartCore restarts a core by name
func (cm *CoreManager) RestartCore(name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	if !core.IsEnabled {
		return fmt.Errorf("core %s is disabled", name)
	}

	// Restart via supervisord
	if err := cm.supervisor.RestartProcess(name); err != nil {
		core.LastError = err.Error()
		cm.db.Save(&core)
		return err
	}

	// Wait for process to start with retry backoff
	if err := cm.waitForProcess(name, 5); err != nil {
		return err
	}

	// Get process info
	info, err := cm.supervisor.GetProcessInfo(name)
	if err != nil {
		return fmt.Errorf("failed to get process info: %w", err)
	}

	// Update database
	core.IsRunning = true
	core.PID = &info.PID
	core.RestartCount++
	core.LastError = ""

	if err := cm.db.Save(&core).Error; err != nil {
		return fmt.Errorf("failed to update core status: %w", err)
	}

	return nil
}

// GetCoreStatus gets the current status of a core
func (cm *CoreManager) GetCoreStatus(name string) (*models.Core, error) {
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return nil, fmt.Errorf("core not found: %w", err)
	}

	// Get real-time status from supervisord
	running, err := cm.supervisor.IsProcessRunning(name)
	if err != nil {
		// If we can't get status, return database status
		return &core, nil
	}

	// Update running status if different
	if core.IsRunning != running {
		core.IsRunning = running
		if !running {
			core.PID = nil
		}
		cm.db.Save(&core)
	}

	// Get process info if running
	if running {
		info, err := cm.supervisor.GetProcessInfo(name)
		if err == nil {
			core.PID = &info.PID
			// Calculate uptime
			if info.Start > 0 {
				core.UptimeSeconds = int(info.Now - info.Start)
			}
		}
	}

	return &core, nil
}

// waitForProcess waits for a process to start with exponential backoff
func (cm *CoreManager) waitForProcess(name string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		running, err := cm.supervisor.IsProcessRunning(name)
		if err == nil && running {
			return nil
		}
	}
	return fmt.Errorf("process %s did not start within expected time", name)
}

// IsCoreRunning checks if a core is running
func (cm *CoreManager) IsCoreRunning(name string) (bool, error) {
	return cm.supervisor.IsProcessRunning(name)
}

// ListCores returns all cores
func (cm *CoreManager) ListCores() ([]models.Core, error) {
	var cores []models.Core
	if err := cm.db.Find(&cores).Error; err != nil {
		return nil, fmt.Errorf("failed to list cores: %w", err)
	}

	// Update real-time status for each core
	for i := range cores {
		running, err := cm.supervisor.IsProcessRunning(cores[i].Name)
		if err == nil && cores[i].IsRunning != running {
			cores[i].IsRunning = running
			if !running {
				cores[i].PID = nil
			}
			cm.db.Save(&cores[i])
		}
	}

	return cores, nil
}
