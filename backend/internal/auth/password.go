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
	ArgonTime       = 3
	ArgonMemory     = 64 * 1024
	ArgonThreads    = 4
	ArgonKeyLength  = 32
	ArgonSaltLength = 16

	legacyArgonTime = 1
)

var pepper []byte

func SetPepper(p string) {
	pepper = []byte(p)
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, ArgonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	pwd := []byte(password)
	if len(pepper) > 0 {
		pwd = append(pwd, pepper...)
	}

	hash := argon2.IDKey(pwd, salt, ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLength)
	return fmt.Sprintf("%s:%s", hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

// VerifyPassword verifies a password against a hash.
// It tries the current parameters first, then falls back to legacy parameters
// for backward compatibility with hashes created before the ArgonTime increase.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, ":")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid hash format")
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	pwd := []byte(password)
	var pepperedPwd []byte
	if len(pepper) > 0 {
		pepperedPwd = append(append([]byte{}, pwd...), pepper...)
	}

	if len(pepperedPwd) > 0 {
		hash := argon2.IDKey(pepperedPwd, salt, ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLength)
		if subtle.ConstantTimeCompare(hash, expectedHash) == 1 {
			return true, nil
		}
	}

	hash := argon2.IDKey(pwd, salt, ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLength)
	if subtle.ConstantTimeCompare(hash, expectedHash) == 1 {
		return true, nil
	}

	legacyHash := argon2.IDKey(pwd, salt, legacyArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLength)
	return subtle.ConstantTimeCompare(legacyHash, expectedHash) == 1, nil
}

// NeedsRehash checks if a password hash was created with legacy parameters
// and should be rehashed with the current parameters on next successful login.
func NeedsRehash(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, ":")
	if len(parts) != 2 {
		return false
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	pwd := []byte(password)
	if len(pepper) > 0 {
		pwd = append(pwd, pepper...)
	}

	hash := argon2.IDKey(pwd, salt, ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLength)
	return subtle.ConstantTimeCompare(hash, expectedHash) != 1
}
