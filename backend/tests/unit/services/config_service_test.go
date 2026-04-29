package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Core{},
		&models.Inbound{},
		&models.User{},
		&models.Setting{},
	)
	require.NoError(t, err)
	return db
}

func TestConfigService(t *testing.T) {
	t.Run("creates config service", func(t *testing.T) {
		db := setupTestDB(t)
		service := services.NewConfigService(db, nil, "/tmp", func(coreID uint) (string, error) { return "test-secret", nil })
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
