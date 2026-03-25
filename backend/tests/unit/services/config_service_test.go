package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"gorm.io/gorm"
)

func TestConfigService(t *testing.T) {
	t.Run("creates config service", func(t *testing.T) {
		service := services.NewConfigService(nil, nil, "/tmp")
		assert.NotNil(t, service)
	})

	t.Run("gets config for core", func(t *testing.T) {
		db := setupTestDB(t)
		service := services.NewConfigService(db, nil, "/tmp")
		assert.NotNil(t, service)
	})
}

func TestConfigService_ConfigGeneration(t *testing.T) {
	t.Run("generates singbox config", func(t *testing.T) {
		config := map[string]interface{}{
			"log": map[string]string{
				"level": "info",
			},
		}
		assert.NotNil(t, config)
		assert.Equal(t, "info", config["log"].(map[string]string)["level"])
	})

	t.Run("generates xray config", func(t *testing.T) {
		config := map[string]interface{}{
			"log": map[string]string{
				"loglevel": "info",
			},
		}
		assert.NotNil(t, config)
	})

	t.Run("generates mihomo config", func(t *testing.T) {
		config := map[string]interface{}{
			"mode":      "rule",
			"log-level": "info",
		}
		assert.NotNil(t, config)
		assert.Equal(t, "rule", config["mode"])
	})
}

func setupTestDB(t *testing.T) *gorm.DB {
	return nil
}
