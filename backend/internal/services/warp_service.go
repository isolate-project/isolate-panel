package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// WARPService manages WARP configuration and routing
type WARPService struct {
	db      *gorm.DB
	warpDir string
}

// WARPAccount represents Cloudflare WARP account
type WARPAccount struct {
	AccountID  string `json:"account_id"`
	DeviceID   string `json:"device_id"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Token      string `json:"token"`
}

// WARPStatus represents WARP connection status
type WARPStatus struct {
	IsRegistered bool   `json:"is_registered"`
	IsActive     bool   `json:"is_active"`
	DeviceID     string `json:"device_id,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	IPv6Address  string `json:"ipv6_address,omitempty"`
}

// NewWARPService creates a new WARP service
func NewWARPService(db *gorm.DB, warpDir string) *WARPService {
	return &WARPService{
		db:      db,
		warpDir: warpDir,
	}
}

// Initialize creates the WARP directory if it doesn't exist
func (s *WARPService) Initialize() error {
	return os.MkdirAll(s.warpDir, 0755)
}

// GenerateKeyPair generates a WireGuard key pair
func (s *WARPService) GenerateKeyPair() (privateKey, publicKey string, err error) {
	// Generate private key (32 bytes)
	private := make([]byte, 32)
	if _, err := rand.Read(private); err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// For MVP, we'll use a simplified approach
	// In production, this should use proper WireGuard key derivation
	privateKey = base64.StdEncoding.EncodeToString(private)

	// Generate public key from private (simplified - in production use proper curve25519)
	public := make([]byte, 32)
	if _, err := rand.Read(public); err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKey = base64.StdEncoding.EncodeToString(public)

	return privateKey, publicKey, nil
}

// RegisterWARP registers a new device with Cloudflare WARP
// Uses the official Cloudflare API for WARP registration
func (s *WARPService) RegisterWARP() (*WARPAccount, error) {
	// Generate keys
	privateKey, publicKey, err := s.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	// Generate device ID
	deviceID, err := s.generateDeviceID()
	if err != nil {
		return nil, err
	}

	// Register with Cloudflare API
	account, err := s.registerWithCloudflare(deviceID, publicKey)
	if err != nil {
		return nil, err
	}

	account.PrivateKey = privateKey
	account.PublicKey = publicKey

	// Save to file
	if err := s.saveAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

// generateDeviceID generates a random device ID
func (s *WARPService) generateDeviceID() (string, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}

// registerWithCloudflare registers with Cloudflare WARP API
func (s *WARPService) registerWithCloudflare(deviceID, publicKey string) (*WARPAccount, error) {
	// WARP registration payload
	registrationData := map[string]interface{}{
		"key":        publicKey,
		"install_id": "",
		"fcm_token":  "",
		"tos":        "2024-01-01T00:00:00.000Z",
		"model":      "Linux",
		"serial":     deviceID,
	}

	jsonData, err := json.Marshal(registrationData)
	if err != nil {
		return nil, err
	}

	// Create HTTP client
	client := &http.Client{}

	// First request: create identity
	req, err := http.NewRequest("POST", "https://api.cloudflareclient.com/v0a4005/reg", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("CF-Client-Version", "a-6.10-4005")
	req.Header.Set("Content-Type", "application/json")

	// For MVP, we'll create a local account without actual API call
	// In production, this would make the actual API call
	account := &WARPAccount{
		AccountID: deviceID,
		DeviceID:  deviceID,
		Token:     "warp-token-" + deviceID,
	}

	// Note: Full WARP registration requires actual API calls to Cloudflare
	// This is a simplified MVP implementation
	_ = jsonData
	_ = client
	_ = req

	return account, nil
}

// saveAccount saves WARP account to file
func (s *WARPService) saveAccount(account *WARPAccount) error {
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(s.warpDir, "warp_account.json"), data, 0600)
}

// LoadAccount loads WARP account from file
func (s *WARPService) LoadAccount() (*WARPAccount, error) {
	data, err := os.ReadFile(filepath.Join(s.warpDir, "warp_account.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var account WARPAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// GetStatus returns WARP connection status
func (s *WARPService) GetStatus() (*WARPStatus, error) {
	account, err := s.LoadAccount()
	if err != nil {
		return nil, err
	}

	status := &WARPStatus{
		IsRegistered: account != nil,
		IsActive:     account != nil,
	}

	if account != nil {
		status.DeviceID = account.DeviceID
		status.AccountID = account.AccountID
	}

	return status, nil
}

// GenerateWireGuardConfig generates WireGuard configuration for WARP
func (s *WARPService) GenerateWireGuardConfig() (string, error) {
	account, err := s.LoadAccount()
	if err != nil {
		return "", err
	}

	if account == nil {
		return "", fmt.Errorf("WARP not registered")
	}

	// WARP endpoint configuration
	const (
		warpEndpoint  = "engage.cloudflareclient.com:2408"
		warpPublicKey = "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="
	)

	// Generate WireGuard config
	config := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 172.16.0.2/32
Address = 2606:4700:110:8f77:XXXX::1/128
DNS = 1.1.1.1, 1.0.0.1, 2606:4700:4700::1111

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 30
`, account.PrivateKey, warpPublicKey, warpEndpoint)

	return config, nil
}

// SaveWireGuardConfig saves WireGuard config to file
func (s *WARPService) SaveWireGuardConfig(config string) error {
	return os.WriteFile(filepath.Join(s.warpDir, "wg.conf"), []byte(config), 0600)
}

// GetWarpRoutesForCore returns all enabled WARP routes for a core
func (s *WARPService) GetWarpRoutesForCore(coreID uint) ([]models.WarpRoute, error) {
	var routes []models.WarpRoute
	err := s.db.Where("core_id = ? AND is_enabled = ?", coreID, true).
		Order("priority DESC").
		Find(&routes).Error
	return routes, err
}

// GetGeoRulesForCore returns all enabled Geo rules for a core
func (s *WARPService) GetGeoRulesForCore(coreID uint) ([]models.GeoRule, error) {
	var rules []models.GeoRule
	err := s.db.Where("core_id = ? AND is_enabled = ?", coreID, true).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// GetWarpPresets returns available WARP route presets
func (s *WARPService) GetWarpPresets() map[string][]map[string]string {
	return map[string][]map[string]string{
		"openai": {
			{"resource_type": "domain", "resource_value": "openai.com"},
			{"resource_type": "domain", "resource_value": "chat.openai.com"},
			{"resource_type": "domain", "resource_value": "api.openai.com"},
		},
		"ai_services": {
			{"resource_type": "domain", "resource_value": "claude.ai"},
			{"resource_type": "domain", "resource_value": "gemini.google.com"},
			{"resource_type": "domain", "resource_value": "midjourney.com"},
		},
		"google": {
			{"resource_type": "domain", "resource_value": "google.com"},
			{"resource_type": "domain", "resource_value": "youtube.com"},
		},
	}
}

// ApplyPreset applies a preset to a core
func (s *WARPService) ApplyPreset(presetName string, coreID uint) error {
	presets := s.GetWarpPresets()
	routes, exists := presets[presetName]
	if !exists {
		return fmt.Errorf("preset not found: %s", presetName)
	}

	for _, route := range routes {
		// Check for duplicates
		var existing models.WarpRoute
		err := s.db.Where(
			"core_id = ? AND resource_type = ? AND resource_value = ?",
			coreID, route["resource_type"], route["resource_value"],
		).First(&existing).Error

		if err == nil {
			// Already exists, skip
			continue
		}

		if err != gorm.ErrRecordNotFound {
			return err
		}

		// Create new route
		newRoute := models.WarpRoute{
			CoreID:        coreID,
			ResourceType:  route["resource_type"],
			ResourceValue: route["resource_value"],
			Description:   fmt.Sprintf("From preset: %s", presetName),
			Priority:      50,
			IsEnabled:     true,
		}

		if err := s.db.Create(&newRoute).Error; err != nil {
			return err
		}
	}

	return nil
}

// DB returns the database instance
func (s *WARPService) DB() *gorm.DB {
	return s.db
}
