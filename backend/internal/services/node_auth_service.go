package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/crypto"
	"github.com/isolate-project/isolate-panel/internal/models"
)

const (
	// APIKeyLength is the length of generated API keys in bytes
	APIKeyLength = 32
	// APIKeyPrefix is the prefix for API keys to identify them
	APIKeyPrefix = "ip_"
	// Argon2Time is the number of iterations for Argon2
	Argon2Time = 3
	// Argon2Memory is the memory usage in KB for Argon2
	Argon2Memory = 64 * 1024
	// Argon2Threads is the number of threads for Argon2
	Argon2Threads = 4
	// Argon2KeyLength is the output key length for Argon2
	Argon2KeyLength = 32
	// SaltLength is the length of the salt in bytes
	SaltLength = 16
)

// NodeAuthService manages per-node API keys for core authentication
type NodeAuthService struct {
	db *gorm.DB
}

// NewNodeAuthService creates a new NodeAuthService
func NewNodeAuthService(db *gorm.DB) *NodeAuthService {
	return &NodeAuthService{db: db}
}

// GenerateAPIKey generates a new API key for a core.
// Returns the plaintext key (shown once) and stores the hash.
// The key format is: ip_<base64url_encoded_random_bytes>
func (s *NodeAuthService) GenerateAPIKey(coreID uint) (string, error) {
	// Generate random API key
	keyBytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Format: ip_<base64url_encoded>
	plaintextKey := APIKeyPrefix + base64.URLEncoding.EncodeToString(keyBytes)

	// Generate salt
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the key using Argon2id
	hash := argon2.IDKey([]byte(plaintextKey), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLength)

	// Get hint (last 4 characters of the key)
	hint := ""
	if len(plaintextKey) >= 4 {
		hint = plaintextKey[len(plaintextKey)-4:]
	}

	// Encrypt the plaintext key for storage
	encrypter, err := crypto.NewFieldEncrypterFromEnv()
	if err != nil {
		return "", fmt.Errorf("failed to initialize encryption: %w", err)
	}

	encryptedKey, err := encrypter.Encrypt(plaintextKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt API key: %w", err)
	}

	// Update core record
	updates := map[string]interface{}{
		"api_key_hash":       hex.EncodeToString(hash),
		"api_key_salt":       hex.EncodeToString(salt),
		"api_key_hint":       hint,
		"api_key_encrypted":  encryptedKey,
	}

	if err := s.db.Model(&models.Core{}).Where("id = ?", coreID).Updates(updates).Error; err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return plaintextKey, nil
}

// VerifyAPIKey verifies an API key against the stored hash for a core.
// Uses constant-time comparison to prevent timing attacks.
func (s *NodeAuthService) VerifyAPIKey(coreName, apiKey string) (bool, error) {
	var core models.Core
	if err := s.db.Where("name = ?", coreName).First(&core).Error; err != nil {
		return false, fmt.Errorf("core not found: %w", err)
	}

	// Check if API key is configured
	if core.APIKeyHash == "" {
		return false, fmt.Errorf("no API key configured for core %s", coreName)
	}

	// Decode stored hash and salt
	storedHash, err := hex.DecodeString(core.APIKeyHash)
	if err != nil {
		return false, fmt.Errorf("invalid stored hash: %w", err)
	}

	salt, err := hex.DecodeString(core.APIKeySalt)
	if err != nil {
		return false, fmt.Errorf("invalid stored salt: %w", err)
	}

	// Hash the provided key
	computedHash := argon2.IDKey([]byte(apiKey), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLength)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(storedHash, computedHash) == 1 {
		return true, nil
	}

	return false, nil
}

// RotateAPIKey invalidates the old API key and generates a new one.
// Returns the new plaintext key.
func (s *NodeAuthService) RotateAPIKey(coreID uint) (string, error) {
	// Verify core exists
	var core models.Core
	if err := s.db.First(&core, coreID).Error; err != nil {
		return "", fmt.Errorf("core not found: %w", err)
	}

	// Generate new key (this overwrites the old one)
	return s.GenerateAPIKey(coreID)
}

// RevokeAPIKey removes the API key from a core.
func (s *NodeAuthService) RevokeAPIKey(coreID uint) error {
	updates := map[string]interface{}{
		"api_key_hash":      "",
		"api_key_salt":      "",
		"api_key_hint":      "",
		"api_key_encrypted": "",
	}

	if err := s.db.Model(&models.Core{}).Where("id = ?", coreID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	return nil
}

// GetCoreAPIKeyHint returns the hint for a core's API key (last 4 chars)
func (s *NodeAuthService) GetCoreAPIKeyHint(coreID uint) (string, error) {
	var core models.Core
	if err := s.db.Select("api_key_hint").First(&core, coreID).Error; err != nil {
		return "", fmt.Errorf("core not found: %w", err)
	}
	return core.APIKeyHint, nil
}

// HasAPIKey checks if a core has an API key configured
func (s *NodeAuthService) HasAPIKey(coreID uint) (bool, error) {
	var core models.Core
	if err := s.db.Select("api_key_hash").First(&core, coreID).Error; err != nil {
		return false, fmt.Errorf("core not found: %w", err)
	}
	return core.APIKeyHash != "", nil
}

// GetCoreByAPIKey looks up a core by its API key.
// This is useful for authenticating incoming requests from cores.
func (s *NodeAuthService) GetCoreByAPIKey(apiKey string) (*models.Core, error) {
	// Validate key format
	if !strings.HasPrefix(apiKey, APIKeyPrefix) {
		return nil, fmt.Errorf("invalid API key format")
	}

	// Get all cores with API keys
	var cores []models.Core
	if err := s.db.Where("api_key_hash != ?", "").Find(&cores).Error; err != nil {
		return nil, fmt.Errorf("failed to query cores: %w", err)
	}

	// Try to verify against each core (this is inefficient but cores are limited)
	for _, core := range cores {
		match, err := s.verifyAPIKeyInternal(&core, apiKey)
		if err != nil {
			continue
		}
		if match {
			return &core, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// verifyAPIKeyInternal verifies an API key against a specific core (internal use)
func (s *NodeAuthService) verifyAPIKeyInternal(core *models.Core, apiKey string) (bool, error) {
	if core.APIKeyHash == "" {
		return false, fmt.Errorf("no API key configured")
	}

	storedHash, err := hex.DecodeString(core.APIKeyHash)
	if err != nil {
		return false, err
	}

	salt, err := hex.DecodeString(core.APIKeySalt)
	if err != nil {
		return false, err
	}

	computedHash := argon2.IDKey([]byte(apiKey), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLength)
	return subtle.ConstantTimeCompare(storedHash, computedHash) == 1, nil
}

// GenerateAPIKeyForCoreName generates an API key for a core by name
func (s *NodeAuthService) GenerateAPIKeyForCoreName(coreName string) (string, error) {
	var core models.Core
	if err := s.db.Where("name = ?", coreName).First(&core).Error; err != nil {
		return "", fmt.Errorf("core not found: %w", err)
	}
	return s.GenerateAPIKey(core.ID)
}

// ListCoresWithAPIKeys returns all cores that have API keys configured
func (s *NodeAuthService) ListCoresWithAPIKeys() ([]models.Core, error) {
	var cores []models.Core
	if err := s.db.Where("api_key_hash != ?", "").Find(&cores).Error; err != nil {
		return nil, fmt.Errorf("failed to query cores: %w", err)
	}
	return cores, nil
}

// GetCoreAPIKey retrieves the plaintext API key for a core by ID.
// Returns an error if the key was generated before encrypted storage was available.
func (s *NodeAuthService) GetCoreAPIKey(coreID uint) (string, error) {
	var core models.Core
	if err := s.db.Select("api_key_hash", "api_key_encrypted").First(&core, coreID).Error; err != nil {
		return "", fmt.Errorf("core not found: %w", err)
	}

	// Check if API key exists
	if core.APIKeyHash == "" {
		return "", fmt.Errorf("no API key configured for core")
	}

	// Check if encrypted key is available
	if core.APIKeyEncrypted == "" {
		return "", fmt.Errorf("API key was generated before encrypted storage was available - please rotate the key")
	}

	// Decrypt the key
	encrypter, err := crypto.NewFieldEncrypterFromEnv()
	if err != nil {
		return "", fmt.Errorf("failed to initialize encryption: %w", err)
	}

	plaintextKey, err := encrypter.Decrypt(core.APIKeyEncrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return plaintextKey, nil
}

// GetCoreAPIKeyByName retrieves the plaintext API key for a core by name.
// Returns an error if the key was generated before encrypted storage was available.
func (s *NodeAuthService) GetCoreAPIKeyByName(coreName string) (string, error) {
	var core models.Core
	if err := s.db.Select("id", "api_key_hash", "api_key_encrypted").Where("name = ?", coreName).First(&core).Error; err != nil {
		return "", fmt.Errorf("core not found: %w", err)
	}
	return s.GetCoreAPIKey(core.ID)
}
