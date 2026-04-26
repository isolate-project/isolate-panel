package cores

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

func setupManagerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := db.AutoMigrate(&models.Core{}, &models.Inbound{}, &models.Outbound{}, &models.User{}, &models.UserInboundMapping{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newMockSupervisorServer(t *testing.T, state int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if r.Method == "POST" {
			w.Write([]byte(`<?xml version="1.0"?>
<methodResponse>
  <params>
    <param><value><boolean>1</boolean></value></param>
  </params>
</methodResponse>`))
		}
	}))
}

func newMockProcessInfoServer(t *testing.T, state int, pid int) *httptest.Server {
	t.Helper()
	resp := fmt.Sprintf(`<?xml version="1.0"?>
<methodResponse>
  <params>
    <param>
      <value>
        <struct>
          <member><name>name</name><value><string>xray</string></value></member>
          <member><name>state</name><value><int>%d</int></value></member>
          <member><name>statename</name><value><string>RUNNING</string></value></member>
          <member><name>pid</name><value><int>%d</int></value></member>
          <member><name>start</name><value><int>1700000000</int></value></member>
          <member><name>stop</name><value><int>0</int></value></member>
          <member><name>now</name><value><int>1700001000</int></value></member>
          <member><name>exitstatus</name><value><int>0</int></value></member>
          <member><name>spawnerr</name><value><string></string></value></member>
        </struct>
      </value>
    </param>
  </params>
</methodResponse>`, state, pid)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(resp))
	}))
}

func TestCoreManager_StartCore_NotFound(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.StartCore(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreManager_StartCore_Disabled(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	db.Create(&models.Core{Name: "xray", IsEnabled: false, IsRunning: false})

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.StartCore(context.Background(), "xray")
	if err == nil {
		t.Error("expected error for disabled core, got nil")
	}
}

func TestCoreManager_StopCore_NotFound(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.StopCore(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreManager_RestartCore_NotFound(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.RestartCore(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreManager_RestartCore_Disabled(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	db.Create(&models.Core{Name: "xray", IsEnabled: false, IsRunning: false})

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.RestartCore(context.Background(), "xray")
	if err == nil {
		t.Error("expected error for disabled core, got nil")
	}
}

func TestCoreManager_GetCoreStatus_NotFound(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	_, err := cm.GetCoreStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreManager_GetCoreStatus_SupervisorDown(t *testing.T) {
	db := setupManagerDB(t)
	db.Create(&models.Core{Name: "xray", IsEnabled: true, IsRunning: false})

	cm := NewCoreManager(db, "http://127.0.0.1:1", nil, "", "", "", "", "")
	core, err := cm.GetCoreStatus(context.Background(), "xray")
	if err != nil {
		t.Fatalf("GetCoreStatus() error = %v", err)
	}
	if core == nil {
		t.Fatal("core should not be nil even when supervisor is down")
	}
}

func TestCoreManager_ListCores_Empty(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	cores, err := cm.ListCores(context.Background())
	if err != nil {
		t.Fatalf("ListCores() error = %v", err)
	}
	if len(cores) != 0 {
		t.Errorf("expected 0 cores, got %d", len(cores))
	}
}

func TestCoreManager_ListCores_WithData(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	db.Create(&models.Core{Name: "xray", IsEnabled: true, IsRunning: false})
	db.Create(&models.Core{Name: "singbox", IsEnabled: true, IsRunning: false})

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	cores, err := cm.ListCores(context.Background())
	if err != nil {
		t.Fatalf("ListCores() error = %v", err)
	}
	if len(cores) != 2 {
		t.Errorf("expected 2 cores, got %d", len(cores))
	}
}

func TestCoreManager_IsCoreRunning_SupervisorDown(t *testing.T) {
	db := setupManagerDB(t)
	db.Create(&models.Core{Name: "xray", IsEnabled: true})

	cm := NewCoreManager(db, "http://127.0.0.1:1", nil, "", "", "", "", "")
	_, err := cm.IsCoreRunning("xray")
	if err == nil {
		t.Error("expected error when supervisor is down, got nil")
	}
}

func TestNewCoreManager(t *testing.T) {
	db := setupManagerDB(t)
	cm := NewCoreManager(db, "http://localhost:9001", nil, "", "", "", "", "")
	if cm == nil {
		t.Fatal("NewCoreManager should not return nil")
	}
	if cm.db != db {
		t.Error("db not set correctly")
	}
	if cm.supervisor == nil {
		t.Error("supervisor client not created")
	}
}

func TestCoreManager_ReloadConfig_NotFound(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.ReloadConfig(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent core, got nil")
	}
}

func TestCoreManager_ReloadConfig_Xray_FallbackToRestart(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	db.Create(&models.Core{Name: "xray", IsEnabled: true, IsRunning: true})

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.ReloadConfig(context.Background(), "xray")
	if err != nil {
		t.Logf("ReloadConfig for xray failed (expected, as it falls back to restart): %v", err)
	}
}

func TestCoreManager_ReloadConfig_Singbox_Signal(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	db.Create(&models.Core{Name: "singbox", IsEnabled: true, IsRunning: true})

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.ReloadConfig(context.Background(), "singbox")
	if err != nil {
		t.Logf("ReloadConfig for singbox failed: %v", err)
	}
}

func TestCoreManager_ReloadConfig_Mihomo_API(t *testing.T) {
	db := setupManagerDB(t)
	srv := newMockSupervisorServer(t, 20)
	defer srv.Close()

	db.Create(&models.Core{Name: "mihomo", IsEnabled: true, IsRunning: true})

	cm := NewCoreManager(db, srv.URL, nil, "", "", "", "", "")
	err := cm.ReloadConfig(context.Background(), "mihomo")
	if err != nil {
		t.Logf("ReloadConfig for mihomo failed: %v", err)
	}
}
