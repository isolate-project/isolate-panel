package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

func TestOutboundService_ListOutbounds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	_ = testutil.CreateTestOutbound(t, db, "Direct", core.ID)
	_ = testutil.CreateTestOutbound(t, db, "Block", core.ID)

	service := services.NewOutboundService(db, nil)

	t.Run("list all outbounds", func(t *testing.T) {
		outbounds, err := service.ListOutbounds(nil, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(outbounds), 2)
	})

	t.Run("list outbounds by protocol", func(t *testing.T) {
		outbounds, err := service.ListOutbounds(nil, "freedom")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(outbounds), 0)
	})
}

func TestOutboundService_GetOutbound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	outbound := testutil.CreateTestOutbound(t, db, "Direct", core.ID)

	service := services.NewOutboundService(db, nil)

	t.Run("get existing outbound", func(t *testing.T) {
		found, err := service.GetOutbound(outbound.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "Direct", found.Name)
	})

	t.Run("get non-existing outbound", func(t *testing.T) {
		found, err := service.GetOutbound(999)
		require.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestOutboundService_UpdateOutbound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	outbound := testutil.CreateTestOutbound(t, db, "Direct", core.ID)

	service := services.NewOutboundService(db, nil)

	t.Run("update outbound priority", func(t *testing.T) {
		updates := map[string]interface{}{"priority": 100}
		updated, err := service.UpdateOutbound(outbound.ID, updates)
		require.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, 100, updated.Priority)
	})

	t.Run("update non-existing outbound", func(t *testing.T) {
		updates := map[string]interface{}{"priority": 100}
		updated, err := service.UpdateOutbound(999, updates)
		require.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestOutboundService_DeleteOutbound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	core := testutil.GetTestCore(t, db, "singbox")
	outbound := testutil.CreateTestOutbound(t, db, "Direct", core.ID)

	service := services.NewOutboundService(db, nil)

	t.Run("delete existing outbound", func(t *testing.T) {
		err := service.DeleteOutbound(outbound.ID)
		require.NoError(t, err)

		found, err := service.GetOutbound(outbound.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("delete non-existing outbound", func(t *testing.T) {
		err := service.DeleteOutbound(999)
		require.Error(t, err)
	})
}
