package cores

import (
	"fmt"
	"net"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// CoreManager manages proxy cores (Xray, Sing-box, Mihomo)
type CoreManager struct {
	db         *gorm.DB
	supervisor *SupervisorClient
	coreCfg    *CoreConfig
}

// NewCoreManager creates a new core manager
func NewCoreManager(db *gorm.DB, supervisorURL string, coreCfg *CoreConfig) *CoreManager {
	if coreCfg == nil {
		coreCfg = &CoreConfig{}
		coreCfg.ApplyDefaults()
	}
	return &CoreManager{
		db:         db,
		supervisor: NewSupervisorClient(supervisorURL),
		coreCfg:    coreCfg,
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

	// Health check: verify first inbound port is actually listening
	var firstInbound models.Inbound
	if err := cm.db.Where("core_id = ? AND is_enabled = ?", core.ID, true).Order("id ASC").First(&firstInbound).Error; err == nil {
		if err := cm.waitForPort(firstInbound.Port, 10*time.Second); err != nil {
			logger.Log.Warn().Int("port", firstInbound.Port).Str("core", name).Msg("Core started but port not yet listening")
		}
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

	// Health check: verify first inbound port is actually listening
	var firstInbound models.Inbound
	if err := cm.db.Where("core_id = ? AND is_enabled = ?", core.ID, true).Order("id ASC").First(&firstInbound).Error; err == nil {
		if err := cm.waitForPort(firstInbound.Port, 10*time.Second); err != nil {
			logger.Log.Warn().Int("port", firstInbound.Port).Str("core", name).Msg("Core restarted but port not yet listening")
		}
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

// waitForPort waits for a TCP port to become available
func (cm *CoreManager) waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("port %d not listening after %s", port, timeout)
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
<<<<<<< Updated upstream
=======

func (cm *CoreManager) checkCoreHealth(name string) error {
	switch name {
	case "xray":
		conn, err := net.DialTimeout("tcp", cm.coreCfg.XrayAPIAddr(), 3*time.Second)
		if err != nil {
			return fmt.Errorf("xray gRPC API not reachable: %w", err)
		}
		conn.Close()
	case "singbox":
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+cm.coreCfg.ClashAPIAddr()+"/version", nil)
		resp, err := healthCheckClient.Do(req)
		if err != nil {
			return fmt.Errorf("singbox API not reachable: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("singbox API returned %d", resp.StatusCode)
		}
	case "mihomo":
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+cm.coreCfg.MihomoAPIAddr()+"/version", nil)
		resp, err := healthCheckClient.Do(req)
		if err != nil {
			return fmt.Errorf("mihomo API not reachable: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("mihomo API returned %d", resp.StatusCode)
		}
	default:
		var core models.Core
		if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
			return fmt.Errorf("core not found: %w", err)
		}
		var firstInbound models.Inbound
		if err := cm.db.Where("core_id = ? AND is_enabled = ?", core.ID, true).Order("id ASC").First(&firstInbound).Error; err == nil {
			if err := cm.waitForPort(firstInbound.Port, 5*time.Second); err != nil {
				return fmt.Errorf("port %d not listening: %w", firstInbound.Port, err)
			}
		}
	}
	return nil
}
>>>>>>> Stashed changes
