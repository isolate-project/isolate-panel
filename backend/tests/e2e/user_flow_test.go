package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
)

// TestCompleteUserFlow tests the complete user lifecycle
func TestCompleteUserFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	// Initialize services
	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)
	settingsService := services.NewSettingsService(db)

	t.Run("Step 1: Create user", func(t *testing.T) {
		req := &services.CreateUserRequest{
			Username: "e2euser",
			Email:    "e2e@example.com",
			Password: "password12345",
		}

		user, err := userService.CreateUser(req, 1)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "e2euser", user.Username)
		assert.NotEmpty(t, user.UUID)
		assert.NotEmpty(t, user.SubscriptionToken)
	})

	t.Run("Step 2: Get user", func(t *testing.T) {
		user, err := userService.GetUser(1)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser1", user.Username)
	})

	t.Run("Step 3: Update user", func(t *testing.T) {
		newLimit := int64(214748364800) // 200GB
		req := &services.UpdateUserRequest{
			TrafficLimitBytes: &newLimit,
		}

		user, err := userService.UpdateUser(1, req)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, newLimit, *user.TrafficLimitBytes)
	})

	t.Run("Step 4: Check monitoring settings", func(t *testing.T) {
		mode, err := settingsService.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "lite", mode)

		interval, err := settingsService.GetMonitoringInterval()
		require.NoError(t, err)
		assert.Equal(t, 60, int(interval.Seconds()))
	})

	t.Run("Step 5: Update monitoring mode", func(t *testing.T) {
		err := settingsService.UpdateMonitoringMode("full")
		require.NoError(t, err)

		mode, err := settingsService.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "full", mode)

		interval, err := settingsService.GetMonitoringInterval()
		require.NoError(t, err)
		assert.Equal(t, 10, int(interval.Seconds()))
	})

	t.Run("Step 6: Delete user", func(t *testing.T) {
		err := userService.DeleteUser(2)
		require.NoError(t, err)

		// Verify user is deleted
		user, err := userService.GetUser(2)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

// TestQuotaEnforcement tests traffic quota enforcement
func TestQuotaEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	t.Run("User with traffic limit", func(t *testing.T) {
		user, err := userService.GetUser(1)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, user.TrafficLimitBytes)
		assert.Greater(t, *user.TrafficLimitBytes, int64(0))
	})

	t.Run("User without traffic limit (unlimited)", func(t *testing.T) {
		// Create unlimited user
		limit := int64(0)
		req := &services.CreateUserRequest{
			Username:          "unlimited",
			Email:             "unlimited@example.com",
			Password:          "password12345",
			TrafficLimitBytes: &limit,
		}

		user, err := userService.CreateUser(req, 1)
		require.NoError(t, err)
		assert.NotNil(t, user)
		// Zero limit means unlimited
		assert.Equal(t, int64(0), *user.TrafficLimitBytes)
	})
}

// TestSettingsPersistence tests that settings persist across operations
func TestSettingsPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	settingsService := services.NewSettingsService(db)

	t.Run("Initial settings", func(t *testing.T) {
		mode, err := settingsService.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "lite", mode)
	})

	t.Run("Update and verify", func(t *testing.T) {
		// Update to full
		err := settingsService.UpdateMonitoringMode("full")
		require.NoError(t, err)

		// Verify immediately
		mode, err := settingsService.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "full", mode)

		// Create new service instance (simulates restart)
		newSettingsService := services.NewSettingsService(db)
		mode, err = newSettingsService.GetMonitoringMode()
		require.NoError(t, err)
		assert.Equal(t, "full", mode)
	})
}
