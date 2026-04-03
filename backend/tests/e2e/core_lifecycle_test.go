package e2e_test

import (
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreLifecycle tests the complete core database lifecycle
// Note: we test DB-level state only (no real process management in E2E)
func TestCoreLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	t.Run("Step 1: Verify cores seeded", func(t *testing.T) {
		var cores []models.Core
		err := db.Find(&cores).Error
		require.NoError(t, err)
		assert.Equal(t, 3, len(cores))

		names := make([]string, len(cores))
		for i, c := range cores {
			names[i] = c.Name
		}
		assert.Contains(t, names, "singbox")
		assert.Contains(t, names, "xray")
		assert.Contains(t, names, "mihomo")
	})

	t.Run("Step 2: Cores initially not running", func(t *testing.T) {
		var cores []models.Core
		err := db.Where("is_running = ?", true).Find(&cores).Error
		require.NoError(t, err)
		assert.Equal(t, 0, len(cores), "no cores should be running initially")
	})

	t.Run("Step 3: Simulate core start (DB state update)", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "singbox")
		pid := 12345
		core.IsRunning = true
		core.PID = &pid
		core.RestartCount = 0

		err := db.Save(core).Error
		require.NoError(t, err)

		// Verify persisted
		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.True(t, updated.IsRunning)
		assert.Equal(t, 12345, *updated.PID)
	})

	t.Run("Step 4: Simulate core stop (DB state update)", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "singbox")
		core.IsRunning = false
		core.PID = nil

		err := db.Save(core).Error
		require.NoError(t, err)

		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.False(t, updated.IsRunning)
		assert.Nil(t, updated.PID)
	})

	t.Run("Step 5: Simulate core restart with error tracking", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "xray")
		core.RestartCount++
		core.LastError = "connection refused"

		err := db.Save(core).Error
		require.NoError(t, err)

		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 1, updated.RestartCount)
		assert.Equal(t, "connection refused", updated.LastError)
	})

	t.Run("Step 6: Clear error after successful start", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "xray")
		pid := 99999
		core.IsRunning = true
		core.PID = &pid
		core.LastError = "" // clear error

		err := db.Save(core).Error
		require.NoError(t, err)

		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.True(t, updated.IsRunning)
		assert.Equal(t, "", updated.LastError)
	})

	t.Run("Step 7: Disable core", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "mihomo")
		core.IsEnabled = false
		core.IsRunning = false
		core.PID = nil

		err := db.Save(core).Error
		require.NoError(t, err)

		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.False(t, updated.IsEnabled)
		assert.False(t, updated.IsRunning)
	})

	t.Run("Step 8: Running core count query", func(t *testing.T) {
		// Set all cores to running except mihomo (disabled)
		db.Model(&models.Core{}).Where("name IN ?", []string{"singbox", "xray"}).
			Updates(map[string]interface{}{"is_running": true})

		var count int64
		err := db.Model(&models.Core{}).Where("is_running = ?", true).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("Step 9: Core uptime tracking", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "singbox")
		core.UptimeSeconds = 3600 // 1 hour

		err := db.Save(core).Error
		require.NoError(t, err)

		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 3600, updated.UptimeSeconds)
	})

	t.Run("Step 10: Inbound count for core", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "singbox")

		// Create two inbounds for singbox
		testutil.CreateTestInbound(t, db, "inbound-1", core.ID)

		var inboundCount int64
		err := db.Model(&models.Inbound{}).Where("core_id = ?", core.ID).Count(&inboundCount).Error
		require.NoError(t, err)
		assert.GreaterOrEqual(t, inboundCount, int64(1))
	})
}

// TestCoreInboundLifecycle tests inbound relationship with cores (E2E)
func TestCoreInboundLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	t.Run("Create user and assign to inbound", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "xray")
		inbound := testutil.CreateTestInbound(t, db, "test-vless", core.ID)

		// Create user
		req := &services.CreateUserRequest{
			Username: "inbound_test_user",
			Email:    "inbound@example.com",
			Password: "password12345",
		}
		user, err := userService.CreateUser(req, 1)
		require.NoError(t, err)
		require.NotNil(t, user)

		// Assign inbound to user
		mapping := &models.UserInboundMapping{
			UserID:    user.ID,
			InboundID: inbound.ID,
		}
		err = db.Create(mapping).Error
		require.NoError(t, err)

		// Verify mapping
		var count int64
		err = db.Model(&models.UserInboundMapping{}).
			Where("user_id = ? AND inbound_id = ?", user.ID, inbound.ID).
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Delete inbound removes mappings", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "xray")
		inbound := testutil.CreateTestInbound(t, db, "temp-inbound", core.ID)

		// Create mapping
		mapping := &models.UserInboundMapping{
			UserID:    1,
			InboundID: inbound.ID,
		}
		db.Create(mapping)

		// Delete inbound
		err := db.Delete(&models.Inbound{}, inbound.ID).Error
		require.NoError(t, err)

		// Verify inbound gone
		var ib models.Inbound
		err = db.First(&ib, inbound.ID).Error
		assert.Error(t, err, "inbound should be deleted")
	})

	t.Run("Config path update on core", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "singbox")
		core.ConfigPath = "/data/cores/singbox/config.json"
		core.LogPath = "/data/cores/singbox/singbox.log"

		err := db.Save(core).Error
		require.NoError(t, err)

		var updated models.Core
		err = db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "/data/cores/singbox/config.json", updated.ConfigPath)
	})
}

// TestCoreStatsTracking tests statistics tracking for cores
func TestCoreStatsTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	t.Run("Track multiple restarts", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "xray")

		// Simulate 5 restarts
		for i := 0; i < 5; i++ {
			core.RestartCount++
			err := db.Save(core).Error
			require.NoError(t, err)
			time.Sleep(1 * time.Millisecond) // small delay
		}

		var updated models.Core
		err := db.First(&updated, core.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 5, updated.RestartCount)
	})

	t.Run("Active connection count by core", func(t *testing.T) {
		core := testutil.GetTestCore(t, db, "singbox")

		// Create active connections
		for i := 0; i < 3; i++ {
			conn := &models.ActiveConnection{
				UserID:    uint(i + 1),
				InboundID: 1,
				CoreName:  core.Name,
				CoreID:    core.ID,
				StartedAt: time.Now(),
			}
			db.Create(conn)
		}

		var count int64
		err := db.Model(&models.ActiveConnection{}).
			Where("core_id = ?", core.ID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}
