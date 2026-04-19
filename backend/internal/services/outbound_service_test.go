package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

func setupOutboundTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&models.Core{}, &models.Outbound{}, &models.Setting{}))
	return db
}

func seedOutboundCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "sing-box", Version: "1.13.8", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func seedSecondCore(t *testing.T, db *gorm.DB) models.Core {
	t.Helper()
	core := models.Core{Name: "xray", Version: "26.3.27", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	return core
}

func TestOutboundService_CreateOutbound_Validation(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	t.Run("name is required", func(t *testing.T) {
		err := svc.CreateOutbound(&models.Outbound{Name: "", Protocol: "direct", CoreID: core.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("protocol is required", func(t *testing.T) {
		err := svc.CreateOutbound(&models.Outbound{Name: "MyOut", Protocol: "", CoreID: core.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "protocol is required")
	})

	t.Run("core_id is required", func(t *testing.T) {
		err := svc.CreateOutbound(&models.Outbound{Name: "MyOut", Protocol: "direct", CoreID: 0})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "core_id is required")
	})
}

func TestOutboundService_CreateOutbound_CoreNotFound(t *testing.T) {
	db := setupOutboundTestDB(t)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{Name: "MyOut", Protocol: "direct", CoreID: 9999})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "core not found")
}

func TestOutboundService_CreateOutbound_UnknownProtocol(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{Name: "MyOut", Protocol: "nonexistent", CoreID: core.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown protocol")
}

func TestOutboundService_CreateOutbound_ProtocolNotSupportedByCore(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := models.Core{Name: "xray", Version: "26.3.27", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{Name: "TorOut", Protocol: "tor", CoreID: core.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported by core")
}

func TestOutboundService_CreateOutbound_InboundOnlyProtocol(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{Name: "MixedOut", Protocol: "mixed", CoreID: core.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inbound-only")
}

func TestOutboundService_CreateOutbound_InvalidConfigJSON(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{
		Name:       "MyOut",
		Protocol:   "direct",
		CoreID:     core.ID,
		ConfigJSON: `{invalid`,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config_json")
}

func TestOutboundService_CreateOutbound_DuplicateName(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "direct", CoreID: core.ID})
	require.NoError(t, err)

	err = svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "block", CoreID: core.ID})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestOutboundService_CreateOutbound_DuplicateNameDifferentCore(t *testing.T) {
	db := setupOutboundTestDB(t)
	core1 := seedOutboundCore(t, db)
	core2 := seedSecondCore(t, db)
	svc := NewOutboundService(db, nil)

	err := svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "direct", CoreID: core1.ID})
	require.NoError(t, err)

	err = svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "direct", CoreID: core2.ID})
	assert.NoError(t, err)
}

func TestOutboundService_CreateOutbound_Success(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	outbound := &models.Outbound{
		Name:       "DirectOut",
		Protocol:   "direct",
		CoreID:     core.ID,
		ConfigJSON: `{"tag":"direct"}`,
		Priority:   10,
		IsEnabled:  true,
	}
	err := svc.CreateOutbound(outbound)
	require.NoError(t, err)
	assert.NotZero(t, outbound.ID)

	found, err := svc.GetOutbound(outbound.ID)
	require.NoError(t, err)
	assert.Equal(t, "DirectOut", found.Name)
	assert.Equal(t, "direct", found.Protocol)
	assert.Equal(t, core.ID, found.CoreID)
	assert.Equal(t, 10, found.Priority)
	assert.True(t, found.IsEnabled)
}

func TestOutboundService_CreateOutbound_EmptyConfigDefaultsToEmptyObject(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	outbound := &models.Outbound{
		Name:     "BlockOut",
		Protocol: "block",
		CoreID:   core.ID,
	}
	err := svc.CreateOutbound(outbound)
	require.NoError(t, err)

	var fetched models.Outbound
	require.NoError(t, db.First(&fetched, outbound.ID).Error)
	assert.Equal(t, "{}", fetched.ConfigJSON)
}

func TestOutboundService_GetOutbound_Found(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "DirectOut").First(&created).Error)

	found, err := svc.GetOutbound(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "DirectOut", found.Name)
	assert.NotNil(t, found.Core)
	assert.Equal(t, "sing-box", found.Core.Name)
}

func TestOutboundService_GetOutbound_NotFound(t *testing.T) {
	db := setupOutboundTestDB(t)
	svc := NewOutboundService(db, nil)

	found, err := svc.GetOutbound(9999)
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Contains(t, err.Error(), "not found")
}

func TestOutboundService_ListOutbounds_NoFilters(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "direct", CoreID: core.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "BlockOut", Protocol: "block", CoreID: core.ID}))

	outbounds, err := svc.ListOutbounds(nil, "")
	require.NoError(t, err)
	assert.Len(t, outbounds, 2)
}

func TestOutboundService_ListOutbounds_CoreIDFilter(t *testing.T) {
	db := setupOutboundTestDB(t)
	core1 := seedOutboundCore(t, db)
	core2 := seedSecondCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Direct1", Protocol: "direct", CoreID: core1.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Block1", Protocol: "block", CoreID: core1.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Direct2", Protocol: "direct", CoreID: core2.ID}))

	outbounds, err := svc.ListOutbounds(&core1.ID, "")
	require.NoError(t, err)
	assert.Len(t, outbounds, 2)
	for _, o := range outbounds {
		assert.Equal(t, core1.ID, o.CoreID)
	}

	outbounds, err = svc.ListOutbounds(&core2.ID, "")
	require.NoError(t, err)
	assert.Len(t, outbounds, 1)
	assert.Equal(t, "Direct2", outbounds[0].Name)
}

func TestOutboundService_ListOutbounds_ProtocolFilter(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "DirectOut", Protocol: "direct", CoreID: core.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "BlockOut", Protocol: "block", CoreID: core.ID}))

	outbounds, err := svc.ListOutbounds(nil, "direct")
	require.NoError(t, err)
	assert.Len(t, outbounds, 1)
	assert.Equal(t, "direct", outbounds[0].Protocol)
}

func TestOutboundService_ListOutbounds_CombinedFilters(t *testing.T) {
	db := setupOutboundTestDB(t)
	core1 := seedOutboundCore(t, db)
	core2 := seedSecondCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Direct1", Protocol: "direct", CoreID: core1.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Block1", Protocol: "block", CoreID: core1.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Direct2", Protocol: "direct", CoreID: core2.ID}))

	outbounds, err := svc.ListOutbounds(&core1.ID, "direct")
	require.NoError(t, err)
	assert.Len(t, outbounds, 1)
	assert.Equal(t, "Direct1", outbounds[0].Name)
}

func TestOutboundService_ListOutbounds_IsEnabledField(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Enabled", Protocol: "direct", CoreID: core.ID, IsEnabled: true}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Disabled", Protocol: "block", CoreID: core.ID}))
	db.Model(&models.Outbound{}).Where("name = ? AND core_id = ?", "Disabled", core.ID).Update("is_enabled", false)

	outbounds, err := svc.ListOutbounds(nil, "")
	require.NoError(t, err)
	assert.Len(t, outbounds, 2)

	enabled := make(map[string]bool)
	for _, o := range outbounds {
		enabled[o.Name] = o.IsEnabled
	}
	assert.True(t, enabled["Enabled"])
	assert.False(t, enabled["Disabled"])
}

func TestOutboundService_ListOutbounds_PriorityOrder(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Low", Protocol: "direct", CoreID: core.ID, Priority: 1}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "High", Protocol: "block", CoreID: core.ID, Priority: 100}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Mid", Protocol: "dns", CoreID: core.ID, Priority: 50}))

	outbounds, err := svc.ListOutbounds(nil, "")
	require.NoError(t, err)
	assert.Equal(t, "High", outbounds[0].Name)
	assert.Equal(t, "Mid", outbounds[1].Name)
	assert.Equal(t, "Low", outbounds[2].Name)
}

func TestOutboundService_UpdateOutbound_PartialName(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "OldName", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "OldName").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"name": "NewName"})
	require.NoError(t, err)
	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, "direct", updated.Protocol)
}

func TestOutboundService_UpdateOutbound_PartialPriority(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID, Priority: 0}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"priority": 42})
	require.NoError(t, err)
	assert.Equal(t, 42, updated.Priority)
}

func TestOutboundService_UpdateOutbound_IsEnabled(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID, IsEnabled: true}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"is_enabled": false})
	require.NoError(t, err)
	assert.False(t, updated.IsEnabled)
}

func TestOutboundService_UpdateOutbound_ConfigJSON(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID, ConfigJSON: `{}`}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"config_json": `{"tag":"updated"}`})
	require.NoError(t, err)
	assert.Equal(t, `{"tag":"updated"}`, updated.ConfigJSON)
}

func TestOutboundService_UpdateOutbound_InvalidConfigJSON(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"config_json": `{bad`})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "invalid config_json")
}

func TestOutboundService_UpdateOutbound_DuplicateName(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "First", Protocol: "direct", CoreID: core.ID}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Second", Protocol: "block", CoreID: core.ID}))
	var second models.Outbound
	require.NoError(t, db.Where("name = ?", "Second").First(&second).Error)

	updated, err := svc.UpdateOutbound(second.ID, map[string]interface{}{"name": "First"})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "already exists")
}

func TestOutboundService_UpdateOutbound_Protocol(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"protocol": "block"})
	require.NoError(t, err)
	assert.Equal(t, "block", updated.Protocol)
}

func TestOutboundService_UpdateOutbound_ProtocolInboundOnly(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"protocol": "mixed"})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "inbound-only")
}

func TestOutboundService_UpdateOutbound_ProtocolUnsupportedByCore(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := models.Core{Name: "xray", Version: "26.3.27", IsEnabled: true}
	require.NoError(t, db.Create(&core).Error)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	updated, err := svc.UpdateOutbound(created.ID, map[string]interface{}{"protocol": "tor"})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "not supported by core")
}

func TestOutboundService_UpdateOutbound_NotFound(t *testing.T) {
	db := setupOutboundTestDB(t)
	svc := NewOutboundService(db, nil)

	updated, err := svc.UpdateOutbound(9999, map[string]interface{}{"name": "X"})
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "not found")
}

func TestOutboundService_DeleteOutbound_Found(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Out", Protocol: "direct", CoreID: core.ID}))
	var created models.Outbound
	require.NoError(t, db.Where("name = ?", "Out").First(&created).Error)

	err := svc.DeleteOutbound(created.ID)
	require.NoError(t, err)

	found, err := svc.GetOutbound(created.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestOutboundService_DeleteOutbound_NotFound(t *testing.T) {
	db := setupOutboundTestDB(t)
	svc := NewOutboundService(db, nil)

	err := svc.DeleteOutbound(9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOutboundService_GetOutboundsByCore(t *testing.T) {
	db := setupOutboundTestDB(t)
	core1 := seedOutboundCore(t, db)
	core2 := seedSecondCore(t, db)
	svc := NewOutboundService(db, nil)

	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Direct1", Protocol: "direct", CoreID: core1.ID, Priority: 10}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Block1", Protocol: "block", CoreID: core1.ID, Priority: 20}))
	require.NoError(t, svc.CreateOutbound(&models.Outbound{Name: "Direct2", Protocol: "direct", CoreID: core2.ID}))

	outbounds, err := svc.GetOutboundsByCore(core1.ID)
	require.NoError(t, err)
	assert.Len(t, outbounds, 2)
	assert.Equal(t, "Block1", outbounds[0].Name)
	assert.Equal(t, "Direct1", outbounds[1].Name)
	for _, o := range outbounds {
		assert.Equal(t, core1.ID, o.CoreID)
		assert.NotNil(t, o.Core)
	}
}

func TestOutboundService_GetOutboundsByCore_Empty(t *testing.T) {
	db := setupOutboundTestDB(t)
	core := seedOutboundCore(t, db)
	svc := NewOutboundService(db, nil)

	outbounds, err := svc.GetOutboundsByCore(core.ID)
	require.NoError(t, err)
	assert.Empty(t, outbounds)
}

func TestOutboundService_GetOutboundsByCore_NonexistentCore(t *testing.T) {
	db := setupOutboundTestDB(t)
	svc := NewOutboundService(db, nil)

	outbounds, err := svc.GetOutboundsByCore(9999)
	require.NoError(t, err)
	assert.Empty(t, outbounds)
}
