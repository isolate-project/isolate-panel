package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/models"
)

func setupWatchdogDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Core{}))
	return db
}

func TestNewWatchdog_Defaults(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")
	w := NewWatchdog(db, cm, 0, 0)
	assert.Equal(t, 30*time.Second, w.interval)
	assert.Equal(t, 5*time.Second, w.timeout)
}

func TestWatchdog_StartStop(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")
	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Second)

	w.Start()
	time.Sleep(200 * time.Millisecond)
	w.Stop()
}

func TestWatchdog_CheckAll_NoCores(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")
	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Second)
	w.checkAll()
}

func TestWatchdog_CheckAll_WithEnabledCore(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")

	core := models.Core{Name: "xray", IsEnabled: true, HealthStatus: "unknown"}
	require.NoError(t, db.Create(&core).Error)

	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Second)
	w.checkAll()

	var updated models.Core
	require.NoError(t, db.First(&updated, core.ID).Error)
	assert.Equal(t, "unhealthy", updated.HealthStatus)
}

func TestWatchdog_CheckAll_DisabledCore(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")

	core := models.Core{Name: "xray", IsEnabled: true, HealthStatus: "unknown"}
	require.NoError(t, db.Create(&core).Error)
	require.NoError(t, db.Model(&core).Update("is_enabled", false).Error)

	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Second)
	w.checkAll()

	var updated models.Core
	require.NoError(t, db.First(&updated, core.ID).Error)
	assert.Equal(t, "unknown", updated.HealthStatus)
	assert.Empty(t, updated.LastError)
}

func TestWatchdog_CheckAll_MultipleCores(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")

	core1 := models.Core{Name: "xray", IsEnabled: true, HealthStatus: "unknown"}
	core2 := models.Core{Name: "singbox", IsEnabled: true, HealthStatus: "unknown"}
	core3 := models.Core{Name: "mihomo", IsEnabled: true, HealthStatus: "unknown"}
	require.NoError(t, db.Create(&core1).Error)
	require.NoError(t, db.Create(&core2).Error)
	require.NoError(t, db.Create(&core3).Error)
	require.NoError(t, db.Model(&core3).Update("is_enabled", false).Error)

	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Second)
	w.checkAll()

	var updated1, updated2, updated3 models.Core
	require.NoError(t, db.First(&updated1, core1.ID).Error)
	require.NoError(t, db.First(&updated2, core2.ID).Error)
	require.NoError(t, db.First(&updated3, core3.ID).Error)

	t.Logf("Core1: %s, Enabled: %v, Health: %s", updated1.Name, updated1.IsEnabled, updated1.HealthStatus)
	t.Logf("Core2: %s, Enabled: %v, Health: %s", updated2.Name, updated2.IsEnabled, updated2.HealthStatus)
	t.Logf("Core3: %s, Enabled: %v, Health: %s", updated3.Name, updated3.IsEnabled, updated3.HealthStatus)

	assert.Equal(t, "unhealthy", updated1.HealthStatus)
	assert.Equal(t, "unhealthy", updated2.HealthStatus)
	assert.Equal(t, "unknown", updated3.HealthStatus)
}

func TestWatchdog_StopBlocksUntilDone(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")
	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Second)

	w.Start()

	done := make(chan bool)
	go func() {
		w.Stop()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Stop() did not complete within timeout")
	}
}

func TestWatchdog_ContextTimeout(t *testing.T) {
	db := setupWatchdogDB(t)
	cm := cores.NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")
	w := NewWatchdog(db, cm, 100*time.Millisecond, 1*time.Millisecond)

	core := models.Core{Name: "xray", IsEnabled: true, HealthStatus: "unknown"}
	require.NoError(t, db.Create(&core).Error)

	w.checkAll()

	var updated models.Core
	require.NoError(t, db.First(&updated, core.ID).Error)
	assert.Equal(t, "unhealthy", updated.HealthStatus)
}