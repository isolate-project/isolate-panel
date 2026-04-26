package cores

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// CoreManager manages proxy cores (Xray, Sing-box, Mihomo)
type CoreManager struct {
	db                 *gorm.DB
	supervisor         *SupervisorClient
	coreCfg            *CoreConfig
	configDir          string
	warpDir            string
	geoDir             string
	coreAPISecret      string
	v2rayAPIListenAddr string
	configMu           map[string]*sync.Mutex
	configMuMu         sync.RWMutex
}

// NewCoreManager creates a new core manager
func NewCoreManager(db *gorm.DB, supervisorURL string, coreCfg *CoreConfig, configDir string, warpDir string, geoDir string, coreAPISecret string, v2rayAPIListenAddr string) *CoreManager {
	if coreCfg == nil {
		coreCfg = &CoreConfig{}
		coreCfg.ApplyDefaults()
	}
	if configDir == "" {
		configDir = "./data/cores"
	}
	if warpDir == "" {
		warpDir = "./data/warp"
	}
	if geoDir == "" {
		geoDir = "./data/geo"
	}
	return &CoreManager{
		db:                 db,
		supervisor:         NewSupervisorClient(supervisorURL),
		coreCfg:            coreCfg,
		configDir:          configDir,
		warpDir:            warpDir,
		geoDir:             geoDir,
		coreAPISecret:      coreAPISecret,
		v2rayAPIListenAddr: v2rayAPIListenAddr,
		configMu:           make(map[string]*sync.Mutex),
	}
}

func (cm *CoreManager) getCoreMutex(name string) *sync.Mutex {
	cm.configMuMu.RLock()
	mu, ok := cm.configMu[name]
	cm.configMuMu.RUnlock()
	if ok {
		return mu
	}
	cm.configMuMu.Lock()
	defer cm.configMuMu.Unlock()
	// Double-check after acquiring write lock
	if mu, ok := cm.configMu[name]; ok {
		return mu
	}
	mu = &sync.Mutex{}
	cm.configMu[name] = mu
	return mu
}

// StartCore starts a core by name
func (cm *CoreManager) StartCore(ctx context.Context, name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	if !core.IsEnabled {
		return fmt.Errorf("core %s is disabled", name)
	}

	// Check if already running
	running, err := cm.supervisor.IsProcessRunning(name + ":" + name)
	if err != nil {
		return fmt.Errorf("failed to check if core is running: %w", err)
	}

	if running {
		return fmt.Errorf("core %s is already running", name)
	}

	// Start via supervisord
	if err := cm.supervisor.StartProcess(name + ":" + name); err != nil {
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
	info, err := cm.supervisor.GetProcessInfo(name + ":" + name)
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
func (cm *CoreManager) StopCore(ctx context.Context, name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	// Check if running
	running, err := cm.supervisor.IsProcessRunning(name + ":" + name)
	if err != nil {
		return fmt.Errorf("failed to check if core is running: %w", err)
	}

	if !running {
		return fmt.Errorf("core %s is not running", name)
	}

	// Stop via supervisord using StopProcessGroup to kill all child processes
	if err := cm.supervisor.StopProcessGroup(name); err != nil {
		core.LastError = err.Error()
		cm.db.Save(&core)
		return err
	}

	time.Sleep(300 * time.Millisecond)
	info, err := cm.supervisor.GetProcessInfo(name + ":" + name)
	if err == nil && info != nil && info.State == 20 {
		logger.Log.Warn().Str("core", name).Msg("Process may still be running after stop command")
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

// StopCoreGroup stops a core's entire process group via supervisord
// This ensures all child processes are terminated (stopasgroup=true)
func (cm *CoreManager) StopCoreGroup(ctx context.Context, name string) error {
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	// Stop the entire process group — this respects stopasgroup=true
	if err := cm.supervisor.StopProcessGroup(name); err != nil {
		core.LastError = err.Error()
		cm.db.Save(&core)
		return err
	}

	core.IsRunning = false
	core.PID = nil
	core.LastError = ""

	if err := cm.db.Save(&core).Error; err != nil {
		return fmt.Errorf("failed to update core status: %w", err)
	}

	return nil
}

// RestartCore restarts a core by name
func (cm *CoreManager) RestartCore(ctx context.Context, name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	if !core.IsEnabled {
		return fmt.Errorf("core %s is disabled", name)
	}

	// Restart via supervisord
	if err := cm.supervisor.RestartProcess(name + ":" + name); err != nil {
		core.LastError = err.Error()
		cm.db.Save(&core)
		return err
	}

	// Wait for process to start with retry backoff
	if err := cm.waitForProcess(name, 5); err != nil {
		return err
	}

	// Get process info
	info, err := cm.supervisor.GetProcessInfo(name + ":" + name)
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
func (cm *CoreManager) GetCoreStatus(ctx context.Context, name string) (*models.Core, error) {
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return nil, fmt.Errorf("core not found: %w", err)
	}

	// Get real-time status from supervisord
	running, err := cm.supervisor.IsProcessRunning(name + ":" + name)
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
		info, err := cm.supervisor.GetProcessInfo(name + ":" + name)
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
		running, err := cm.supervisor.IsProcessRunning(name + ":" + name)
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
	return cm.supervisor.IsProcessRunning(name + ":" + name)
}

// ListCores returns all cores
func (cm *CoreManager) ListCores(ctx context.Context) ([]models.Core, error) {
	var cores []models.Core
	if err := cm.db.Find(&cores).Error; err != nil {
		return nil, fmt.Errorf("failed to list cores: %w", err)
	}

	// Update real-time status for each core
	for i := range cores {
		running, err := cm.supervisor.IsProcessRunning(cores[i].Name + ":" + cores[i].Name)
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

func (cm *CoreManager) checkCoreHealth(name string) error {
	adapter, err := GetCoreAdapter(name)
	if err != nil {
		return fmt.Errorf("no adapter for core %s: %w", name, err)
	}
	if setter, ok := adapter.(interface{ SetCoreConfig(*CoreConfig) }); ok {
		setter.SetCoreConfig(cm.coreCfg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return adapter.CheckHealth(ctx, 5*time.Second)
}

// ReloadConfig regenerates and reloads configuration for a core using hot-reload when available
func (cm *CoreManager) ReloadConfig(ctx context.Context, name string) error {
	// Get core from database
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	if !core.IsEnabled {
		return fmt.Errorf("core %s is disabled, cannot reload", name)
	}

	// Get adapter
	adapter, err := GetCoreAdapter(name)
	if err != nil {
		return fmt.Errorf("failed to get core adapter: %w", err)
	}
	if setter, ok := adapter.(interface{ SetCoreConfig(*CoreConfig) }); ok {
		setter.SetCoreConfig(cm.coreCfg)
	}

	// Generate new config
	config, err := adapter.GenerateConfig(&ConfigContext{
		DB:                 cm.db,
		CoreConfig:         cm.coreCfg,
		WarpDir:            cm.warpDir,
		GeoDir:             cm.geoDir,
		CoreAPISecret:      cm.coreAPISecret,
		V2RayAPIListenAddr: cm.v2rayAPIListenAddr,
	}, core.ID)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Validate config
	if err := adapter.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	coreMu := cm.getCoreMutex(name)
	coreMu.Lock()
	defer coreMu.Unlock()

	// Write config to disk
	configPath := cm.configDir + "/" + name + "/" + adapter.ConfigFilename()
	if err := adapter.WriteConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Check if core is running
	isRunning, err := cm.supervisor.IsProcessRunning(name + ":" + name)
	if err != nil {
		return fmt.Errorf("failed to check core status: %w", err)
	}

	if !isRunning {
		return nil
	}

	// Get hot-reload method
	method, signal, _ := adapter.HotReloadInfo()

	switch method {
	case HotReloadSignal:
		// Send signal via supervisord
		if err := cm.supervisor.SignalProcess(name+":"+name, signal); err != nil {
			return fmt.Errorf("failed to send signal %s to core %s: %w", signal, name, err)
		}
		return nil

	case HotReloadAPI:
		// Use adapter's ReloadConfig which has core-specific API details (auth, method, path)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := adapter.ReloadConfig(ctx); err != nil {
			return fmt.Errorf("failed to reload config via API: %w", err)
		}
		return nil

	case HotReloadNone:
		// Fall back to full restart
		return cm.RestartCore(ctx, name)

	default:
		return fmt.Errorf("unknown hot-reload method: %d", method)
	}
}

// GetCoreAPIPort returns the API port for a core from the database
func (cm *CoreManager) GetCoreAPIPort(name string) (int, error) {
	var core models.Core
	if err := cm.db.Where("name = ?", name).First(&core).Error; err != nil {
		return 0, fmt.Errorf("core not found: %w", err)
	}
	return core.APIPort, nil
}

// ReloadCore reloads a core's configuration using hot-reload when available
func (cm *CoreManager) ReloadCore(ctx context.Context, name string) error {
	adapter, err := GetCoreAdapter(name)
	if err != nil {
		return fmt.Errorf("no adapter for core %s: %w", name, err)
	}
	if setter, ok := adapter.(interface{ SetCoreConfig(*CoreConfig) }); ok {
		setter.SetCoreConfig(cm.coreCfg)
	}
	if !adapter.SupportsHotReload() {
		return cm.RestartCore(ctx, name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := adapter.ReloadConfig(ctx); err != nil {
		cm.db.Model(&models.Core{}).Where("name = ?", name).Update("last_error", err.Error())
		return fmt.Errorf("hot-reload failed, try restart: %w", err)
	}
	return nil
}

// GetCoreConfig returns the CoreConfig
func (cm *CoreManager) GetCoreConfig() *CoreConfig {
	return cm.coreCfg
}
