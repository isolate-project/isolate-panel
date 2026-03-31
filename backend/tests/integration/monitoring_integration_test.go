package integration_test

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

// TestMonitoringFlow exercises the full monitoring pipeline:
//   create user → assign inbound → record traffic → aggregate → enforce quota
func TestMonitoringFlow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// --- Step 1: Create core, inbound, user, and assign user to inbound ---
	core := &models.Core{
		Name:      "xray",
		Version:   "26.2.6",
		IsEnabled: true,
		IsRunning: true,
	}
	require.NoError(t, db.Create(core).Error)

	inbound := testutil.CreateTestInbound(t, db, "test-vmess", core.ID)

	trafficLimit := int64(10000) // 10KB limit
	user := &models.User{
		UUID:              "flow-uuid-1",
		Username:          "flow_user",
		Email:             "flow@test.com",
		SubscriptionToken: "tok-flow",
		IsActive:          true,
		TrafficLimitBytes: &trafficLimit,
		TrafficUsedBytes:  0,
	}
	require.NoError(t, db.Create(user).Error)

	// Assign user to inbound
	mapping := &models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: inbound.ID,
	}
	require.NoError(t, db.Create(mapping).Error)

	// --- Step 2: Simulate traffic recording (as TrafficCollector would do) ---
	rawStats := []models.TrafficStats{
		{
			UserID:      user.ID,
			InboundID:   inbound.ID,
			CoreID:      core.ID,
			Upload:      3000,
			Download:    5000,
			Total:       8000,
			RecordedAt:  time.Now(),
			Granularity: "raw",
		},
	}
	for _, stat := range rawStats {
		require.NoError(t, db.Create(&stat).Error)
	}

	// Update user's cumulative traffic
	user.TrafficUsedBytes = 8000
	require.NoError(t, db.Save(user).Error)

	// --- Step 3: Verify DataAggregator can start without error ---
	da := services.NewDataAggregator(db, 1*time.Hour)
	assert.NotPanics(t, func() {
		da.Start()
		da.Stop()
	})

	// --- Step 4: Verify ConnectionTracker works ---
	ct := services.NewConnectionTracker(db, 10*time.Second, "", "", "", "", "")

	conn := &models.ActiveConnection{
		UserID:       user.ID,
		InboundID:    inbound.ID,
		CoreID:       core.ID,
		CoreName:     "xray",
		SourceIP:     "10.0.0.1",
		SourcePort:   54321,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	require.NoError(t, ct.AddConnection(conn))

	conns, err := ct.GetUserConnections(user.ID)
	require.NoError(t, err)
	assert.Len(t, conns, 1)
	assert.Equal(t, "10.0.0.1", conns[0].SourceIP)

	count, err := ct.GetActiveConnectionsCount()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// --- Step 5: Enforce quota (user at 80%, should get warning but stay active) ---
	qe := services.NewQuotaEnforcer(db, nil, nil)
	qe.CheckAndEnforce(context.Background())

	var afterWarning models.User
	require.NoError(t, db.First(&afterWarning, user.ID).Error)
	assert.True(t, afterWarning.IsActive, "User at 80% should still be active")

	// --- Step 6: Push user over 100% limit and check enforcement ---
	user.TrafficUsedBytes = 11000 // Over 10KB limit
	require.NoError(t, db.Save(user).Error)

	qe.CheckAndEnforce(context.Background())

	var afterBlock models.User
	require.NoError(t, db.First(&afterBlock, user.ID).Error)
	assert.False(t, afterBlock.IsActive, "User over quota should be disabled")

	// --- Step 7: Reset traffic and verify re-enable ---
	err = qe.ResetUserTraffic(user.ID)
	require.NoError(t, err)

	// Reload user from DB after traffic reset
	var userAfterReset models.User
	require.NoError(t, db.First(&userAfterReset, user.ID).Error)
	assert.Equal(t, int64(0), userAfterReset.TrafficUsedBytes)

	err = qe.EnableUser(context.Background(), &userAfterReset)
	require.NoError(t, err)

	var afterReset models.User
	require.NoError(t, db.First(&afterReset, user.ID).Error)
	assert.True(t, afterReset.IsActive)
	assert.Equal(t, int64(0), afterReset.TrafficUsedBytes)

	// --- Step 8: DataRetentionService cleans up old data ---
	// Create old raw stat
	oldStat := &models.TrafficStats{
		UserID:      user.ID,
		InboundID:   inbound.ID,
		CoreID:      core.ID,
		Upload:      100,
		Download:    200,
		Total:       300,
		RecordedAt:  time.Now().AddDate(0, 0, -15), // 15 days ago
		Granularity: "raw",
	}
	require.NoError(t, db.Create(oldStat).Error)

	dr := services.NewDataRetentionService(db, 1*time.Hour)
	dr.Start()
	time.Sleep(200 * time.Millisecond)
	dr.Stop()

	// Old raw stat should be cleaned up (default retention: 7 days)
	var rawCount int64
	db.Model(&models.TrafficStats{}).Where("granularity = ?", "raw").Count(&rawCount)
	assert.Equal(t, int64(1), rawCount, "Only the recent raw stat should remain")

	// --- Step 9: Cleanup connections ---
	err = ct.RemoveConnection(conn.ID)
	require.NoError(t, err)

	count, _ = ct.GetActiveConnectionsCount()
	assert.Equal(t, int64(0), count)
}
