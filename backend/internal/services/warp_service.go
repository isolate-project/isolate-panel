package services

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// WARPService manages WARP configuration and routing
type WARPService struct {
	db       *gorm.DB
	warpDir  string
	stopCh   chan struct{}
	stopOnce sync.Once
}

// WARPAccount represents Cloudflare WARP account
type WARPAccount struct {
	AccountID   string `json:"account_id"`
	DeviceID    string `json:"device_id"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	Token       string `json:"token"`
	IPv4Address string `json:"ipv4_address,omitempty"`
	IPv6Address string `json:"ipv6_address,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
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

// GenerateKeyPair generates a proper WireGuard key pair using curve25519
func (s *WARPService) GenerateKeyPair() (privateKey, publicKey string, err error) {
	// Generate private key using wireguard-go library
	private, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Get public key from private key
	public := private.PublicKey()

	// Encode to base64
	privateKey = private.String()
	publicKey = public.String()

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
// Reference: https://github.com/cloudflare/warp-client
func (s *WARPService) registerWithCloudflare(deviceID, publicKey string) (*WARPAccount, error) {
	// WARP registration payload
	registrationData := map[string]interface{}{
		"key":          publicKey,
		"install_id":   "",
		"fcm_token":    "",
		"tos":          time.Now().Format(time.RFC3339),
		"model":        "Linux",
		"serial":       deviceID,
		"locale":       "en_US",
		"warp_enabled": true,
	}

	jsonData, err := json.Marshal(registrationData)
	if err != nil {
		return nil, err
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("POST", "https://api.cloudflareclient.com/v0a4005/reg", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("CF-Client-Version", "a-6.10-4005")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "okhttp/3.12.1")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to register with Cloudflare: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Cloudflare API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var apiResp map[string]interface{}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	// Extract account ID and token
	accountID, ok := apiResp["account_id"].(string)
	if !ok {
		return nil, fmt.Errorf("account_id not found in response")
	}

	token, ok := apiResp["token"].(string)
	if !ok {
		return nil, fmt.Errorf("token not found in response")
	}

	account := &WARPAccount{
		AccountID: accountID,
		DeviceID:  deviceID,
		Token:     token,
	}

	// Extract IP addresses from config.interface.addresses
	if config, ok := apiResp["config"].(map[string]interface{}); ok {
		if iface, ok := config["interface"].(map[string]interface{}); ok {
			if addrs, ok := iface["addresses"].(map[string]interface{}); ok {
				if v4, ok := addrs["v4"].(string); ok {
					account.IPv4Address = v4
				}
				if v6, ok := addrs["v6"].(string); ok {
					account.IPv6Address = v6
				}
			}
		}
		if clientID, ok := config["client_id"].(string); ok {
			account.ClientID = clientID
		}
	}

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
		status.IPAddress = account.IPv4Address
		status.IPv6Address = account.IPv6Address
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

	// Generate WireGuard config with real assigned addresses
	addressV4 := "172.16.0.2/32"
	addressV6 := "2606:4700:110:8f77:XXXX::1/128"
	if account.IPv4Address != "" {
		addressV4 = account.IPv4Address + "/32"
	}
	if account.IPv6Address != "" {
		addressV6 = account.IPv6Address + "/128"
	}

	config := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
Address = %s
DNS = 1.1.1.1, 1.0.0.1, 2606:4700:4700::1111

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 30
`, account.PrivateKey, addressV4, addressV6, warpPublicKey, warpEndpoint)

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

// RefreshToken refreshes the WARP token with Cloudflare API
func (s *WARPService) RefreshToken() error {
	account, err := s.LoadAccount()
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("WARP not registered")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("https://api.cloudflareclient.com/v0a4005/reg/%s", account.DeviceID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+account.Token)
	req.Header.Set("CF-Client-Version", "a-6.10-4005")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed: %s (status %d)", string(body), resp.StatusCode)
	}

	// Parse updated response
	var apiResp map[string]interface{}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}

	// Update token if present
	if newToken, ok := apiResp["token"].(string); ok && newToken != "" {
		account.Token = newToken
	}

	// Update IPs if present
	if config, ok := apiResp["config"].(map[string]interface{}); ok {
		if iface, ok := config["interface"].(map[string]interface{}); ok {
			if addrs, ok := iface["addresses"].(map[string]interface{}); ok {
				if v4, ok := addrs["v4"].(string); ok {
					account.IPv4Address = v4
				}
				if v6, ok := addrs["v6"].(string); ok {
					account.IPv6Address = v6
				}
			}
		}
	}

	return s.saveAccount(account)
}

// StartAutoRefresh starts a background goroutine that refreshes the token periodically
func (s *WARPService) StartAutoRefresh(interval time.Duration) {
	s.stopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.RefreshToken(); err != nil {
					log.Printf("[WARP] Token refresh failed: %v", err)
				} else {
					log.Printf("[WARP] Token refreshed successfully")
				}
			case <-s.stopCh:
				return
			}
		}
	}()
}

// StopAutoRefresh stops the background token refresh
func (s *WARPService) StopAutoRefresh() {
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
}
