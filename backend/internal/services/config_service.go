package services

import (
	"fmt"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// ConfigService handles configuration generation and management
type ConfigService struct {
	db                 *gorm.DB
	coreManager        *cores.CoreManager
	configDir          string
	warpDir            string
	geoDir             string
	coreAPISecret      string
	v2rayAPIListenAddr string
	coreCfg            *cores.CoreConfig
}

// NewConfigService creates a new config service
func NewConfigService(db *gorm.DB, coreManager *cores.CoreManager, configDir, coreAPISecret string) *ConfigService {
	if configDir == "" {
		configDir = "./data/cores"
	}
	return &ConfigService{
		db:            db,
		coreManager:   coreManager,
		configDir:     configDir,
		warpDir:       "./data/warp",
		geoDir:        "./data/geo",
		coreAPISecret: coreAPISecret,
	}
}

// SetV2RayAPIListenAddr sets the sing-box v2ray_api listen address
func (s *ConfigService) SetV2RayAPIListenAddr(addr string) {
	s.v2rayAPIListenAddr = addr
}

func (s *ConfigService) SetCoreConfig(cfg *cores.CoreConfig) {
	s.coreCfg = cfg
}

// configContext creates a ConfigContext for generators
func (s *ConfigService) configContext() *cores.ConfigContext {
	return &cores.ConfigContext{
		DB:                 s.db,
		WarpDir:            s.warpDir,
		GeoDir:             s.geoDir,
		CoreAPISecret:      s.coreAPISecret,
		V2RayAPIListenAddr: s.v2rayAPIListenAddr,
		CoreConfig:         s.coreCfg,
	}
}

// RegenerateConfig regenerates configuration for a specific core
func (s *ConfigService) RegenerateConfig(coreName string) error {
	// Get core from database
	var coreModel models.Core
	if err := s.db.Where("name = ?", coreName).First(&coreModel).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

	adapter, err := cores.GetCoreAdapter(coreName)
	if err != nil {
		return fmt.Errorf("failed to get core adapter: %w", err)
	}

	configPath := filepath.Join(s.configDir, coreName, adapter.ConfigFilename())

	config, err := adapter.GenerateConfig(s.configContext(), coreModel.ID)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if err := adapter.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if err := adapter.WriteConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logger.Log.Info().Str("core", coreName).Str("path", configPath).Msg("Config regenerated")
	return nil
}

// RegenerateAndReload regenerates config and reloads the core
func (s *ConfigService) RegenerateAndReload(coreName string) error {
	if err := s.RegenerateConfig(coreName); err != nil {
		return err
	}

	isRunning, err := s.coreManager.IsCoreRunning(coreName)
	if err != nil {
		return fmt.Errorf("failed to check core status: %w", err)
	}

	if isRunning {
		if err := s.coreManager.RestartCore(coreName); err != nil {
			return fmt.Errorf("failed to reload core: %w", err)
		}
		logger.Log.Info().Str("core", coreName).Msg("Core reloaded")
	} else {
		logger.Log.Info().Str("core", coreName).Msg("Core is not running, skipping reload")
	}

	return nil
}
