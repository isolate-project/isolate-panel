package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
)

// DefaultEncryptionKeyPath is the default file path for the user credential encryption key
const DefaultEncryptionKeyPath = "/app/data/.user_cred_key"

var (
	encryptionKeyPath = getKeyPath()
	encryptionKeyMu   sync.RWMutex
	testKey           []byte // in-memory override for tests
)

func getKeyPath() string {
	if path := os.Getenv("ISOLATE_USER_CRED_KEY_PATH"); path != "" {
		return path
	}
	return DefaultEncryptionKeyPath
}

// SetEncryptionKeyPath sets the key file path (for configuration/testing)
func SetEncryptionKeyPath(path string) {
	encryptionKeyMu.Lock()
	defer encryptionKeyMu.Unlock()
	encryptionKeyPath = path
}

// SetTestEncryptionKey sets an in-memory encryption key (for tests only)
func SetTestEncryptionKey(key []byte) {
	encryptionKeyMu.Lock()
	defer encryptionKeyMu.Unlock()
	testKey = key
}

// GetOrCreateEncryptionKey reads or creates the AES-256 key for user credential encryption
func GetOrCreateEncryptionKey() ([]byte, error) {
	encryptionKeyMu.RLock()
	if testKey != nil {
		k := testKey
		encryptionKeyMu.RUnlock()
		return k, nil
	}
	path := encryptionKeyPath
	encryptionKeyMu.RUnlock()

	data, err := os.ReadFile(path)
	if err == nil && len(data) == 32 {
		return data, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptCredential encrypts a plaintext credential using AES-256-GCM
func EncryptCredential(plaintext string) (string, error) {
	key, err := GetOrCreateEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// DecryptCredential decrypts an AES-256-GCM encrypted credential
func DecryptCredential(encoded string) (string, error) {
	key, err := GetOrCreateEncryptionKey()
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// IsEncrypted checks if a string looks like an encrypted credential (hex-encoded AES-GCM)
func IsEncrypted(s string) bool {
	_, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	return len(s) > 64
}
