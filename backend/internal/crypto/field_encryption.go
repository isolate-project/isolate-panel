// Package crypto provides field-level encryption for sensitive database fields.
// Uses AES-256-GCM for authenticated encryption at rest.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	// KeySize is the required key size for AES-256 (32 bytes = 256 bits)
	KeySize = 32
	// DefaultEncryptionKeyPath is the default file path for the data encryption key
	DefaultEncryptionKeyPath = "/app/data/.field_encryption_key"
)

var (
	globalKeyPath = DefaultEncryptionKeyPath
	globalKeyMu   sync.RWMutex
	testKey       []byte // in-memory override for tests
)

// FieldEncrypter provides AES-256-GCM encryption for database fields
type FieldEncrypter struct {
	key []byte // 32 bytes for AES-256
}

// NewFieldEncrypter creates a new FieldEncrypter with the provided key.
// The key must be exactly 32 bytes for AES-256.
func NewFieldEncrypter(key []byte) (*FieldEncrypter, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("encryption key must be %d bytes, got %d", KeySize, len(key))
	}
	// Create a copy to prevent external modification
	keyCopy := make([]byte, KeySize)
	copy(keyCopy, key)
	return &FieldEncrypter{key: keyCopy}, nil
}

// NewFieldEncrypterFromEnv creates a new FieldEncrypter from environment variable or file.
// Checks DATA_ENCRYPTION_KEY env var first, then DATA_ENCRYPTION_KEY_FILE.
func NewFieldEncrypterFromEnv() (*FieldEncrypter, error) {
	key, err := LoadEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load encryption key: %w", err)
	}
	return NewFieldEncrypter(key)
}

// SetKeyPath sets the key file path (for configuration/testing)
func SetKeyPath(path string) {
	globalKeyMu.Lock()
	defer globalKeyMu.Unlock()
	globalKeyPath = path
}

// SetTestKey sets an in-memory encryption key (for tests only)
func SetTestKey(key []byte) {
	globalKeyMu.Lock()
	defer globalKeyMu.Unlock()
	testKey = make([]byte, len(key))
	copy(testKey, key)
}

// ClearTestKey clears the test encryption key
func ClearTestKey() {
	globalKeyMu.Lock()
	defer globalKeyMu.Unlock()
	testKey = nil
}

// LoadEncryptionKey reads or creates the AES-256 key for field encryption.
// Priority:
// 1. Test key (if set)
// 2. DATA_ENCRYPTION_KEY environment variable (base64 encoded)
// 3. DATA_ENCRYPTION_KEY_FILE environment variable (path to file with base64 encoded key)
// 4. Default key file path (reads or generates new key)
func LoadEncryptionKey() ([]byte, error) {
	globalKeyMu.RLock()
	if testKey != nil {
		k := make([]byte, len(testKey))
		copy(k, testKey)
		globalKeyMu.RUnlock()
		return k, nil
	}
	globalKeyMu.RUnlock()

	// Check environment variable for base64-encoded key
	if envKey := os.Getenv("DATA_ENCRYPTION_KEY"); envKey != "" {
		key, err := base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 in DATA_ENCRYPTION_KEY: %w", err)
		}
		if len(key) != KeySize {
			return nil, fmt.Errorf("DATA_ENCRYPTION_KEY must decode to %d bytes, got %d", KeySize, len(key))
		}
		return key, nil
	}

	// Check environment variable for key file path
	keyPath := globalKeyPath
	if envPath := os.Getenv("DATA_ENCRYPTION_KEY_FILE"); envPath != "" {
		keyPath = envPath
	}

	// Read or generate key from file
	return loadOrCreateKeyFile(keyPath)
}

// loadOrCreateKeyFile reads an existing key file or generates a new one
func loadOrCreateKeyFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		// Try to decode as base64 first
		key, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(key) == KeySize {
			return key, nil
		}
		// Try raw bytes
		if len(data) == KeySize {
			return data, nil
		}
		return nil, fmt.Errorf("existing key file has invalid size: %d bytes", len(data))
	}

	// Generate new key
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Write key to file (base64 encoded for readability)
	keyB64 := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(path, []byte(keyB64), 0600); err != nil {
		return nil, fmt.Errorf("failed to write encryption key file: %w", err)
	}

	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64(nonce+ciphertext)
func (e *FieldEncrypter) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	ciphertext, err := e.EncryptBytes([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64(nonce+ciphertext) and returns plaintext
func (e *FieldEncrypter) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	// Check if this looks like an encrypted value (base64)
	if !isBase64(ciphertext) {
		// Not encrypted, return as-is (for migration/compatibility)
		return ciphertext, nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// Not valid base64, might be plaintext from before encryption
		return ciphertext, nil
	}
	plaintext, err := e.DecryptBytes(data)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptBytes encrypts plaintext bytes using AES-256-GCM
// Returns: nonce || ciphertext (authenticated encryption)
func (e *FieldEncrypter) EncryptBytes(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBytes decrypts ciphertext bytes (nonce || ciphertext) using AES-256-GCM
func (e *FieldEncrypter) DecryptBytes(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return []byte{}, nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(ciphertext))
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key or corrupted data): %w", err)
	}

	return plaintext, nil
}

// IsEncrypted checks if a string appears to be encrypted (base64 encoded with minimum length)
func (e *FieldEncrypter) IsEncrypted(s string) bool {
	if s == "" {
		return false
	}
	// Encrypted values are base64 encoded and have minimum length
	// (nonce + tag + at least some ciphertext)
	if len(s) < 44 { // 32 bytes = 44 base64 chars minimum for empty plaintext + nonce + tag
		return false
	}
	return isBase64(s)
}

// isBase64 checks if a string is valid base64
func isBase64(s string) bool {
	// Quick check: base64 length should be divisible by 4 (with padding)
	// or contain only base64 characters
	if len(s)%4 != 0 && !strings.ContainsAny(s, "=") {
		// Might still be valid base64 without padding, check characters
		for _, c := range s {
			if !isBase64Char(c) {
				return false
			}
		}
		return true
	}
	// Try to decode
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func isBase64Char(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '+' || c == '/' || c == '='
}

// GenerateKey generates a new random 32-byte encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a new random encryption key as base64 string
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
