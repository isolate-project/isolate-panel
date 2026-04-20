package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/models"
)

func newSupervisorStub(t *testing.T, state int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf(`<?xml version="1.0"?>
<methodResponse>
  <params>
    <param><value><boolean>1</boolean></value></param>
  </params>
</methodResponse>`)))
	}))
}

func setupLifecycleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := db.AutoMigrate(&models.Core{}, &models.Inbound{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	db.Create(&models.Core{Name: "xray", IsEnabled: true})
	db.Create(&models.Core{Name: "singbox", IsEnabled: true})
	db.Create(&models.Core{Name: "mihomo", IsEnabled: true})
	return db
}

func TestCoreLifecycleManager_InitializeCores(t *testing.T) {
	db := setupLifecycleDB(t)
	var xrayCore models.Core
	db.Where("name = ?", "xray").First(&xrayCore)

	db.Create(&models.Inbound{Name: "test-in", Protocol: "vmess", CoreID: xrayCore.ID, Port: 443, IsEnabled: true})

	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	err := clm.InitializeCores()
	if err != nil {
		t.Fatalf("InitializeCores() error = %v", err)
	}
}

func TestCoreLifecycleManager_InitializeCores_NoInbounds(t *testing.T) {
	db := setupLifecycleDB(t)
	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	err := clm.InitializeCores()
	if err != nil {
		t.Fatalf("InitializeCores() error = %v", err)
	}
}

func TestCoreLifecycleManager_OnInboundCreated_CoreNotFound(t *testing.T) {
	db := setupLifecycleDB(t)
	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	inbound := &models.Inbound{Name: "new-in", Protocol: "vmess", CoreID: 9999, Port: 443, IsEnabled: true}
	err := clm.OnInboundCreated(inbound)
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreLifecycleManager_ShouldCoreBeRunning(t *testing.T) {
	db := setupLifecycleDB(t)
	var xrayCore models.Core
	db.Where("name = ?", "xray").First(&xrayCore)

	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	should, err := clm.shouldCoreBeRunning("xray")
	if err != nil {
		t.Fatalf("shouldCoreBeRunning() error = %v", err)
	}
	if should {
		t.Error("xray should NOT be running (no inbounds)")
	}

	db.Create(&models.Inbound{Name: "in1", Protocol: "vmess", CoreID: xrayCore.ID, Port: 443, IsEnabled: true})

	should, err = clm.shouldCoreBeRunning("xray")
	if err != nil {
		t.Fatalf("shouldCoreBeRunning() error = %v", err)
	}
	if !should {
		t.Error("xray SHOULD be running (has active inbound)")
	}
}

func TestCoreLifecycleManager_ShouldCoreBeRunning_NotFound(t *testing.T) {
	db := setupLifecycleDB(t)
	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	_, err := clm.shouldCoreBeRunning("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreLifecycleManager_SetConfigService(t *testing.T) {
	db := setupLifecycleDB(t)
	clm := NewCoreLifecycleManager(db, nil)
	clm.SetConfigService(nil)
	clm.SetNotificationService(nil)
}

func TestCoreLifecycleManager_OnInboundDeleted_NotLastInbound(t *testing.T) {
	db := setupLifecycleDB(t)
	var xrayCore models.Core
	db.Where("name = ?", "xray").First(&xrayCore)

	in1 := &models.Inbound{Name: "in1", Protocol: "vmess", CoreID: xrayCore.ID, Port: 443, IsEnabled: true}
	in2 := &models.Inbound{Name: "in2", Protocol: "vless", CoreID: xrayCore.ID, Port: 8443, IsEnabled: true}
	db.Create(in1)
	db.Create(in2)

	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	err := clm.OnInboundDeleted(in1)
	if err != nil {
		t.Fatalf("OnInboundDeleted() error = %v", err)
	}
}

func TestCoreLifecycleManager_OnInboundDeleted_CoreNotFound(t *testing.T) {
	db := setupLifecycleDB(t)
	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	inbound := &models.Inbound{Name: "orphan", Protocol: "vmess", CoreID: 9999, Port: 443, IsEnabled: true}
	err := clm.OnInboundDeleted(inbound)
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreLifecycleManager_OnInboundUpdated_CoreNotFound(t *testing.T) {
	db := setupLifecycleDB(t)
	srv := newSupervisorStub(t, 20)
	defer srv.Close()

	coreMgr := cores.NewCoreManager(db, srv.URL, nil)
	clm := NewCoreLifecycleManager(db, coreMgr)

	inbound := &models.Inbound{Name: "orphan", Protocol: "vmess", CoreID: 9999, Port: 443, IsEnabled: true}
	err := clm.OnInboundUpdated(inbound, true)
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}
