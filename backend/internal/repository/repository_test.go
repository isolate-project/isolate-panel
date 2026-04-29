package repository

import (
	"context"
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:?_foreign_keys=ON"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate tables
	err = db.AutoMigrate(
		&models.User{},
		&models.Inbound{},
		&models.Outbound{},
		&models.Core{},
		&models.Admin{},
		&models.Backup{},
		&models.UserInboundMapping{},
	)
	require.NoError(t, err)

	return db
}

func TestGORMUserRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGORMUserRepository(db)
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		user := &models.User{
			Username:          "testuser",
			Email:             "test@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440000",
			Password:          "hashedpassword",
			SubscriptionToken: "token123",
			IsActive:          true,
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Get by ID
		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Username, found.Username)
		assert.Equal(t, user.UUID, found.UUID)

		// Get by UUID
		foundByUUID, err := repo.GetByUUID(ctx, user.UUID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, foundByUUID.ID)

		// Get by Token
		foundByToken, err := repo.GetByToken(ctx, user.SubscriptionToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID, foundByToken.ID)
	})

	t.Run("GetByID Not Found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Update", func(t *testing.T) {
		user := &models.User{
			Username:          "updateuser",
			Email:             "update@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440001",
			Password:          "hashedpassword",
			SubscriptionToken: "token456",
			IsActive:          true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		user.Email = "updated@example.com"
		err = repo.Update(ctx, user)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", found.Email)
	})

	t.Run("Delete", func(t *testing.T) {
		user := &models.User{
			Username:          "deleteuser",
			Email:             "delete@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440002",
			Password:          "hashedpassword",
			SubscriptionToken: "token789",
			IsActive:          true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, user.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple users
		for i := 0; i < 5; i++ {
			user := &models.User{
				Username:          "listuser" + string(rune('0'+i)),
				Email:             "list" + string(rune('0'+i)) + "@example.com",
				UUID:              "550e8400-e29b-41d4-a716-44665544000" + string(rune('3'+i)),
				Password:          "hashedpassword",
				SubscriptionToken: "listtoken" + string(rune('0'+i)),
				IsActive:          i%2 == 0,
			}
			err := repo.Create(ctx, user)
			require.NoError(t, err)
		}

		users, total, err := repo.List(ctx, 0, 10, "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(5))
		assert.GreaterOrEqual(t, len(users), 5)

		// Test pagination
		users, total, err = repo.List(ctx, 0, 2, "", "")
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// Test status filter
		users, total, err = repo.List(ctx, 0, 10, "", "active")
		require.NoError(t, err)
		for _, u := range users {
			assert.True(t, u.IsActive)
		}
	})

	t.Run("Search", func(t *testing.T) {
		users, total, err := repo.Search(ctx, "listuser", 0, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.GreaterOrEqual(t, len(users), 1)
	})

	t.Run("UpdateTrafficUsed", func(t *testing.T) {
		user := &models.User{
			Username:          "trafficuser",
			Email:             "traffic@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440010",
			Password:          "hashedpassword",
			SubscriptionToken: "traffictoken",
			IsActive:          true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		err = repo.UpdateTrafficUsed(ctx, user.ID, 1024)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1024), found.TrafficUsedBytes)
	})

	t.Run("UpdateOnlineStatus", func(t *testing.T) {
		user := &models.User{
			Username:          "onlineuser",
			Email:             "online@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440011",
			Password:          "hashedpassword",
			SubscriptionToken: "onlinetoken",
			IsActive:          true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		err = repo.UpdateOnlineStatus(ctx, user.ID, true)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.True(t, found.IsOnline)
	})

	t.Run("UpdateLastConnected", func(t *testing.T) {
		user := &models.User{
			Username:          "lastuser",
			Email:             "last@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440012",
			Password:          "hashedpassword",
			SubscriptionToken: "lasttoken",
			IsActive:          true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		err = repo.UpdateLastConnected(ctx, user.ID)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, found.LastConnectedAt)
	})
}

func TestGORMCoreRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGORMCoreRepository(db)
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		core := &models.Core{
			Name:      "xray",
			Version:   "1.8.0",
			IsEnabled: true,
			APIPort:   10085,
		}

		err := repo.Create(ctx, core)
		require.NoError(t, err)
		assert.NotZero(t, core.ID)

		found, err := repo.GetByID(ctx, core.ID)
		require.NoError(t, err)
		assert.Equal(t, core.Name, found.Name)

		foundByName, err := repo.GetByName(ctx, core.Name)
		require.NoError(t, err)
		assert.Equal(t, core.ID, foundByName.ID)
	})

	t.Run("List", func(t *testing.T) {
		cores, err := repo.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(cores), 1)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		core := &models.Core{
			Name:      "singbox",
			Version:   "1.9.0",
			IsEnabled: true,
			APIPort:   10086,
		}
		err := repo.Create(ctx, core)
		require.NoError(t, err)

		pid := 1234
		err = repo.UpdateStatus(ctx, core.ID, true, &pid)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, core.ID)
		require.NoError(t, err)
		assert.True(t, found.IsRunning)
		require.NotNil(t, found.PID)
		assert.Equal(t, pid, *found.PID)
	})

	t.Run("UpdateHealth", func(t *testing.T) {
		core := &models.Core{
			Name:      "mihomo",
			Version:   "1.18.0",
			IsEnabled: true,
			APIPort:   10087,
		}
		err := repo.Create(ctx, core)
		require.NoError(t, err)

		err = repo.UpdateHealth(ctx, core.ID, "healthy", "")
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, core.ID)
		require.NoError(t, err)
		assert.Equal(t, "healthy", found.HealthStatus)
	})
}

func TestGORMAdminRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGORMAdminRepository(db)
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		admin := &models.Admin{
			Username:     "admin1",
			PasswordHash: "hashedpassword",
			Email:        "admin@example.com",
			IsSuperAdmin: true,
			IsActive:     true,
		}

		err := repo.Create(ctx, admin)
		require.NoError(t, err)
		assert.NotZero(t, admin.ID)

		found, err := repo.GetByID(ctx, admin.ID)
		require.NoError(t, err)
		assert.Equal(t, admin.Username, found.Username)

		foundByUsername, err := repo.GetByUsername(ctx, admin.Username)
		require.NoError(t, err)
		assert.Equal(t, admin.ID, foundByUsername.ID)
	})

	t.Run("UpdateLastLogin", func(t *testing.T) {
		admin := &models.Admin{
			Username:     "admin2",
			PasswordHash: "hashedpassword",
			IsActive:     true,
		}
		err := repo.Create(ctx, admin)
		require.NoError(t, err)

		err = repo.UpdateLastLogin(ctx, admin.ID)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, admin.ID)
		require.NoError(t, err)
		assert.NotNil(t, found.LastLoginAt)
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		admin := &models.Admin{
			Username:     "admin3",
			PasswordHash: "oldpassword",
			IsActive:     true,
		}
		err := repo.Create(ctx, admin)
		require.NoError(t, err)

		err = repo.UpdatePassword(ctx, admin.ID, "newpassword")
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, admin.ID)
		require.NoError(t, err)
		assert.Equal(t, "newpassword", found.PasswordHash)
	})
}

func TestGORMBackupRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGORMBackupRepository(db)
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		backup := &models.Backup{
			Filename:       "backup_2024_01.tar.gz",
			FilePath:       "/backups/backup_2024_01.tar.gz",
			FileSizeBytes:  1024,
			ChecksumSHA256: "abc123",
			BackupType:     models.BackupTypeFull,
			Destination:    models.BackupDestinationLocal,
			Status:         models.BackupStatusPending,
		}

		err := repo.Create(ctx, backup)
		require.NoError(t, err)
		assert.NotZero(t, backup.ID)

		found, err := repo.GetByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Equal(t, backup.Filename, found.Filename)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		backup := &models.Backup{
			Filename:    "backup_status.tar.gz",
			FilePath:    "/backups/backup_status.tar.gz",
			BackupType:  models.BackupTypeManual,
			Destination: models.BackupDestinationLocal,
			Status:      models.BackupStatusPending,
		}
		err := repo.Create(ctx, backup)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, backup.ID, models.BackupStatusRunning, "")
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, backup.ID)
		require.NoError(t, err)
		assert.Equal(t, models.BackupStatusRunning, found.Status)
	})

	t.Run("List", func(t *testing.T) {
		backups, total, err := repo.List(ctx, 0, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.GreaterOrEqual(t, len(backups), 1)
	})
}

func TestGORMInboundRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGORMInboundRepository(db)
	coreRepo := NewGORMCoreRepository(db)
	userRepo := NewGORMUserRepository(db)
	ctx := context.Background()

	// Create a core first
	core := &models.Core{
		Name:      "xray",
		Version:   "1.8.0",
		IsEnabled: true,
		APIPort:   10085,
	}
	err := coreRepo.Create(ctx, core)
	require.NoError(t, err)

	t.Run("Create and Get", func(t *testing.T) {
		inbound := &models.Inbound{
			Name:       "vless_inbound",
			Protocol:   "vless",
			CoreID:     core.ID,
			Port:       443,
			ConfigJSON: `{"port":443,"protocol":"vless"}`,
			IsEnabled:  true,
		}

		err := repo.Create(ctx, inbound)
		require.NoError(t, err)
		assert.NotZero(t, inbound.ID)

		found, err := repo.GetByID(ctx, inbound.ID)
		require.NoError(t, err)
		assert.Equal(t, inbound.Name, found.Name)
	})

	t.Run("GetByCore", func(t *testing.T) {
		inbounds, err := repo.GetByCore(ctx, core.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(inbounds), 1)
	})

	t.Run("Assign and Unassign User", func(t *testing.T) {
		// Create user
		user := &models.User{
			Username:          "inbounduser",
			Email:             "inbound@example.com",
			UUID:              "550e8400-e29b-41d4-a716-446655440020",
			Password:          "hashedpassword",
			SubscriptionToken: "inboundtoken",
			IsActive:          true,
		}
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		// Create inbound
		inbound := &models.Inbound{
			Name:       "vmess_inbound",
			Protocol:   "vmess",
			CoreID:     core.ID,
			Port:       444,
			ConfigJSON: `{"port":444,"protocol":"vmess"}`,
			IsEnabled:  true,
		}
		err = repo.Create(ctx, inbound)
		require.NoError(t, err)

		// Assign user
		err = repo.AssignToUser(ctx, user.ID, inbound.ID)
		require.NoError(t, err)

		// Get users for inbound
		users, err := repo.GetUsers(ctx, inbound.ID)
		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, user.ID, users[0].ID)

		// Get inbounds for user
		inbounds, err := repo.GetByUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, inbounds, 1)
		assert.Equal(t, inbound.ID, inbounds[0].ID)

		// Unassign user
		err = repo.UnassignFromUser(ctx, user.ID, inbound.ID)
		require.NoError(t, err)

		users, err = repo.GetUsers(ctx, inbound.ID)
		require.NoError(t, err)
		assert.Len(t, users, 0)
	})
}

func TestGORMOutboundRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGORMOutboundRepository(db)
	coreRepo := NewGORMCoreRepository(db)
	ctx := context.Background()

	// Create a core first
	core := &models.Core{
		Name:      "xray",
		Version:   "1.8.0",
		IsEnabled: true,
		APIPort:   10085,
	}
	err := coreRepo.Create(ctx, core)
	require.NoError(t, err)

	t.Run("Create and Get", func(t *testing.T) {
		outbound := &models.Outbound{
			Name:       "direct_outbound",
			Protocol:   "freedom",
			CoreID:     core.ID,
			ConfigJSON: `{"protocol":"freedom"}`,
			Priority:   1,
			IsEnabled:  true,
		}

		err := repo.Create(ctx, outbound)
		require.NoError(t, err)
		assert.NotZero(t, outbound.ID)

		found, err := repo.GetByID(ctx, outbound.ID)
		require.NoError(t, err)
		assert.Equal(t, outbound.Name, found.Name)
	})

	t.Run("GetByCore", func(t *testing.T) {
		outbounds, err := repo.GetByCore(ctx, core.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(outbounds), 1)
	})
}

func TestUnitOfWork(t *testing.T) {
	db := setupTestDB(t)
	factory := NewGORMUnitOfWorkFactory(db)
	ctx := context.Background()

	t.Run("Successful Transaction", func(t *testing.T) {
		err := factory.WithTransaction(ctx, func(uow UnitOfWork) error {
			user := &models.User{
				Username:          "txuser",
				Email:             "tx@example.com",
				UUID:              "550e8400-e29b-41d4-a716-446655440030",
				Password:          "hashedpassword",
				SubscriptionToken: "txtoken",
				IsActive:          true,
			}
			return uow.UserRepository().Create(ctx, user)
		})
		require.NoError(t, err)

		// Verify user was created
		repo := NewGORMUserRepository(db)
		user, err := repo.GetByUUID(ctx, "550e8400-e29b-41d4-a716-446655440030")
		require.NoError(t, err)
		assert.Equal(t, "txuser", user.Username)
	})

	t.Run("Rollback on Error", func(t *testing.T) {
		err := factory.WithTransaction(ctx, func(uow UnitOfWork) error {
			user := &models.User{
				Username:          "rollbackuser",
				Email:             "rollback@example.com",
				UUID:              "550e8400-e29b-41d4-a716-446655440031",
				Password:          "hashedpassword",
				SubscriptionToken: "rollbacktoken",
				IsActive:          true,
			}
			if err := uow.UserRepository().Create(ctx, user); err != nil {
				return err
			}
			// Return an error to trigger rollback
			return ErrInvalidInput
		})
		assert.ErrorIs(t, err, ErrInvalidInput)

		// Verify user was NOT created
		repo := NewGORMUserRepository(db)
		_, err = repo.GetByUUID(ctx, "550e8400-e29b-41d4-a716-446655440031")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestWrapError(t *testing.T) {
	t.Run("Nil error", func(t *testing.T) {
		err := WrapError(nil)
		assert.NoError(t, err)
	})

	t.Run("Record not found", func(t *testing.T) {
		err := WrapError(gorm.ErrRecordNotFound)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Duplicate key", func(t *testing.T) {
		err := WrapError(gorm.ErrDuplicatedKey)
		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("Invalid data", func(t *testing.T) {
		err := WrapError(gorm.ErrInvalidData)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("Invalid transaction", func(t *testing.T) {
		err := WrapError(gorm.ErrInvalidTransaction)
		assert.ErrorIs(t, err, ErrTransaction)
	})

	t.Run("Other errors wrapped as database error", func(t *testing.T) {
		err := WrapError(assert.AnError)
		assert.ErrorIs(t, err, ErrDatabase)
	})
}

func TestErrorHelpers(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		assert.True(t, IsNotFound(ErrNotFound))
		assert.False(t, IsNotFound(ErrAlreadyExists))
	})

	t.Run("IsAlreadyExists", func(t *testing.T) {
		assert.True(t, IsAlreadyExists(ErrAlreadyExists))
		assert.False(t, IsAlreadyExists(ErrNotFound))
	})

	t.Run("IsInvalidInput", func(t *testing.T) {
		assert.True(t, IsInvalidInput(ErrInvalidInput))
		assert.False(t, IsInvalidInput(ErrNotFound))
	})

	t.Run("IsUnauthorized", func(t *testing.T) {
		assert.True(t, IsUnauthorized(ErrUnauthorized))
		assert.False(t, IsUnauthorized(ErrNotFound))
	})
}
