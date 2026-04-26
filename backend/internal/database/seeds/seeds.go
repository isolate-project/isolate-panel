package seeds

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/isolate-project/isolate-panel/internal/auth"
	"gorm.io/gorm"
)

type Seeder struct {
	db *gorm.DB
}

type Admin struct {
	ID                uint   `gorm:"primaryKey"`
	Username          string `gorm:"uniqueIndex;not null"`
	PasswordHash      string `gorm:"not null"`
	IsSuperAdmin      bool   `gorm:"default:false"`
	MustChangePassword bool   `gorm:"default:false"`
}

type Setting struct {
	ID          uint   `gorm:"primaryKey"`
	Key         string `gorm:"uniqueIndex;not null"`
	Value       string
	ValueType   string `gorm:"default:'string'"`
	Description string
}

type User struct {
	ID                uint   `gorm:"primaryKey"`
	Username          string `gorm:"uniqueIndex;not null"`
	Email             string
	UUID              string `gorm:"uniqueIndex;not null"`
	Password          string `gorm:"not null"`
	Token             string `gorm:"uniqueIndex"`
	SubscriptionToken string `gorm:"uniqueIndex;not null"`
	TrafficLimitBytes *int64
	TrafficUsedBytes  int64 `gorm:"default:0"`
	IsActive          bool  `gorm:"default:true"`
}

type Core struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"uniqueIndex;not null"`
	Version    string `gorm:"not null"`
	IsEnabled  bool   `gorm:"default:true"`
	IsRunning  bool   `gorm:"default:false"`
	ConfigPath string
	LogPath    string
}

func NewSeeder(db *gorm.DB) *Seeder {
	return &Seeder{db: db}
}

// RunAll runs all seeders
func (s *Seeder) RunAll(adminPassword string) error {
	if err := s.SeedDefaultAdmin(adminPassword); err != nil {
		return err
	}
	if err := s.SeedDefaultSettings(); err != nil {
		return err
	}
	if err := s.SeedCores(); err != nil {
		return err
	}
	
	// Note: SeedDevelopmentUsers is disabled for safety in automated runs.
	// To be implemented via a dedicated CLI command (e.g. isolate-migrate seed-dev) in the future if needed.
	
	return nil
}

// SeedDefaultAdmin creates default admin user
func (s *Seeder) SeedDefaultAdmin(adminPassword string) error {
	var count int64
	s.db.Model(&Admin{}).Count(&count)

	if count > 0 {
		return nil // Admin already exists
	}

	if adminPassword == "" {
		adminPassword = "admin"
	}

	passwordHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	admin := &Admin{
		Username:           "admin",
		PasswordHash:       passwordHash,
		IsSuperAdmin:       true,
		MustChangePassword: true, // Always require first-login password change
	}

	if err := s.db.Table("admins").Create(admin).Error; err != nil {
		return fmt.Errorf("failed to seed default admin: %w", err)
	}

	fmt.Println("✓ Default admin created")
	fmt.Println("  IMPORTANT: You must change the default password on first login")
	return nil
}

// SeedDefaultSettings creates default system settings
func (s *Seeder) SeedDefaultSettings() error {
	defaultSettings := []Setting{
		{Key: "panel_name", Value: "Isolate Panel", ValueType: "string", Description: "Panel display name"},
		{Key: "traffic_collection_interval", Value: "60", ValueType: "int", Description: "Traffic collection interval in seconds"},
		{Key: "data_retention_days", Value: "90", ValueType: "int", Description: "Data retention period in days"},
		{Key: "max_login_attempts", Value: "5", ValueType: "int", Description: "Maximum login attempts before blocking"},
		{Key: "jwt_access_token_ttl", Value: "900", ValueType: "int", Description: "JWT access token TTL in seconds (15 minutes)"},
		{Key: "jwt_refresh_token_ttl", Value: "604800", ValueType: "int", Description: "JWT refresh token TTL in seconds (7 days)"},
		{Key: "monitoring_mode", Value: "lite", ValueType: "string", Description: "Monitoring mode: lite or full"},
		{Key: "backup_enabled", Value: "false", ValueType: "bool", Description: "Automatic backups enabled"},
		{Key: "backup_retention_count", Value: "3", ValueType: "int", Description: "Number of backups to keep"},
		{Key: "warp_enabled", Value: "false", ValueType: "bool", Description: "WARP integration enabled"},
	}

	for _, setting := range defaultSettings {
		var existing Setting
		err := s.db.Table("settings").Where("key = ?", setting.Key).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := s.db.Table("settings").Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to seed setting %s: %w", setting.Key, err)
			}
		}
	}

	fmt.Println("✓ Default settings seeded")
	return nil
}

// SeedCores creates default core entries
func (s *Seeder) SeedCores() error {
	cores := []Core{
		{
			Name:       "singbox",
			Version:    "1.13.8",
			IsEnabled:  true,
			IsRunning:  false,
			ConfigPath: "/app/data/cores/singbox/config.json",
			LogPath:    "/var/log/isolate-panel/singbox.log",
		},
		{
			Name:       "xray",
			Version:    "26.3.27",
			IsEnabled:  true,
			IsRunning:  false,
			ConfigPath: "/app/data/cores/xray/config.json",
			LogPath:    "/var/log/isolate-panel/xray.log",
		},
		{
			Name:       "mihomo",
			Version:    "1.19.23",
			IsEnabled:  true,
			IsRunning:  false,
			ConfigPath: "/app/data/cores/mihomo/config.yaml",
			LogPath:    "/var/log/isolate-panel/mihomo.log",
		},
	}

	for _, core := range cores {
		var existing Core
		err := s.db.Table("cores").Where("name = ?", core.Name).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := s.db.Table("cores").Create(&core).Error; err != nil {
				return fmt.Errorf("failed to seed core %s: %w", core.Name, err)
			}
		}
	}

	fmt.Println("✓ Cores seeded (singbox, xray, mihomo)")
	return nil
}

// SeedDevelopmentUsers creates test users (only in development)
func (s *Seeder) SeedDevelopmentUsers() error {
	// Only run in development mode
	if os.Getenv("APP_ENV") != "development" {
		return nil
	}

	testUsers := []User{
		{
			UUID:              "550e8400-e29b-41d4-a716-446655440001",
			Username:          "testuser1",
			Email:             "test1@example.com",
			Password:          "testpass123",
			Token:             generateRandomToken(32),
			SubscriptionToken: generateRandomToken(32),
			IsActive:          true,
			TrafficLimitBytes: int64Ptr(107374182400), // 100GB
			TrafficUsedBytes:  0,
		},
		{
			UUID:              "550e8400-e29b-41d4-a716-446655440002",
			Username:          "testuser2",
			Email:             "test2@example.com",
			Password:          "testpass456",
			Token:             generateRandomToken(32),
			SubscriptionToken: generateRandomToken(32),
			IsActive:          true,
			TrafficLimitBytes: int64Ptr(53687091200), // 50GB
			TrafficUsedBytes:  10737418240,           // 10GB used
		},
	}

	for _, user := range testUsers {
		var existing User
		err := s.db.Table("users").Where("username = ?", user.Username).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := s.db.Table("users").Create(&user).Error; err != nil {
				return fmt.Errorf("failed to seed user %s: %w", user.Username, err)
			}
		}
	}

	fmt.Println("✓ Development users seeded")
	return nil
}

// generateRandomToken generates a random token
func generateRandomToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// int64Ptr returns a pointer to an int64
func int64Ptr(i int64) *int64 {
	return &i
}

// EncryptExistingPasswords encrypts all plaintext user passwords
// This should be called after migration 000040_encrypt_user_passwords
func (s *Seeder) EncryptExistingPasswords() error {
	type User struct {
		ID                uint
		Password          string
		PasswordEncrypted bool
	}

	var users []User
	if err := s.db.Table("users").Where("password_encrypted = ?", false).Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("✓ All passwords already encrypted")
		return nil
	}

	fmt.Printf("Encrypting %d user passwords...\n", len(users))

	for _, user := range users {
		encryptedPassword, err := auth.EncryptCredential(user.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password for user %d: %w", user.ID, err)
		}

		if err := s.db.Table("users").
			Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"password":            encryptedPassword,
				"password_encrypted":  true,
			}).Error; err != nil {
			return fmt.Errorf("failed to update user %d: %w", user.ID, err)
		}
	}

	fmt.Printf("✓ Encrypted %d user passwords\n", len(users))
	return nil
}
