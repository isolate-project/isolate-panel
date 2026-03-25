package benchmarks_test

import (
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

// BenchmarkUserService_CreateUser benchmarks user creation performance
func BenchmarkUserService_CreateUser(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		req := &services.CreateUserRequest{
			Username: "benchuser_" + string(rune(i)),
			Email:    "bench" + string(rune(i)) + "@example.com",
		}
		b.StartTimer()

		_, err := userService.CreateUser(req, 1)
		if err != nil {
			b.Fatalf("Failed to create user: %v", err)
		}
	}
}

// BenchmarkUserService_GetUser benchmarks user retrieval by ID
func BenchmarkUserService_GetUser(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	// Create test user
	testutil.CreateTestUser(b, db, "benchuser", "bench@example.com")

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := userService.GetUser(1)
		if err != nil {
			b.Fatalf("Failed to get user: %v", err)
		}
	}
}

// BenchmarkInboundService_ListInbounds benchmarks inbound listing
func BenchmarkInboundService_ListInbounds(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	// Create test core
	db.Create(&models.Core{
		Name:      "singbox",
		Version:   "1.13.3",
		IsEnabled: true,
		IsRunning: false,
	})

	// Create test inbounds
	for i := 0; i < 10; i++ {
		db.Create(&models.Inbound{
			Name:       "test_inbound_" + string(rune(i)),
			Protocol:   "vless",
			CoreID:     1,
			Port:       10000 + i,
			ConfigJSON: `{"tag": "test"}`,
		})
	}

	coreManager := &services.CoreLifecycleManager{}
	inboundService := services.NewInboundService(db, coreManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := inboundService.ListInbounds(nil, nil)
		if err != nil {
			b.Fatalf("Failed to list inbounds: %v", err)
		}
	}
}

// BenchmarkSubscriptionService_GenerateV2Ray benchmarks V2Ray subscription generation
func BenchmarkSubscriptionService_GenerateV2Ray(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	// Create test user and inbounds
	user := testutil.CreateTestUser(b, db, "bench_sub", "bench@example.com")

	db.Create(&models.Core{
		Name:      "singbox",
		Version:   "1.13.3",
		IsEnabled: true,
	})

	db.Create(&models.Inbound{
		Name:       "test_inbound_sub",
		Protocol:   "vless",
		CoreID:     1,
		Port:       443,
		ConfigJSON: `{"tag": "test"}`,
	})

	db.Create(&models.UserInboundMapping{
		UserID:    user.ID,
		InboundID: 1,
	})

	subService := services.NewSubscriptionService(db, "http://localhost:8080")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := subService.GetUserSubscriptionData(user.SubscriptionToken)
		if err != nil {
			b.Fatalf("Failed to get subscription data: %v", err)
		}

		_, err = subService.GenerateV2Ray(data)
		if err != nil {
			b.Fatalf("Failed to generate V2Ray: %v", err)
		}
	}
}

// BenchmarkSettingsService_Get benchmarks settings retrieval
func BenchmarkSettingsService_Get(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	// Create test settings
	db.Create(&models.Setting{
		Key:         "test_setting",
		Value:       "test_value",
		ValueType:   "string",
		Description: "Test setting",
	})

	settingsService := services.NewSettingsService(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := settingsService.GetSettingValue("test_setting")
		if err != nil {
			b.Fatalf("Failed to get setting: %v", err)
		}
	}
}

// BenchmarkDatabase_Query benchmarks raw database query performance
func BenchmarkDatabase_Query(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	// Create test data
	for i := 0; i < 100; i++ {
		testutil.CreateTestUser(b, db, "user_"+string(rune(i)), "user"+string(rune(i))+"@example.com")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var users []models.User
		err := db.Where("is_active = ?", true).Find(&users).Error
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// BenchmarkDatabase_Insert benchmarks database insert performance
func BenchmarkDatabase_Insert(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &models.User{
			Username: "insert_user_" + string(rune(i)),
			Email:    "insert" + string(rune(i)) + "@example.com",
		}
		err := db.Create(user).Error
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

// BenchmarkDatabase_Update benchmarks database update performance
func BenchmarkDatabase_Update(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	// Create test user
	user := testutil.CreateTestUser(b, db, "update_user", "update@example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.Model(user).Update("email", "updated_"+string(rune(i))+"@example.com").Error
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

// BenchmarkDatabase_Delete benchmarks database delete performance
func BenchmarkDatabase_Delete(b *testing.B) {
	db := testutil.SetupTestDB(b)
	defer testutil.TeardownTestDB(b, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		user := &models.User{
			Username: "delete_user_" + string(rune(i)),
			Email:    "delete" + string(rune(i)) + "@example.com",
		}
		db.Create(user)
		b.StartTimer()

		err := db.Delete(user).Error
		if err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}
}
