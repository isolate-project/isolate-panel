package services

import (
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInboundTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.Core{},
		&models.Inbound{},
		&models.User{},
		&models.UserInboundMapping{},
		&models.Certificate{},
		&models.Setting{},
	))
	return db
}

func seedInboundCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "sing-box", Version: "1.13.8", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func seedSecondInboundCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "xray", Version: "26.3.27", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func seedInboundUser(t *testing.T, db *gorm.DB, username string) models.User {
	t.Helper()
	user := models.User{Username: username, UUID: username + "-uuid", Password: "pass", SubscriptionToken: username + "-token"}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func newInboundSvc(t *testing.T) (*InboundService, *gorm.DB) {
	t.Helper()
	db := setupInboundTestDB(t)
	return NewInboundService(db, nil, nil), db
}

// ---------------------------------------------------------------------------
// CreateInbound
// ---------------------------------------------------------------------------

func TestInboundService_CreateInbound_ValidData(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	inbound := &models.Inbound{
		Name:          "vless-in",
		Protocol:      "vless",
		CoreID:        core.ID,
		Port:          443,
		ListenAddress: "0.0.0.0",
		IsEnabled:     true,
	}
	err := svc.CreateInbound(inbound)
	require.NoError(t, err)
	assert.NotZero(t, inbound.ID)

	found, err := svc.GetInbound(inbound.ID)
	require.NoError(t, err)
	assert.Equal(t, "vless-in", found.Name)
	assert.Equal(t, "vless", found.Protocol)
	assert.Equal(t, core.ID, found.CoreID)
	assert.Equal(t, 443, found.Port)
	assert.Equal(t, "0.0.0.0", found.ListenAddress)
	assert.True(t, found.IsEnabled)
}

func TestInboundService_CreateInbound_DefaultListenAddress(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	inbound := &models.Inbound{
		Name:     "vmess-in",
		Protocol: "vmess",
		CoreID:   core.ID,
		Port:     10086,
	}
	err := svc.CreateInbound(inbound)
	require.NoError(t, err)

	found, err := svc.GetInbound(inbound.ID)
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", found.ListenAddress)
}

func TestInboundService_CreateInbound_ValidationRequiredFields(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	t.Run("name is required", func(t *testing.T) {
		err := svc.CreateInbound(&models.Inbound{Name: "", Protocol: "vless", CoreID: core.ID, Port: 443})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("protocol is required", func(t *testing.T) {
		err := svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "", CoreID: core.ID, Port: 443})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "protocol is required")
	})

	t.Run("port is required", func(t *testing.T) {
		err := svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 0})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "port is required")
	})

	t.Run("core_id is required", func(t *testing.T) {
		err := svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: 0, Port: 443})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "core_id is required")
	})
}

func TestInboundService_CreateInbound_PortConflictSameCore(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	err := svc.CreateInbound(&models.Inbound{Name: "in1", Protocol: "vless", CoreID: core.ID, Port: 443})
	require.NoError(t, err)

	err = svc.CreateInbound(&models.Inbound{Name: "in2", Protocol: "vmess", CoreID: core.ID, Port: 443})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port 443 is already in use")
}

func TestInboundService_CreateInbound_SamePortDifferentCore(t *testing.T) {
	svc, db := newInboundSvc(t)
	core1 := seedInboundCore(t, db)
	core2 := seedSecondInboundCore(t, db)

	err := svc.CreateInbound(&models.Inbound{Name: "in1", Protocol: "vless", CoreID: core1.ID, Port: 443})
	require.NoError(t, err)

	err = svc.CreateInbound(&models.Inbound{Name: "in2", Protocol: "vless", CoreID: core2.ID, Port: 443})
	assert.NoError(t, err)
}

func TestInboundService_CreateInbound_DuplicateNameAllowed(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	err := svc.CreateInbound(&models.Inbound{Name: "duplicate", Protocol: "vless", CoreID: core.ID, Port: 443})
	require.NoError(t, err)

	err = svc.CreateInbound(&models.Inbound{Name: "duplicate", Protocol: "vmess", CoreID: core.ID, Port: 8443})
	assert.NoError(t, err)

	inbounds, err := svc.ListInbounds(nil, nil)
	require.NoError(t, err)
	assert.Len(t, inbounds, 2)
}

func TestInboundService_CreateInbound_InvalidConfigJSON(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	err := svc.CreateInbound(&models.Inbound{
		Name:       "bad-config",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		ConfigJSON: `{invalid`,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestInboundService_CreateInbound_ValidConfigJSON(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	err := svc.CreateInbound(&models.Inbound{
		Name:       "good-config",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		ConfigJSON: `{"uuid":"abc-def-123","users":[{"uuid":"abc"}]}`,
	})
	require.NoError(t, err)

	var fetched models.Inbound
	require.NoError(t, db.Where("name = ?", "good-config").First(&fetched).Error)
	assert.Equal(t, `{"uuid":"abc-def-123","users":[{"uuid":"abc"}]}`, fetched.ConfigJSON)
}

func TestInboundService_CreateInbound_TLSDisabledClearsCert(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	inbound := &models.Inbound{
		Name:       "no-tls",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: false,
	}
	err := svc.CreateInbound(inbound)
	require.NoError(t, err)

	var fetched models.Inbound
	require.NoError(t, db.First(&fetched, inbound.ID).Error)
	assert.Nil(t, fetched.TLSCertID)
}

func TestInboundService_CreateInbound_TLSCertNotFound(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	badCertID := uint(9999)
	inbound := &models.Inbound{
		Name:       "tls-missing-cert",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		TLSCertID:  &badCertID,
	}
	err := svc.CreateInbound(inbound)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "certificate not found")
}

func TestInboundService_CreateInbound_TLSCertExpired(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	cert := models.Certificate{
		Domain: "expired.example.com",
		Status: models.CertificateStatusExpired,
	}
	require.NoError(t, db.Create(&cert).Error)

	inbound := &models.Inbound{
		Name:       "tls-expired-cert",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		TLSCertID:  &cert.ID,
	}
	err := svc.CreateInbound(inbound)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestInboundService_CreateInbound_TLSCertRevoked(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	cert := models.Certificate{
		Domain: "revoked.example.com",
		Status: models.CertificateStatusRevoked,
	}
	require.NoError(t, db.Create(&cert).Error)

	inbound := &models.Inbound{
		Name:       "tls-revoked-cert",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		TLSCertID:  &cert.ID,
	}
	err := svc.CreateInbound(inbound)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestInboundService_CreateInbound_TLSCertActive(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	cert := models.Certificate{
		Domain: "active.example.com",
		Status: models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(&cert).Error)

	inbound := &models.Inbound{
		Name:       "tls-active-cert",
		Protocol:   "vless",
		CoreID:     core.ID,
		Port:       443,
		TLSEnabled: true,
		TLSCertID:  &cert.ID,
	}
	err := svc.CreateInbound(inbound)
	require.NoError(t, err)

	var fetched models.Inbound
	require.NoError(t, db.First(&fetched, inbound.ID).Error)
	require.NotNil(t, fetched.TLSCertID)
	assert.Equal(t, cert.ID, *fetched.TLSCertID)
}

// ---------------------------------------------------------------------------
// GetInbound
// ---------------------------------------------------------------------------

func TestInboundService_GetInbound_Found(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "get-me", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "get-me").First(&created).Error)

	found, err := svc.GetInbound(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "get-me", found.Name)
	assert.Equal(t, "vless", found.Protocol)
	assert.Equal(t, core.ID, found.CoreID)
	assert.Equal(t, 443, found.Port)
}

func TestInboundService_GetInbound_NotFound(t *testing.T) {
	svc, _ := newInboundSvc(t)

	found, err := svc.GetInbound(9999)
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Contains(t, err.Error(), "inbound not found")
}

// ---------------------------------------------------------------------------
// ListInbounds
// ---------------------------------------------------------------------------

func TestInboundService_ListInbounds_NoFilters(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in1", Protocol: "vless", CoreID: core.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in2", Protocol: "vmess", CoreID: core.ID, Port: 8443}))

	inbounds, err := svc.ListInbounds(nil, nil)
	require.NoError(t, err)
	assert.Len(t, inbounds, 2)
}

func TestInboundService_ListInbounds_CoreIDFilter(t *testing.T) {
	svc, db := newInboundSvc(t)
	core1 := seedInboundCore(t, db)
	core2 := seedSecondInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c1-in1", Protocol: "vless", CoreID: core1.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c1-in2", Protocol: "vmess", CoreID: core1.ID, Port: 8443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c2-in1", Protocol: "vless", CoreID: core2.ID, Port: 2053}))

	inbounds, err := svc.ListInbounds(&core1.ID, nil)
	require.NoError(t, err)
	assert.Len(t, inbounds, 2)
	for _, ib := range inbounds {
		assert.Equal(t, core1.ID, ib.CoreID)
	}

	inbounds, err = svc.ListInbounds(&core2.ID, nil)
	require.NoError(t, err)
	assert.Len(t, inbounds, 1)
	assert.Equal(t, "c2-in1", inbounds[0].Name)
}

func TestInboundService_ListInbounds_IsEnabledFilter(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "enabled", Protocol: "vless", CoreID: core.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "disabled", Protocol: "vmess", CoreID: core.ID, Port: 8443}))
	db.Model(&models.Inbound{}).Where("name = ? AND core_id = ?", "disabled", core.ID).Update("is_enabled", false)

	enabled := true
	inbounds, err := svc.ListInbounds(nil, &enabled)
	require.NoError(t, err)
	assert.Len(t, inbounds, 1)
	assert.Equal(t, "enabled", inbounds[0].Name)

	disabled := false
	inbounds, err = svc.ListInbounds(nil, &disabled)
	require.NoError(t, err)
	assert.Len(t, inbounds, 1)
	assert.Equal(t, "disabled", inbounds[0].Name)
}

func TestInboundService_ListInbounds_CombinedFilters(t *testing.T) {
	svc, db := newInboundSvc(t)
	core1 := seedInboundCore(t, db)
	core2 := seedSecondInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c1-on", Protocol: "vless", CoreID: core1.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c1-off", Protocol: "vmess", CoreID: core1.ID, Port: 8443}))
	db.Model(&models.Inbound{}).Where("name = ? AND core_id = ?", "c1-off", core1.ID).Update("is_enabled", false)
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c2-on", Protocol: "vless", CoreID: core2.ID, Port: 2053}))

	enabled := true
	inbounds, err := svc.ListInbounds(&core1.ID, &enabled)
	require.NoError(t, err)
	assert.Len(t, inbounds, 1)
	assert.Equal(t, "c1-on", inbounds[0].Name)
}

func TestInboundService_ListInbounds_Empty(t *testing.T) {
	svc, _ := newInboundSvc(t)

	inbounds, err := svc.ListInbounds(nil, nil)
	require.NoError(t, err)
	assert.Empty(t, inbounds)
}

func TestInboundService_ListInbounds_OrderByID(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "third", Protocol: "vless", CoreID: core.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "first", Protocol: "vmess", CoreID: core.ID, Port: 8443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "second", Protocol: "trojan", CoreID: core.ID, Port: 2053}))

	inbounds, err := svc.ListInbounds(nil, nil)
	require.NoError(t, err)
	assert.Len(t, inbounds, 3)
	for i := 1; i < len(inbounds); i++ {
		assert.Less(t, inbounds[i-1].ID, inbounds[i].ID)
	}
}

// ---------------------------------------------------------------------------
// UpdateInbound
// ---------------------------------------------------------------------------

func TestInboundService_UpdateInbound_Name(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "old-name", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "old-name").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"name": "new-name"})
	require.NoError(t, err)
	assert.Equal(t, "new-name", updated.Name)
	assert.Equal(t, "vless", updated.Protocol)
}

func TestInboundService_UpdateInbound_Port(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"port": 8443})
	require.NoError(t, err)
	assert.Equal(t, 8443, updated.Port)
}

func TestInboundService_UpdateInbound_PortConflict(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in1", Protocol: "vless", CoreID: core.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in2", Protocol: "vmess", CoreID: core.ID, Port: 8443}))
	var in2 models.Inbound
	require.NoError(t, db.Where("name = ?", "in2").First(&in2).Error)

	updated, err := svc.UpdateInbound(in2.ID, map[string]interface{}{"port": 443})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "port 443 is already in use")
}

func TestInboundService_UpdateInbound_SamePortNoConflict(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"port": 443})
	require.NoError(t, err)
	assert.Equal(t, 443, updated.Port)
}

func TestInboundService_UpdateInbound_IsEnabled(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443, IsEnabled: true}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"is_enabled": false})
	require.NoError(t, err)
	assert.False(t, updated.IsEnabled)
}

func TestInboundService_UpdateInbound_ConfigJSON(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443, ConfigJSON: `{"uuid":"test-uuid-123"}`}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"config_json": `{"uuid":"test-uuid-123","tag":"updated"}`})
	require.NoError(t, err)
	assert.Equal(t, `{"uuid":"test-uuid-123","tag":"updated"}`, updated.ConfigJSON)
}

func TestInboundService_UpdateInbound_InvalidConfigJSON(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"config_json": `{bad`})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestInboundService_UpdateInbound_TLSDisabledClearsCert(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	cert := models.Certificate{Domain: "clear.example.com", Status: models.CertificateStatusActive}
	require.NoError(t, db.Create(&cert).Error)

	require.NoError(t, svc.CreateInbound(&models.Inbound{
		Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443,
		TLSEnabled: true, TLSCertID: &cert.ID,
	}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"tls_enabled": false})
	require.NoError(t, err)
	assert.False(t, updated.TLSEnabled)

	var fetched models.Inbound
	require.NoError(t, db.First(&fetched, created.ID).Error)
	assert.Nil(t, fetched.TLSCertID)
}

func TestInboundService_UpdateInbound_TLSCertExpired(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	cert := models.Certificate{Domain: "exp2.example.com", Status: models.CertificateStatusExpired}
	require.NoError(t, db.Create(&cert).Error)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"tls_cert_id": cert.ID})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "expired")
}

func TestInboundService_UpdateInbound_NotFound(t *testing.T) {
	svc, _ := newInboundSvc(t)

	updated, err := svc.UpdateInbound(9999, map[string]interface{}{"name": "x"})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "inbound not found")
}

// ---------------------------------------------------------------------------
// DeleteInbound
// ---------------------------------------------------------------------------

func TestInboundService_DeleteInbound_Found(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "del-me", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "del-me").First(&created).Error)

	err := svc.DeleteInbound(created.ID)
	require.NoError(t, err)

	found, err := svc.GetInbound(created.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestInboundService_DeleteInbound_NotFound(t *testing.T) {
	svc, _ := newInboundSvc(t)

	err := svc.DeleteInbound(9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inbound not found")
}

// ---------------------------------------------------------------------------
// GetInboundsByCore
// ---------------------------------------------------------------------------

func TestInboundService_GetInboundsByCore(t *testing.T) {
	svc, db := newInboundSvc(t)
	core1 := seedInboundCore(t, db)
	core2 := seedSecondInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c1-in1", Protocol: "vless", CoreID: core1.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c1-in2", Protocol: "vmess", CoreID: core1.ID, Port: 8443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "c2-in1", Protocol: "vless", CoreID: core2.ID, Port: 2053}))

	inbounds, err := svc.GetInboundsByCore(core1.ID)
	require.NoError(t, err)
	assert.Len(t, inbounds, 2)
	for _, ib := range inbounds {
		assert.Equal(t, core1.ID, ib.CoreID)
		assert.NotNil(t, ib.Core)
	}
}

func TestInboundService_GetInboundsByCore_Empty(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	inbounds, err := svc.GetInboundsByCore(core.ID)
	require.NoError(t, err)
	assert.Empty(t, inbounds)
}

// ---------------------------------------------------------------------------
// GetInboundsByCoreName
// ---------------------------------------------------------------------------

func TestInboundService_GetInboundsByCoreName(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "by-name", Protocol: "vless", CoreID: core.ID, Port: 443}))

	inbounds, err := svc.GetInboundsByCoreName("sing-box")
	require.NoError(t, err)
	assert.Len(t, inbounds, 1)
	assert.Equal(t, "by-name", inbounds[0].Name)
}

func TestInboundService_GetInboundsByCoreName_NotFound(t *testing.T) {
	svc, _ := newInboundSvc(t)

	inbounds, err := svc.GetInboundsByCoreName("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, inbounds)
	assert.Contains(t, err.Error(), "core not found")
}

// ---------------------------------------------------------------------------
// GetInboundsByUser
// ---------------------------------------------------------------------------

func TestInboundService_GetInboundsByUser(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user := seedInboundUser(t, db, "testuser")

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "user-in1", Protocol: "vless", CoreID: core.ID, Port: 443}))
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "user-in2", Protocol: "vmess", CoreID: core.ID, Port: 8443}))
	var in1, in2 models.Inbound
	require.NoError(t, db.Where("name = ?", "user-in1").First(&in1).Error)
	require.NoError(t, db.Where("name = ?", "user-in2").First(&in2).Error)

	require.NoError(t, svc.AssignInboundToUser(user.ID, in1.ID))
	require.NoError(t, svc.AssignInboundToUser(user.ID, in2.ID))

	inbounds, err := svc.GetInboundsByUser(user.ID)
	require.NoError(t, err)
	assert.Len(t, inbounds, 2)
}

func TestInboundService_GetInboundsByUser_NoAssignments(t *testing.T) {
	svc, db := newInboundSvc(t)
	user := seedInboundUser(t, db, "no-inbounds")

	inbounds, err := svc.GetInboundsByUser(user.ID)
	require.NoError(t, err)
	assert.Empty(t, inbounds)
}

// ---------------------------------------------------------------------------
// AssignInboundToUser / UnassignInboundFromUser
// ---------------------------------------------------------------------------

func TestInboundService_AssignInboundToUser(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user := seedInboundUser(t, db, "assign-user")
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "assign-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "assign-in").First(&inbound).Error)

	err := svc.AssignInboundToUser(user.ID, inbound.ID)
	require.NoError(t, err)

	var mapping models.UserInboundMapping
	require.NoError(t, db.Where("user_id = ? AND inbound_id = ?", user.ID, inbound.ID).First(&mapping).Error)
	assert.Equal(t, user.ID, mapping.UserID)
	assert.Equal(t, inbound.ID, mapping.InboundID)
}

func TestInboundService_AssignInboundToUser_Duplicate(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user := seedInboundUser(t, db, "dup-assign")
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "dup-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "dup-in").First(&inbound).Error)

	require.NoError(t, svc.AssignInboundToUser(user.ID, inbound.ID))

	err := svc.AssignInboundToUser(user.ID, inbound.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already assigned")
}

func TestInboundService_AssignInboundToUser_UserNotFound(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&inbound).Error)

	err := svc.AssignInboundToUser(9999, inbound.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestInboundService_AssignInboundToUser_InboundNotFound(t *testing.T) {
	svc, db := newInboundSvc(t)
	user := seedInboundUser(t, db, "no-inbound-user")

	err := svc.AssignInboundToUser(user.ID, 9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inbound not found")
}

func TestInboundService_UnassignInboundFromUser(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user := seedInboundUser(t, db, "unassign-user")
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "un-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "un-in").First(&inbound).Error)

	require.NoError(t, svc.AssignInboundToUser(user.ID, inbound.ID))

	err := svc.UnassignInboundFromUser(user.ID, inbound.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.UserInboundMapping{}).Where("user_id = ? AND inbound_id = ?", user.ID, inbound.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestInboundService_UnassignInboundFromUser_NotMapped(t *testing.T) {
	svc, db := newInboundSvc(t)
	user := seedInboundUser(t, db, "un-nomatch")

	err := svc.UnassignInboundFromUser(user.ID, 9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mapping not found")
}

// ---------------------------------------------------------------------------
// GetInboundUsers
// ---------------------------------------------------------------------------

func TestInboundService_GetInboundUsers(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user1 := seedInboundUser(t, db, "u1")
	user2 := seedInboundUser(t, db, "u2")
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "multi-user-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "multi-user-in").First(&inbound).Error)

	require.NoError(t, svc.AssignInboundToUser(user1.ID, inbound.ID))
	require.NoError(t, svc.AssignInboundToUser(user2.ID, inbound.ID))

	users, err := svc.GetInboundUsers(inbound.ID)
	require.NoError(t, err)
	assert.Len(t, users, 2)

	usernames := map[string]bool{}
	for _, u := range users {
		usernames[u.Username] = true
	}
	assert.True(t, usernames["u1"])
	assert.True(t, usernames["u2"])
}

func TestInboundService_GetInboundUsers_NoUsers(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "no-users-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "no-users-in").First(&inbound).Error)

	users, err := svc.GetInboundUsers(inbound.ID)
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestInboundService_GetInboundUsers_InboundNotFound(t *testing.T) {
	svc, _ := newInboundSvc(t)

	users, err := svc.GetInboundUsers(9999)
	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "inbound not found")
}

// ---------------------------------------------------------------------------
// BulkAssignUsers
// ---------------------------------------------------------------------------

func TestInboundService_BulkAssignUsers(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user1 := seedInboundUser(t, db, "bulk1")
	user2 := seedInboundUser(t, db, "bulk2")
	user3 := seedInboundUser(t, db, "bulk3")
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "bulk-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "bulk-in").First(&inbound).Error)

	require.NoError(t, svc.AssignInboundToUser(user1.ID, inbound.ID))
	require.NoError(t, svc.AssignInboundToUser(user2.ID, inbound.ID))

	added, removed, err := svc.BulkAssignUsers(inbound.ID, []uint{user3.ID}, []uint{user1.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 1, removed)

	users, err := svc.GetInboundUsers(inbound.ID)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	usernames := map[string]bool{}
	for _, u := range users {
		usernames[u.Username] = true
	}
	assert.True(t, usernames["bulk2"])
	assert.True(t, usernames["bulk3"])
	assert.False(t, usernames["bulk1"])
}

func TestInboundService_BulkAssignUsers_InboundNotFound(t *testing.T) {
	svc, _ := newInboundSvc(t)

	added, removed, err := svc.BulkAssignUsers(9999, nil, nil)
	assert.Error(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 0, removed)
	assert.Contains(t, err.Error(), "inbound not found")
}

func TestInboundService_BulkAssignUsers_DuplicateAdd(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	user := seedInboundUser(t, db, "bulkdup")
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "bulk-dup-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "bulk-dup-in").First(&inbound).Error)

	require.NoError(t, svc.AssignInboundToUser(user.ID, inbound.ID))

	added, removed, err := svc.BulkAssignUsers(inbound.ID, []uint{user.ID}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 0, removed)
}

func TestInboundService_BulkAssignUsers_NonexistentUser(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)
	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "bulk-bad-in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var inbound models.Inbound
	require.NoError(t, db.Where("name = ?", "bulk-bad-in").First(&inbound).Error)

	added, removed, err := svc.BulkAssignUsers(inbound.ID, []uint{9999}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 0, removed)
}

// ---------------------------------------------------------------------------
// ValidateInboundConfig
// ---------------------------------------------------------------------------

func TestInboundService_ValidateInboundConfig_Empty(t *testing.T) {
	svc, _ := newInboundSvc(t)

	err := svc.ValidateInboundConfig("vless", "")
	assert.NoError(t, err)
}

func TestInboundService_ValidateInboundConfig_ValidJSON(t *testing.T) {
	svc, _ := newInboundSvc(t)

	err := svc.ValidateInboundConfig("vless", `{"uuid":"abc-def-123","users":[{"uuid":"abc"}]}`)
	assert.NoError(t, err)
}

func TestInboundService_ValidateInboundConfig_InvalidJSON(t *testing.T) {
	svc, _ := newInboundSvc(t)

	err := svc.ValidateInboundConfig("vless", `{not-json}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

// ---------------------------------------------------------------------------
// Port conflict — update with different port type representations
// ---------------------------------------------------------------------------

func TestInboundService_UpdateInbound_PortAsFloat64(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"port": float64(2053)})
	require.NoError(t, err)
	assert.Equal(t, 2053, updated.Port)
}

func TestInboundService_UpdateInbound_PortAsUint(t *testing.T) {
	svc, db := newInboundSvc(t)
	core := seedInboundCore(t, db)

	require.NoError(t, svc.CreateInbound(&models.Inbound{Name: "in", Protocol: "vless", CoreID: core.ID, Port: 443}))
	var created models.Inbound
	require.NoError(t, db.Where("name = ?", "in").First(&created).Error)

	updated, err := svc.UpdateInbound(created.ID, map[string]interface{}{"port": uint(2053)})
	require.NoError(t, err)
	assert.Equal(t, 2053, updated.Port)
}
