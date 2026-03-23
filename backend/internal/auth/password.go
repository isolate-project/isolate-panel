package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters (OWASP recommended)
	ArgonTime       = 1
	ArgonMemory     = 64 * 1024 // 64 MB
	ArgonThreads    = 4
	ArgonKeyLength  = 32
	ArgonSaltLength = 16
)

// HashPassword hashes a password using Argon2id
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, ArgonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		ArgonTime,
		ArgonMemory,
		ArgonThreads,
		ArgonKeyLength,
	)

	// Encode as: salt:hash (both hex encoded)
	return fmt.Sprintf("%s:%s", hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Split salt and hash
	parts := strings.Split(encodedHash, ":")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Decode salt and hash
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the provided password with the same salt
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		ArgonTime,
		ArgonMemory,
		ArgonThreads,
		ArgonKeyLength,
	)

	// Compare hashes using constant-time comparison
	return subtle.ConstantTimeCompare(hash, expectedHash) == 1, nil
}
