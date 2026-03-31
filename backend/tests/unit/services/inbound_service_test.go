package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

func TestInboundService_ListInbounds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	// Create test inbounds
	core := testutil.GetTestCore(t, db, "singbox")
	_ = testutil.CreateTestInbound(t, db, "VMess-443", core.ID)
	_ = testutil.CreateTestInbound(t, db, "VLESS-8443", core.ID)

	// Create service with nil lifecycle manager
	service := services.NewInboundService(db, nil, nil)

	t.Run("list all inbounds", func(t *testing.T) {
		inbounds, err := service.ListInbounds(nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(inbounds), 2)
	})

	t.Run("list enabled inbounds", func(t *testing.T) {
		enabled := true
		inbounds, err := service.ListInbounds(nil, &enabled)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(inbounds), 0)
	})
}

func TestInboundService_GetInbound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	inbound := testutil.CreateTestInbound(t, db, "VMess-443", core.ID)

	service := services.NewInboundService(db, nil, nil)

	t.Run("get existing inbound", func(t *testing.T) {
		found, err := service.GetInbound(inbound.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "VMess-443", found.Name)
	})

	t.Run("get non-existing inbound", func(t *testing.T) {
		found, err := service.GetInbound(999)
		require.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestInboundService_UpdateInbound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	inbound := testutil.CreateTestInbound(t, db, "VMess-443", core.ID)

	service := services.NewInboundService(db, nil, nil)

	t.Run("update inbound port", func(t *testing.T) {
		updates := map[string]interface{}{"port": 10443}
		updated, err := service.UpdateInbound(inbound.ID, updates)
		require.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, 10443, updated.Port)
	})

	t.Run("update non-existing inbound", func(t *testing.T) {
		updates := map[string]interface{}{"port": 10443}
		updated, err := service.UpdateInbound(999, updates)
		require.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInboundService_DeleteInbound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	inbound := testutil.CreateTestInbound(t, db, "VMess-443", core.ID)

	service := services.NewInboundService(db, nil, nil)

	t.Run("delete existing inbound", func(t *testing.T) {
		err := service.DeleteInbound(inbound.ID)
		require.NoError(t, err)

		// Verify deleted
		found, err := service.GetInbound(inbound.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("delete non-existing inbound", func(t *testing.T) {
		err := service.DeleteInbound(999)
		require.Error(t, err)
	})
}
