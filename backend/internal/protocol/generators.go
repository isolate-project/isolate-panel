package protocol

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

// GenerateUUIDv4 generates a new UUID v4 string
func GenerateUUIDv4() string {
	return uuid.New().String()
}

// GeneratePassword generates a cryptographically secure password of the given length
func GeneratePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback: should never happen with crypto/rand
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// GenerateBase64Token generates a base64-encoded random token
func GenerateBase64Token(byteLength int) string {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateRandomPath generates a random URL path with optional prefix
func GenerateRandomPath(prefix string) string {
	suffix := GeneratePassword(8)
	if prefix == "" {
		return "/" + suffix
	}
	return fmt.Sprintf("/%s/%s", prefix, suffix)
}

// GenerateShortID generates a short alphanumeric ID (for short URLs, etc.)
func GenerateShortID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// AutoGenerate calls the appropriate generator function by name
func AutoGenerate(funcName string) (interface{}, error) {
	switch funcName {
	case "generate_uuid_v4":
		return GenerateUUIDv4(), nil
	case "generate_password_8":
		return GeneratePassword(8), nil
	case "generate_password_16":
		return GeneratePassword(16), nil
	case "generate_password_32":
		return GeneratePassword(32), nil
	case "generate_base64_token_32":
		return GenerateBase64Token(32), nil
	case "generate_base64_token_44":
		return GenerateBase64Token(44), nil
	case "generate_random_path":
		return GenerateRandomPath(""), nil
	case "generate_short_id_8":
		return GenerateShortID(8), nil
	default:
		return nil, fmt.Errorf("unknown auto-generate function: %s", funcName)
	}
}
