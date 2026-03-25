package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		request *services.CreateUserRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user",
			request: &services.CreateUserRequest{
				Username:          "newuser",
				Email:             "new@example.com",
				Password:          "password123",
				TrafficLimitBytes: nil,
			},
			wantErr: false,
		},
		{
			name: "duplicate username",
			request: &services.CreateUserRequest{
				Username:          "testuser1", // Already exists in seed data
				Email:             "another@example.com",
				Password:          "password123",
				TrafficLimitBytes: nil,
			},
			wantErr: true,
			errMsg:  "username already exists",
		},
		{
			name: "invalid email",
			request: &services.CreateUserRequest{
				Username:          "user3",
				Email:             "invalid-email",
				Password:          "password123",
				TrafficLimitBytes: nil,
			},
			wantErr: true,
			errMsg:  "validation",
		},
		{
			name: "short username",
			request: &services.CreateUserRequest{
				Username:          "ab",
				Email:             "short@example.com",
				Password:          "password123",
				TrafficLimitBytes: nil,
			},
			wantErr: true,
			errMsg:  "validation",
		},
		{
			name: "short password",
			request: &services.CreateUserRequest{
				Username:          "user4",
				Email:             "shortpwd@example.com",
				Password:          "short",
				TrafficLimitBytes: nil,
			},
			wantErr: true,
			errMsg:  "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := testutil.SetupTestDB(t)
			defer testutil.TeardownTestDB(t, db)
			testutil.SeedTestData(t, db)

			notificationService := services.NewNotificationService(nil, "", "", "", "")
			service := services.NewUserService(db, notificationService)

			// Execute
			user, err := service.CreateUser(tt.request, 1)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Username, user.Username)
				assert.Equal(t, tt.request.Email, user.Email)
				assert.NotEmpty(t, user.UUID)
				assert.NotEmpty(t, user.SubscriptionToken)
				assert.True(t, user.IsActive)
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(nil, "", "", "", "")
	service := services.NewUserService(db, notificationService)

	t.Run("get existing user", func(t *testing.T) {
		user, err := service.GetUser(1)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser1", user.Username)
	})

	t.Run("get non-existing user", func(t *testing.T) {
		user, err := service.GetUser(999)
		require.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(nil, "", "", "", "")
	service := services.NewUserService(db, notificationService)

	t.Run("update username", func(t *testing.T) {
		newUsername := "updateduser"
		req := &services.UpdateUserRequest{
			Username: &newUsername,
		}

		user, err := service.UpdateUser(1, req)
		require.NoError(t, err)
		assert.Equal(t, newUsername, user.Username)
	})

	t.Run("update traffic limit", func(t *testing.T) {
		newLimit := int64(214748364800) // 200GB
		req := &services.UpdateUserRequest{
			TrafficLimitBytes: &newLimit,
		}

		user, err := service.UpdateUser(1, req)
		require.NoError(t, err)
		assert.NotNil(t, user.TrafficLimitBytes)
		assert.Equal(t, newLimit, *user.TrafficLimitBytes)
	})

	t.Run("update non-existing user", func(t *testing.T) {
		newUsername := "updateduser"
		req := &services.UpdateUserRequest{
			Username: &newUsername,
		}

		user, err := service.UpdateUser(999, req)
		require.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(nil, "", "", "", "")
	service := services.NewUserService(db, notificationService)

	t.Run("delete existing user", func(t *testing.T) {
		err := service.DeleteUser(1)
		require.NoError(t, err)

		// Verify user is deleted
		user, err := service.GetUser(1)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("delete non-existing user", func(t *testing.T) {
		err := service.DeleteUser(999)
		require.Error(t, err)
	})
}

func TestUserService_CheckExpiringUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(nil, "", "", "", "")
	service := services.NewUserService(db, notificationService)

	// This should not panic
	service.CheckExpiringUsers()
}

func TestUserService_RegenerateCredentials(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	notificationService := services.NewNotificationService(nil, "", "", "", "")
	service := services.NewUserService(db, notificationService)

	t.Run("regenerate existing user", func(t *testing.T) {
		user, err := service.RegenerateCredentials(1)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotEmpty(t, user.SubscriptionToken)
	})

	t.Run("regenerate non-existing user", func(t *testing.T) {
		user, err := service.RegenerateCredentials(999)
		require.Error(t, err)
		assert.Nil(t, user)
	})
}
