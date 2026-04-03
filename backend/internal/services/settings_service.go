package services

import (
	"fmt"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cache"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// SettingsService manages application settings
type SettingsService struct {
	db    *gorm.DB
	cache *cache.Cache
}

// NewSettingsService creates a new settings service
// cacheManager is optional - if nil, caching will be disabled
func NewSettingsService(db *gorm.DB, cacheManager ...*cache.CacheManager) *SettingsService {
	var settingsCache *cache.Cache
	if len(cacheManager) > 0 && cacheManager[0] != nil {
		settingsCache = cacheManager[0].GetSettingsCache()
	}
	return &SettingsService{
		db:    db,
		cache: settingsCache,
	}
}

// GetSetting retrieves a setting by key
func (s *SettingsService) GetSetting(key string) (*models.Setting, error) {
	// Try cache first
	if s.cache != nil {
		if cached, found := s.cache.Get("setting:" + key); found {
			if setting, ok := cached.(*models.Setting); ok {
				return setting, nil
			}
		}
	}

	// Query database
	var setting models.Setting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set("setting:"+key, &setting)
	}

	return &setting, nil
}

// GetSettingValue retrieves a setting value by key
func (s *SettingsService) GetSettingValue(key string) (string, error) {
	// Try cache first for simple string lookup
	if s.cache != nil {
		if cached, found := s.cache.GetString("setting_value:" + key); found {
			return cached, nil
		}
	}

	setting, err := s.GetSetting(key)
	if err != nil {
		return "", err
	}

	// Cache the value
	if s.cache != nil {
		s.cache.Set("setting_value:"+key, setting.Value)
	}

	return setting.Value, nil
}

// UpdateSetting updates a setting value
func (s *SettingsService) UpdateSetting(key string, value string) error {
	var setting models.Setting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new setting if it doesn't exist
			newSetting := models.Setting{
				Key:   key,
				Value: value,
			}
			if err := s.db.Create(&newSetting).Error; err != nil {
				return err
			}
			// Cache the new setting
			if s.cache != nil {
				s.cache.Set("setting:"+key, &newSetting)
				s.cache.Set("setting_value:"+key, value)
			}
			return nil
		}
		return err
	}

	if err := s.db.Model(&setting).Update("value", value).Error; err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Delete("setting:" + key)
		s.cache.Delete("setting_value:" + key)
	}

	return nil
}

// GetMonitoringMode returns the current monitoring mode (lite or full)
func (s *SettingsService) GetMonitoringMode() (string, error) {
	value, err := s.GetSettingValue("monitoring_mode")
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "lite", nil // Default to lite
		}
		return "", err
	}
	return value, nil
}

// GetMonitoringInterval returns the monitoring interval based on mode
func (s *SettingsService) GetMonitoringInterval() (time.Duration, error) {
	mode, err := s.GetMonitoringMode()
	if err != nil {
		return 60 * time.Second, err
	}

	switch mode {
	case "full":
		return 10 * time.Second, nil
	case "lite":
		fallthrough
	default:
		return 60 * time.Second, nil
	}
}

// UpdateMonitoringMode updates the monitoring mode
func (s *SettingsService) UpdateMonitoringMode(mode string) error {
	if mode != "lite" && mode != "full" {
		return fmt.Errorf("invalid monitoring mode: %s (must be 'lite' or 'full')", mode)
	}
	return s.UpdateSetting("monitoring_mode", mode)
}

// GetTrafficResetSchedule returns the traffic auto-reset schedule
func (s *SettingsService) GetTrafficResetSchedule() (string, error) {
	value, err := s.GetSettingValue("traffic_reset_schedule")
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "disabled", nil
		}
		return "", err
	}
	return value, nil
}

// SetTrafficResetSchedule saves the traffic auto-reset schedule
func (s *SettingsService) SetTrafficResetSchedule(schedule string) error {
	switch schedule {
	case "disabled", "weekly", "monthly":
	default:
		return fmt.Errorf("invalid traffic reset schedule: %s", schedule)
	}
	return s.UpdateSetting("traffic_reset_schedule", schedule)
}

// GetAllSettings returns all settings
func (s *SettingsService) GetAllSettings() ([]models.Setting, error) {
	var settings []models.Setting
	if err := s.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// UpdateSettings updates multiple settings
func (s *SettingsService) UpdateSettings(updates map[string]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range updates {
			if err := tx.Model(&models.Setting{}).
				Where("key = ?", key).
				Update("value", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
