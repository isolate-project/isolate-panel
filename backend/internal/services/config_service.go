package services

import (
	"fmt"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	mihomocore "github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	singboxcore "github.com/isolate-project/isolate-panel/internal/cores/singbox"
	xraycore "github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/cache"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// ConfigService handles configuration generation and management
type ConfigService struct {
	db          *gorm.DB
	coreManager *cores.CoreManager
	configDir   string
	warpDir     string
	geoDir      string
	cache       *cache.Cache
}

// NewConfigService creates a new config service
func NewConfigService(db *gorm.DB, coreManager *cores.CoreManager, configDir string, cacheManager ...*cache.CacheManager) *ConfigService {
	var configCache *cache.Cache
	if len(cacheManager) > 0 && cacheManager[0] != nil {
		configCache = cacheManager[0].GetConfigCache()
	}
	if configDir == "" {
		configDir = "./data/cores"
	}
	return &ConfigService{
		db:          db,
		coreManager: coreManager,
		configDir:   configDir,
		warpDir:     "./data/warp",
		geoDir:      "./data/geo",
		cache:       configCache,
	}
}

// configContext creates a ConfigContext for generators
func (s *ConfigService) configContext() *cores.ConfigContext {
	return &cores.ConfigContext{
		DB:      s.db,
		WarpDir: s.warpDir,
		GeoDir:  s.geoDir,
	}
}

// RegenerateConfig regenerates configuration for a specific core
func (s *ConfigService) RegenerateConfig(coreName string) error {
	// Get core from database
	var coreModel models.Core
	if err := s.db.Where("name = ?", coreName).First(&coreModel).Error; err != nil {
		return fmt.Errorf("core not found: %w", err)
	}

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
		err = s.generateSingboxConfig(inbounds, outbounds, configPath)
	case "xray":
		configPath = filepath.Join(s.configDir, "xray", "config.json")
		err = s.generateXrayConfig(inbounds, outbounds, configPath)
	case "mihomo":
		configPath = filepath.Join(s.configDir, "mihomo", "config.yaml")
		err = s.generateMihomoConfig(inbounds, outbounds, configPath)
	default:
		return fmt.Errorf("unknown core type: %s", coreName)
	}

	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	fmt.Printf("✓ Config regenerated for %s: %s\n", coreName, configPath)
	return nil
}

// RegenerateAndReload regenerates config and reloads the core
func (s *ConfigService) RegenerateAndReload(coreName string) error {
	// Regenerate config
	if err := s.RegenerateConfig(coreName); err != nil {
		return err
	}

	// Check if core is running
	isRunning, err := s.coreManager.IsCoreRunning(coreName)
	if err != nil {
		return fmt.Errorf("failed to check core status: %w", err)
	}

	// Only reload if core is running
	if isRunning {
		if err := s.coreManager.RestartCore(coreName); err != nil {
			return fmt.Errorf("failed to reload core: %w", err)
		}
		fmt.Printf("✓ Core %s reloaded\n", coreName)
	} else {
		fmt.Printf("ℹ Core %s is not running, skipping reload\n", coreName)
	}

	return nil
}

// generateSingboxConfig generates Sing-box configuration
func (s *ConfigService) generateSingboxConfig(inbounds []models.Inbound, outbounds []models.Outbound, path string) error {
	if len(inbounds) == 0 {
		return fmt.Errorf("no inbounds provided")
	}

	// Use the first inbound's core ID
	coreID := inbounds[0].CoreID

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
func (s *ConfigService) generateXrayConfig(inbounds []models.Inbound, outbounds []models.Outbound, path string) error {
	if len(inbounds) == 0 {
		return fmt.Errorf("no inbounds provided")
	}

	// Use the first inbound's core ID
	coreID := inbounds[0].CoreID

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
func (s *ConfigService) generateMihomoConfig(inbounds []models.Inbound, outbounds []models.Outbound, path string) error {
	if len(inbounds) == 0 {
		return fmt.Errorf("no inbounds provided")
	}

	// Use the first inbound's core ID
	coreID := inbounds[0].CoreID

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
