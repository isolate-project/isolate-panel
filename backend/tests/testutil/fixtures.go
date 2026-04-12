package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// Fixture paths
const (
	FixtureUsersPath     = "tests/fixtures/users.json"
	FixtureInboundsPath  = "tests/fixtures/inbounds.json"
	FixtureOutboundsPath = "tests/fixtures/outbounds.json"
	FixtureCoresPath     = "tests/fixtures/cores.json"
)

// LoadFixtureUsers loads users from JSON fixture file
func LoadFixtureUsers(t *testing.T, db *gorm.DB, fixturePath string) []models.User {
	t.Helper()

	data, err := loadFixture(fixturePath)
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	var users []models.User
	if err := json.Unmarshal(data, &users); err != nil {
		t.Fatalf("Failed to unmarshal users fixture: %v", err)
	}

	// Create users in database
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create user from fixture: %v", err)
		}
	}

	return users
}

// LoadFixtureInbounds loads inbounds from JSON fixture file
func LoadFixtureInbounds(t *testing.T, db *gorm.DB, fixturePath string) []models.Inbound {
	t.Helper()

	data, err := loadFixture(fixturePath)
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	var inbounds []models.Inbound
	if err := json.Unmarshal(data, &inbounds); err != nil {
		t.Fatalf("Failed to unmarshal inbounds fixture: %v", err)
	}

	// Create inbounds in database
	for _, inbound := range inbounds {
		if err := db.Create(&inbound).Error; err != nil {
			t.Fatalf("Failed to create inbound from fixture: %v", err)
		}
	}

	return inbounds
}

// loadFixture reads a fixture file and returns its contents
func loadFixture(fixturePath string) ([]byte, error) {
	// Try relative to backend directory first
	absPath := fixturePath
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// Try relative to tests directory
		absPath = filepath.Join("backend", fixturePath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			// Try from tests directory
			absPath = filepath.Join("../", fixturePath)
		}
	}

	return os.ReadFile(absPath)
}

// CreateUsersFixture creates a JSON fixture file for users
func CreateUsersFixture(t *testing.T, outputPath string, users []models.User) {
	t.Helper()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal users: %v", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		t.Fatalf("Failed to write fixture file: %v", err)
	}
}

// SampleUsers returns sample users for testing
func SampleUsers() []models.User {
	trafficLimit1 := int64(107374182400) // 100GB
	trafficLimit2 := int64(53687091200)  // 50GB

	return []models.User{
		{
			UUID:              "sample-uuid-1",
			Username:          "alice",
			Email:             "alice@example.com",
			SubscriptionToken: "alice-token-000000000000000000000",
			IsActive:          true,
			TrafficLimitBytes: &trafficLimit1,
			TrafficUsedBytes:  0,
		},
		{
			UUID:              "sample-uuid-2",
			Username:          "bob",
			Email:             "bob@example.com",
			SubscriptionToken: "bob-token-00000000000000000000000",
			IsActive:          true,
			TrafficLimitBytes: &trafficLimit2,
			TrafficUsedBytes:  1073741824, // 1GB
		},
		{
			UUID:              "sample-uuid-3",
			Username:          "charlie",
			Email:             "charlie@example.com",
			SubscriptionToken: "charlie-token-0000000000000000000",
			IsActive:          false,
			TrafficLimitBytes: nil, // Unlimited
			TrafficUsedBytes:  0,
		},
	}
}

// SampleInbounds returns sample inbounds for testing
func SampleInbounds() []models.Inbound {
	return []models.Inbound{
		{
			Name:          "VMess-443",
			Protocol:      "vmess",
			CoreID:        1,
			ListenAddress: "0.0.0.0",
			Port:          443,
			ConfigJSON:    `{"clients":[]}`,
			IsEnabled:     true,
		},
		{
			Name:          "VLESS-8443",
			Protocol:      "vless",
			CoreID:        1,
			ListenAddress: "0.0.0.0",
			Port:          8443,
			ConfigJSON:    `{"clients":[]}`,
			IsEnabled:     true,
		},
		{
			Name:          "Trojan-9000",
			Protocol:      "trojan",
			CoreID:        2,
			ListenAddress: "0.0.0.0",
			Port:          9000,
			ConfigJSON:    `{"passwords":[]}`,
			IsEnabled:     true,
		},
	}
}

// SampleOutbounds returns sample outbounds for testing
func SampleOutbounds() []models.Outbound {
	return []models.Outbound{
		{
			Name:       "Direct",
			Protocol:   "freedom",
			CoreID:     1,
			ConfigJSON: `{"tag":"direct"}`,
			Priority:   0,
			IsEnabled:  true,
		},
		{
			Name:       "Block",
			Protocol:   "blackhole",
			CoreID:     1,
			ConfigJSON: `{"tag":"block"}`,
			Priority:   999,
			IsEnabled:  true,
		},
	}
}
