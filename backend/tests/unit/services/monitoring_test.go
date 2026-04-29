package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
)

// =======================================================================
// ConnectionTracker Tests
// =======================================================================

func TestConnectionTracker_AddAndGetConnections(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", func() string { return "" }, func() string { return "" })
	defer ct.Stop()

	conn := &models.ActiveConnection{
		UserID:     1,
		InboundID:  1,
		CoreID:     1,
		CoreName:   "xray",
		SourceIP:   "192.168.1.1",
		SourcePort: 12345,
	}

	err := ct.AddConnection(conn)
	require.NoError(t, err)
	assert.NotZero(t, conn.ID)

	// Get connections for user
	conns, err := ct.GetUserConnections(1)
	require.NoError(t, err)
	assert.Len(t, conns, 1)
	assert.Equal(t, "192.168.1.1", conns[0].SourceIP)
	assert.Equal(t, "xray", conns[0].CoreName)
}

func TestConnectionTracker_RemoveConnection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", func() string { return "" }, func() string { return "" })

	conn := &models.ActiveConnection{
		UserID:   1,
		CoreID:   1,
		CoreName: "singbox",
	}
	require.NoError(t, ct.AddConnection(conn))

	err := ct.RemoveConnection(conn.ID)
	require.NoError(t, err)

	conns, err := ct.GetUserConnections(1)
	require.NoError(t, err)
	assert.Empty(t, conns)
}

func TestConnectionTracker_GetActiveConnectionsCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", func() string { return "" }, func() string { return "" })

	// Initially 0
	count, err := ct.GetActiveConnectionsCount()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add 3 connections
	for i := 0; i < 3; i++ {
		require.NoError(t, ct.AddConnection(&models.ActiveConnection{
			UserID:   uint(i + 1),
			CoreID:   1,
			CoreName: "xray",
		}))
	}

	count, err = ct.GetActiveConnectionsCount()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestConnectionTracker_CleanupStaleConnections(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", func() string { return "" }, func() string { return "" })

	// Create a stale connection (last activity 5 minutes ago)
	stale := &models.ActiveConnection{
		UserID:       1,
		CoreID:       1,
		CoreName:     "xray",
		LastActivity: time.Now().Add(-5 * time.Minute),
	}
	require.NoError(t, db.Create(stale).Error)

	// Create a fresh connection
	fresh := &models.ActiveConnection{
		UserID:       2,
		CoreID:       1,
		CoreName:     "xray",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	require.NoError(t, ct.AddConnection(fresh))

	// Cleanup connections older than 2 minutes
	err := ct.CleanupStaleConnections(2 * time.Minute)
	require.NoError(t, err)

	// Only fresh connection should remain
	count, _ := ct.GetActiveConnectionsCount()
	assert.Equal(t, int64(1), count)
}

// =======================================================================
// QuotaEnforcer Tests
// =======================================================================

func TestQuotaEnforcer_CheckAndEnforce_DisablesExceededUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create notification service (minimal, no real sending)
	notifService := services.NewNotificationService(db, "", "", "", "")
	_ = notifService.Initialize()

	qe := services.NewQuotaEnforcer(db, nil, notifService)

	// Create user who exceeded quota
	trafficLimit := int64(1000)
	user := &models.User{
		UUID:              "test-uuid-exceed",
		Username:          "exceed_user",
		Email:             "exceed@test.com",
		SubscriptionToken: "tok-exceed",
		IsActive:          true,
		TrafficLimitBytes: &trafficLimit,
		TrafficUsedBytes:  1500, // Over limit
	}
	require.NoError(t, db.Create(user).Error)

	qe.CheckAndEnforce(context.Background())

	// Verify user is disabled
	var updated models.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	assert.False(t, updated.IsActive, "User should be disabled after exceeding quota")
}

func TestQuotaEnforcer_CheckAndEnforce_KeepsActiveUnderLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	qe := services.NewQuotaEnforcer(db, nil, nil)

	trafficLimit := int64(10000)
	user := &models.User{
		UUID:              "test-uuid-under",
		Username:          "under_user",
		Email:             "under@test.com",
		SubscriptionToken: "tok-under",
		IsActive:          true,
		TrafficLimitBytes: &trafficLimit,
		TrafficUsedBytes:  5000, // Under limit (50%)
	}
	require.NoError(t, db.Create(user).Error)

	qe.CheckAndEnforce(context.Background())

	var updated models.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	assert.True(t, updated.IsActive, "User should remain active under quota limit")
}

func TestQuotaEnforcer_EnableUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	qe := services.NewQuotaEnforcer(db, nil, nil)

	user := &models.User{
		UUID:              "test-uuid-enable",
		Username:          "disabled_user",
		Email:             "disabled@test.com",
		SubscriptionToken: "tok-disabled",
		IsActive:          false,
	}
	require.NoError(t, db.Create(user).Error)

	err := qe.EnableUser(context.Background(), user)
	require.NoError(t, err)

	var updated models.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	assert.True(t, updated.IsActive)
}

func TestQuotaEnforcer_ResetUserTraffic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	qe := services.NewQuotaEnforcer(db, nil, nil)

	user := &models.User{
		UUID:              "test-uuid-reset",
		Username:          "reset_user",
		Email:             "reset@test.com",
		SubscriptionToken: "tok-reset",
		IsActive:          true,
		TrafficUsedBytes:  999999,
	}
	require.NoError(t, db.Create(user).Error)

	err := qe.ResetUserTraffic(user.ID)
	require.NoError(t, err)

	var updated models.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	assert.Equal(t, int64(0), updated.TrafficUsedBytes)
}

// =======================================================================
// DataAggregator Tests
// =======================================================================

func TestDataAggregator_StartStop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	da := services.NewDataAggregator(db, 1*time.Hour)

	// Should not panic
	assert.NotPanics(t, func() {
		da.Start()
		da.Stop()
	})
}

// =======================================================================
// DataRetentionService Tests
// =======================================================================

func TestDataRetentionService_StartStop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	dr := services.NewDataRetentionService(db, 24*time.Hour)

	assert.NotPanics(t, func() {
		dr.Start()
		dr.Stop()
	})
}

func TestDataRetentionService_CleansOldRawStats(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create old raw stats (10 days ago)
	oldStat := &models.TrafficStats{
		UserID:      1,
		InboundID:   1,
		CoreID:      1,
		Upload:      100,
		Download:    200,
		Total:       300,
		RecordedAt:  time.Now().AddDate(0, 0, -10),
		Granularity: "raw",
	}
	require.NoError(t, db.Create(oldStat).Error)

	// Create recent raw stats (1 day ago)
	recentStat := &models.TrafficStats{
		UserID:      1,
		InboundID:   1,
		CoreID:      1,
		Upload:      500,
		Download:    600,
		Total:       1100,
		RecordedAt:  time.Now().AddDate(0, 0, -1),
		Granularity: "raw",
	}
	require.NoError(t, db.Create(recentStat).Error)

	// Run retention with short interval so it runs immediately
	dr := services.NewDataRetentionService(db, 1*time.Hour)
	dr.Start()
	// Give it a moment to run the initial cleanup
	time.Sleep(200 * time.Millisecond)
	dr.Stop()

	// Old stat should be deleted, recent should remain
	var count int64
	db.Model(&models.TrafficStats{}).Where("granularity = ?", "raw").Count(&count)
	assert.Equal(t, int64(1), count, "Only recent raw stat should remain")
}

func TestDataRetentionService_CleansStaleConnections(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Stale connection (2 hours ago)
	stale := &models.ActiveConnection{
		UserID:       1,
		CoreID:       1,
		CoreName:     "xray",
		StartedAt:    time.Now().Add(-3 * time.Hour),
		LastActivity: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, db.Create(stale).Error)

	// Fresh connection
	fresh := &models.ActiveConnection{
		UserID:       2,
		CoreID:       1,
		CoreName:     "xray",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	require.NoError(t, db.Create(fresh).Error)

	dr := services.NewDataRetentionService(db, 1*time.Hour)
	dr.Start()
	time.Sleep(200 * time.Millisecond)
	dr.Stop()

	var count int64
	db.Model(&models.ActiveConnection{}).Count(&count)
	assert.Equal(t, int64(1), count, "Only fresh connection should remain")
}
