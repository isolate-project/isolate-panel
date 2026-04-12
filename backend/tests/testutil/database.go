package testutil

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// SetupTestDB creates a true in-memory SQLite database for testing
// Each call creates a new isolated in-memory database
func SetupTestDB(t testing.TB) *gorm.DB {
	t.Helper()

	// Use true in-memory database (no file created)
	// Without cache=shared, each connection gets its own isolated DB
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Get underlying sql.DB and set connection pool
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Run auto migrations for tests
	if err := runAutoMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// TeardownTestDB closes the test database
func TeardownTestDB(t testing.TB, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Errorf("Failed to get sql.DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
}

func runAutoMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Admin{},
		&models.User{},
		&models.Core{},
		&models.Inbound{},
		&models.Outbound{},
		&models.UserInboundMapping{},
		&models.Setting{},
		&models.TrafficStats{},
		&models.ActiveConnection{},
		&models.Notification{},
		&models.NotificationSettings{},
		&models.Backup{},
		&models.Certificate{},
		&models.GeoRule{},
		&models.WarpRoute{},
		&models.SubscriptionAccess{},
		&models.SubscriptionShortURL{},
	)
}

// SeedTestData seeds the database with test data
func SeedTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Create test admin
	admin := &models.Admin{
		Username:     "testadmin",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		IsSuperAdmin: true,
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("Failed to seed admin: %v", err)
	}

	// Create test users
	trafficLimit1 := int64(107374182400) // 100GB
	trafficLimit2 := int64(53687091200)  // 50GB
	trafficUsed2 := int64(10737418240)   // 10GB

	users := []models.User{
		{
			UUID:              "test-uuid-1",
			Username:          "testuser1",
			Email:             "test1@example.com",
			SubscriptionToken: "token1-00000000000000000000000000",
			IsActive:          true,
			TrafficLimitBytes: &trafficLimit1,
			TrafficUsedBytes:  0,
		},
		{
			UUID:              "test-uuid-2",
			Username:          "testuser2",
			Email:             "test2@example.com",
			SubscriptionToken: "token2-00000000000000000000000000",
			IsActive:          false,
			TrafficLimitBytes: &trafficLimit2,
			TrafficUsedBytes:  trafficUsed2,
		},
	}

	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("Failed to seed user: %v", err)
		}
	}

	// Create test cores
	cores := []models.Core{
		{
			Name:      "singbox",
			Version:   "1.13.3",
			IsEnabled: true,
			IsRunning: false,
		},
		{
			Name:      "xray",
			Version:   "26.2.6",
			IsEnabled: true,
			IsRunning: false,
		},
		{
			Name:      "mihomo",
			Version:   "1.19.21",
			IsEnabled: true,
			IsRunning: false,
		},
	}

	for _, core := range cores {
		if err := db.Create(&core).Error; err != nil {
			t.Fatalf("Failed to seed core: %v", err)
		}
	}

	// Seed default settings
	settings := []models.Setting{
		{Key: "monitoring_mode", Value: "lite", ValueType: "string", Description: "Monitoring mode: lite or full"},
		{Key: "backup_enabled", Value: "false", ValueType: "bool", Description: "Automatic backups enabled"},
		{Key: "backup_schedule", Value: "0 2 * * *", ValueType: "string", Description: "Backup schedule (cron)"},
		{Key: "backup_retention", Value: "7", ValueType: "int", Description: "Number of backups to retain"},
	}

	for _, setting := range settings {
		if err := db.Create(&setting).Error; err != nil {
			t.Fatalf("Failed to seed setting: %v", err)
		}
	}
}

// CreateTestUser creates a test user and returns it
func CreateTestUser(t testing.TB, db *gorm.DB, username, email string) *models.User {
	t.Helper()

	trafficLimit := int64(107374182400)
	user := &models.User{
		UUID:              fmt.Sprintf("test-uuid-%s", username),
		Username:          username,
		Email:             email,
		SubscriptionToken: fmt.Sprintf("token-%s-00000000000000000000000000", username),
		IsActive:          true,
		TrafficLimitBytes: &trafficLimit,
		TrafficUsedBytes:  0,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CreateTestInbound creates a test inbound and returns it
func CreateTestInbound(t testing.TB, db *gorm.DB, name string, coreID uint) *models.Inbound {
	t.Helper()

	inbound := &models.Inbound{
		Name:          name,
		Protocol:      "vmess",
		CoreID:        coreID,
		ListenAddress: "0.0.0.0",
		Port:          10000,
		ConfigJSON:    `{"clients":[]}`,
		IsEnabled:     true,
	}

	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("Failed to create test inbound: %v", err)
	}

	return inbound
}

// CreateTestOutbound creates a test outbound and returns it
func CreateTestOutbound(t *testing.T, db *gorm.DB, name string, coreID uint) *models.Outbound {
	t.Helper()

	outbound := &models.Outbound{
		Name:       name,
		Protocol:   "direct",
		CoreID:     coreID,
		ConfigJSON: `{"tag":"direct"}`,
		Priority:   0,
		IsEnabled:  true,
	}

	if err := db.Create(outbound).Error; err != nil {
		t.Fatalf("Failed to create test outbound: %v", err)
	}

	return outbound
}

// GetTestAdmin retrieves the test admin
func GetTestAdmin(t *testing.T, db *gorm.DB) *models.Admin {
	t.Helper()

	var admin models.Admin
	if err := db.Where("username = ?", "testadmin").First(&admin).Error; err != nil {
		t.Fatalf("Failed to get test admin: %v", err)
	}

	return &admin
}

// GetTestCore retrieves a test core by name
func GetTestCore(t *testing.T, db *gorm.DB, name string) *models.Core {
	t.Helper()

	var core models.Core
	if err := db.Where("name = ?", name).First(&core).Error; err != nil {
		t.Fatalf("Failed to get test core: %v", err)
	}

	return &core
}

// TruncateAll truncates all tables (useful for test cleanup)
func TruncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()

	tables := []string{
		"user_inbound_mapping",
		"traffic_stats",
		"notifications",
		"backups",
		"certificates",
		"geo_rules",
		"warp_routes",
		"outbounds",
		"inbounds",
		"cores",
		"users",
		"admins",
		"settings",
	}

	// Disable foreign key checks temporarily
	db.Exec("PRAGMA foreign_keys = OFF")

	for _, table := range tables {
		db.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}

	// Enable foreign key checks
	db.Exec("PRAGMA foreign_keys = ON")
}
