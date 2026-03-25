package services_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

func TestSettingsService_GetSetting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("get existing setting", func(t *testing.T) {
		setting, err := service.GetSetting("monitoring_mode")
		require.NoError(t, err)
		assert.NotNil(t, setting)
		assert.Equal(t, "monitoring_mode", setting.Key)
		assert.Equal(t, "lite", setting.Value)
	})

	t.Run("get non-existing setting", func(t *testing.T) {
		setting, err := service.GetSetting("non_existent_key")
		require.Error(t, err)
		assert.Nil(t, setting)
	})
}

func TestSettingsService_GetMonitoringMode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("get default monitoring mode", func(t *testing.T) {
		mode, err := service.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "lite", mode)
	})
}

func TestSettingsService_GetMonitoringInterval(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("get lite interval", func(t *testing.T) {
		interval, err := service.GetMonitoringInterval()
		require.NoError(t, err)
		assert.Equal(t, 60*time.Second, interval)
	})

	t.Run("get full interval", func(t *testing.T) {
		// Update to full mode
		err := service.UpdateMonitoringMode("full")
		require.NoError(t, err)

		interval, err := service.GetMonitoringInterval()
		require.NoError(t, err)
		assert.Equal(t, 10*time.Second, interval)
	})
}

func TestSettingsService_UpdateMonitoringMode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("update to full mode", func(t *testing.T) {
		err := service.UpdateMonitoringMode("full")
		require.NoError(t, err)

		mode, err := service.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "full", mode)
	})

	t.Run("update back to lite mode", func(t *testing.T) {
		err := service.UpdateMonitoringMode("lite")
		require.NoError(t, err)

		mode, err := service.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "lite", mode)
	})

	t.Run("update with invalid mode", func(t *testing.T) {
		err := service.UpdateMonitoringMode("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid monitoring mode")
	})
}

func TestSettingsService_UpdateSetting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("update existing setting", func(t *testing.T) {
		err := service.UpdateSetting("monitoring_mode", "full")
		require.NoError(t, err)

		value, err := service.GetSettingValue("monitoring_mode")
		require.NoError(t, err)
		assert.Equal(t, "full", value)
	})

	t.Run("update non-existing setting", func(t *testing.T) {
		err := service.UpdateSetting("new_setting", "value")
		require.NoError(t, err)

		value, err := service.GetSettingValue("new_setting")
		require.NoError(t, err)
		assert.Equal(t, "value", value)
	})
}

func TestSettingsService_GetSettingValue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("get existing value", func(t *testing.T) {
		value, err := service.GetSettingValue("monitoring_mode")
		require.NoError(t, err)
		assert.Equal(t, "lite", value)
	})

	t.Run("get non-existing value", func(t *testing.T) {
		value, err := service.GetSettingValue("non_existent")
		require.Error(t, err)
		assert.Empty(t, value)
	})
}

func TestSettingsService_GetAllSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("get all settings", func(t *testing.T) {
		settings, err := service.GetAllSettings()
		require.NoError(t, err)
		assert.NotEmpty(t, settings)
		assert.Greater(t, len(settings), 0)
	})
}

func TestSettingsService_UpdateSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	service := services.NewSettingsService(db)

	t.Run("update multiple settings", func(t *testing.T) {
		updates := map[string]string{
			"monitoring_mode": "full",
		}

		err := service.UpdateSettings(updates)
		require.NoError(t, err)

		mode, err := service.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "full", mode)
	})
}
