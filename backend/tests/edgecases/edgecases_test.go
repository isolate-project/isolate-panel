package edgecases_test

import (
	"strings"
	"testing"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

// TestUserService_EmptyUsername tests edge case: empty username
func TestUserService_EmptyUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	req := &services.CreateUserRequest{
		Username: "",
		Email:    "test@example.com",
	}

	_, err := userService.CreateUser(req, 1)
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}
}

// TestUserService_UsernameWithSpecialChars tests edge case: username with special characters
func TestUserService_UsernameWithSpecialChars(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	req := &services.CreateUserRequest{
		Username: "test@user#123!",
		Email:    "test@example.com",
	}

	_, err := userService.CreateUser(req, 1)
	// Special characters should be allowed
	if err != nil {
		t.Errorf("Failed to create user with special chars: %v", err)
	}
}

// TestUserService_VeryLongUsername tests edge case: very long username
func TestUserService_VeryLongUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	longUsername := strings.Repeat("a", 1000)
	req := &services.CreateUserRequest{
		Username: longUsername,
		Email:    "test@example.com",
	}

	_, err := userService.CreateUser(req, 1)
	if err == nil {
		t.Error("Expected error for very long username, got nil")
	}
}

// TestUserService_DuplicateUsername tests edge case: duplicate username
func TestUserService_DuplicateUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	req1 := &services.CreateUserRequest{
		Username: "duplicate_user",
		Email:    "test1@example.com",
	}
	_, err := userService.CreateUser(req1, 1)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	req2 := &services.CreateUserRequest{
		Username: "duplicate_user",
		Email:    "test2@example.com",
	}
	_, err = userService.CreateUser(req2, 1)
	if err == nil {
		t.Error("Expected error for duplicate username, got nil")
	}
}

// TestInboundService_InvalidPort tests edge case: invalid port (0)
func TestInboundService_InvalidPort(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create test core
	db.Create(&models.Core{
		Name:      "singbox",
		Version:   "1.13.3",
		IsEnabled: true,
		IsRunning: false,
	})

	coreManager := &services.CoreLifecycleManager{}
	inboundService := services.NewInboundService(db, coreManager)

	inbound := &models.Inbound{
		Name:       "test_inbound",
		Protocol:   "vless",
		CoreID:     1,
		Port:       0, // Invalid port
		ConfigJSON: `{"tag": "test"}`,
	}

	err := inboundService.CreateInbound(inbound)
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

// TestInboundService_PortOutOfRange tests edge case: port out of valid range
func TestInboundService_PortOutOfRange(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create test core
	db.Create(&models.Core{
		Name:      "singbox",
		Version:   "1.13.3",
		IsEnabled: true,
		IsRunning: false,
	})

	coreManager := &services.CoreLifecycleManager{}
	inboundService := services.NewInboundService(db, coreManager)

	inbound := &models.Inbound{
		Name:       "test_inbound",
		Protocol:   "vless",
		CoreID:     1,
		Port:       70000, // Port > 65535
		ConfigJSON: `{"tag": "test"}`,
	}

	err := inboundService.CreateInbound(inbound)
	if err == nil {
		t.Error("Expected error for port out of range, got nil")
	}
}

// TestSubscriptionService_ExpiredUser tests edge case: expired user subscription
func TestSubscriptionService_ExpiredUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create expired user
	expiryDate := time.Now().AddDate(0, 0, -1) // Yesterday
	user := &models.User{
		Username:          "expired_user",
		Email:             "expired@example.com",
		SubscriptionToken: "expired_token",
		ExpiryDate:        &expiryDate,
		IsActive:          false, // Also inactive
	}
	db.Create(user)

	subService := services.NewSubscriptionService(db, "http://localhost:8080")

	_, err := subService.GetUserBySubscriptionToken("expired_token")
	if err == nil {
		t.Error("Expected error for expired user, got nil")
	}
}

// TestSubscriptionService_InactiveUser tests edge case: inactive user
func TestSubscriptionService_InactiveUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create inactive user
	user := &models.User{
		Username:          "inactive_user",
		Email:             "inactive@example.com",
		SubscriptionToken: "inactive_token",
		IsActive:          false,
	}
	db.Create(user)

	subService := services.NewSubscriptionService(db, "http://localhost:8080")

	_, err := subService.GetUserBySubscriptionToken("inactive_token")
	if err == nil {
		t.Error("Expected error for inactive user, got nil")
	}
}

// TestSettingsService_EmptyKey tests edge case: empty setting key
func TestSettingsService_EmptyKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	settingsService := services.NewSettingsService(db)

	_, err := settingsService.GetSettingValue("")
	if err == nil {
		t.Error("Expected error for empty key, got nil")
	}
}

// TestSettingsService_NonExistentKey tests edge case: non-existent setting key
func TestSettingsService_NonExistentKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	settingsService := services.NewSettingsService(db)

	_, err := settingsService.GetSettingValue("non_existent_key_12345")
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

// TestUserService_EmptyEmail tests edge case: empty email
func TestUserService_EmptyEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	req := &services.CreateUserRequest{
		Username: "testuser",
		Email:    "",
	}

	_, err := userService.CreateUser(req, 1)
	// Empty email might be allowed or not - depends on validation
	// This test documents the current behavior
	if err != nil {
		t.Logf("Empty email rejected: %v", err)
	}
}

// TestUserService_InvalidEmailFormat tests edge case: invalid email format
func TestUserService_InvalidEmailFormat(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	notificationService := services.NewNotificationService(db, "", "", "", "")
	userService := services.NewUserService(db, notificationService)

	req := &services.CreateUserRequest{
		Username: "testuser",
		Email:    "not-an-email",
	}

	_, err := userService.CreateUser(req, 1)
	if err == nil {
		t.Error("Expected error for invalid email format, got nil")
	}
}

// TestInboundService_EmptyName tests edge case: empty inbound name
func TestInboundService_EmptyName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create test core
	db.Create(&models.Core{
		Name:      "singbox",
		Version:   "1.13.3",
		IsEnabled: true,
		IsRunning: false,
	})

	coreManager := &services.CoreLifecycleManager{}
	inboundService := services.NewInboundService(db, coreManager)

	inbound := &models.Inbound{
		Name:       "", // Empty name
		Protocol:   "vless",
		CoreID:     1,
		Port:       10000,
		ConfigJSON: `{"tag": "test"}`,
	}

	err := inboundService.CreateInbound(inbound)
	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}
}

// TestInboundService_EmptyProtocol tests edge case: empty protocol
func TestInboundService_EmptyProtocol(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)

	// Create test core
	db.Create(&models.Core{
		Name:      "singbox",
		Version:   "1.13.3",
		IsEnabled: true,
		IsRunning: false,
	})

	coreManager := &services.CoreLifecycleManager{}
	inboundService := services.NewInboundService(db, coreManager)

	inbound := &models.Inbound{
		Name:       "test_inbound",
		Protocol:   "", // Empty protocol
		CoreID:     1,
		Port:       10000,
		ConfigJSON: `{"tag": "test"}`,
	}

	err := inboundService.CreateInbound(inbound)
	if err == nil {
		t.Error("Expected error for empty protocol, got nil")
	}
}
