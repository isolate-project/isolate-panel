package services

import (
	"fmt"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// SettingsService manages application settings
type SettingsService struct {
	db *gorm.DB
}

// NewSettingsService creates a new settings service
func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{
		db: db,
	}
}

// GetSetting retrieves a setting by key
func (s *SettingsService) GetSetting(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetSettingValue retrieves a setting value by key
func (s *SettingsService) GetSettingValue(key string) (string, error) {
	setting, err := s.GetSetting(key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// UpdateSetting updates a setting value
func (s *SettingsService) UpdateSetting(key string, value string) error {
	return s.db.Model(&models.Setting{}).
		Where("key = ?", key).
		Update("value", value).Error
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
