package services

import (
	"fmt"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
<<<<<<< Updated upstream
	mihomocore "github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	singboxcore "github.com/isolate-project/isolate-panel/internal/cores/singbox"
	xraycore "github.com/isolate-project/isolate-panel/internal/cores/xray"
=======
>>>>>>> Stashed changes
	"github.com/isolate-project/isolate-panel/internal/models"
)

// ConfigService handles configuration generation and management
type ConfigService struct {
<<<<<<< Updated upstream
	db            *gorm.DB
	coreManager   *cores.CoreManager
	configDir     string
	warpDir       string
	geoDir        string
	coreAPISecret string
=======
	db                 *gorm.DB
	coreManager        *cores.CoreManager
	configDir          string
	warpDir            string
	geoDir             string
	coreAPISecret      string
	v2rayAPIListenAddr string
	coreCfg            *cores.CoreConfig
>>>>>>> Stashed changes
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

<<<<<<< Updated upstream
// configContext creates a ConfigContext for generators
func (s *ConfigService) configContext() *cores.ConfigContext {
	return &cores.ConfigContext{
		DB:            s.db,
		WarpDir:       s.warpDir,
		GeoDir:        s.geoDir,
		CoreAPISecret: s.coreAPISecret,
=======
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
>>>>>>> Stashed changes
	}
}

// RegenerateConfig regenerates configuration for a specific core
func (s *ConfigService) RegenerateConfig(coreName string) error {
	// Get core from database
	var coreModel models.Core
	if err := s.db.Where("name = ?", coreName).First(&coreModel).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

<<<<<<< Updated upstream
	// Get all enabled inbounds for this core
	var inbounds []models.Inbound
	if err := s.db.Where("core_id = ? AND is_enabled = ?", coreModel.ID, true).Find(&inbounds).Error; err != nil {
		return fmt.Errorf("failed to get inbounds: %w", err)
	}

	// Get all enabled outbounds for this core
	var outbounds []models.Outbound
	if err := s.db.Where("core_id = ? AND is_enabled = ?", coreModel.ID, true).Find(&outbounds).Error; err != nil {
		return fmt.Errorf("failed to get outbounds: %w", err)
	}

	// Generate config based on core type
	var configPath string
	var err error

	switch coreName {
	case "singbox":
		configPath = filepath.Join(s.configDir, "singbox", "config.json")
		err = s.generateSingboxConfig(coreModel.ID, inbounds, outbounds, configPath)
	case "xray":
		configPath = filepath.Join(s.configDir, "xray", "config.json")
		err = s.generateXrayConfig(coreModel.ID, inbounds, outbounds, configPath)
	case "mihomo":
		configPath = filepath.Join(s.configDir, "mihomo", "config.yaml")
		err = s.generateMihomoConfig(coreModel.ID, inbounds, outbounds, configPath)
	default:
		return fmt.Errorf("unknown core type: %s", coreName)
=======
	adapter, err := cores.GetCoreAdapter(coreName)
	if err != nil {
		return fmt.Errorf("failed to get core adapter: %w", err)
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream

// generateSingboxConfig generates Sing-box configuration
func (s *ConfigService) generateSingboxConfig(coreID uint, inbounds []models.Inbound, outbounds []models.Outbound, path string) error {
	config, err := singboxcore.GenerateConfig(s.configContext(), coreID)
	if err != nil {
		return err
	}

	// Validate config first
	if err := singboxcore.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Write config using built-in function
	if err := singboxcore.WriteConfig(config, path); err != nil {
		return err
	}

	return nil
}

// generateXrayConfig generates Xray configuration
func (s *ConfigService) generateXrayConfig(coreID uint, inbounds []models.Inbound, outbounds []models.Outbound, path string) error {
	config, err := xraycore.GenerateConfig(s.configContext(), coreID)
	if err != nil {
		return err
	}

	// Validate config first
	if err := xraycore.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Write config using built-in function
	if err := xraycore.WriteConfig(config, path); err != nil {
		return err
	}

	return nil
}

// generateMihomoConfig generates Mihomo configuration
func (s *ConfigService) generateMihomoConfig(coreID uint, inbounds []models.Inbound, outbounds []models.Outbound, path string) error {
	config, err := mihomocore.GenerateConfig(s.configContext(), coreID)
	if err != nil {
		return err
	}

	// Validate config first
	if err := mihomocore.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Write config using built-in function
	if err := mihomocore.WriteConfig(config, path); err != nil {
		return err
	}

	return nil
}
=======
>>>>>>> Stashed changes
