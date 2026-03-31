package auth_test

import (
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/auth"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "mySecurePassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // Argon2id can hash empty strings
		},
		{
			name:     "long password",
			password: "verylongpasswordthatexceeds100charactersverylongpasswordthatexceeds100charactersverylongpasswordthatexceeds100characters",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := auth.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
			if !tt.wantErr && hash == tt.password {
				t.Error("HashPassword() returned plaintext password")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testPassword123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			password: "wrongPassword",
			hash:     hash,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
			wantErr:  false,
		},
		{
			name:     "invalid hash",
			password: password,
			hash:     "invalid-hash",
			want:     false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.VerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "testPassword"

	// Hash the same password twice
	hash1, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("First hash failed: %v", err)
	}

	hash2, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Second hash failed: %v", err)
	}

	// Hashes should be different (due to random salt)
	if hash1 == hash2 {
		t.Error("HashPassword() produced identical hashes for same password (salt not random)")
	}

	// But both should verify correctly
	valid1, err := auth.VerifyPassword(password, hash1)
	if err != nil || !valid1 {
		t.Error("First hash doesn't verify")
	}
	valid2, err := auth.VerifyPassword(password, hash2)
	if err != nil || !valid2 {
		t.Error("Second hash doesn't verify")
	}
}

func TestTokenService_GenerateAccessToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := auth.NewTokenService(secret, accessTTL, refreshTTL)

	adminID := uint(1)
	username := "testadmin"
	isSuperAdmin := true

	token, err := service.GenerateAccessToken(adminID, username, isSuperAdmin)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}
}

func TestTokenService_ValidateAccessToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := auth.NewTokenService(secret, accessTTL, refreshTTL)

	adminID := uint(1)
	username := "testadmin"
	isSuperAdmin := true

	// Generate token
	token, err := service.GenerateAccessToken(adminID, username, isSuperAdmin)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Validate token
	claims, err := service.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.AdminID != adminID {
		t.Errorf("Claims.AdminID = %v, want %v", claims.AdminID, adminID)
	}

	if claims.Username != username {
		t.Errorf("Claims.Username = %v, want %v", claims.Username, username)
	}

	if claims.IsSuperAdmin != isSuperAdmin {
		t.Errorf("Claims.IsSuperAdmin = %v, want %v", claims.IsSuperAdmin, isSuperAdmin)
	}
}

func TestTokenService_ValidateAccessToken_Invalid(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := auth.NewTokenService(secret, accessTTL, refreshTTL)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid token format",
			token:   "invalid.token.format",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateAccessToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenService_GenerateRefreshToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := auth.NewTokenService(secret, accessTTL, refreshTTL)

	token, err := service.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	// Verify token is hex encoded (64 chars for 32 bytes)
	if len(token) != 64 {
		t.Errorf("GenerateRefreshToken() returned token of length %d, expected 64", len(token))
	}
}

func TestTokenService_GetRefreshTokenTTL(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service := auth.NewTokenService(secret, accessTTL, refreshTTL)

	ttl := service.GetRefreshTokenTTL()
	if ttl != refreshTTL {
		t.Errorf("GetRefreshTokenTTL() = %v, want %v", ttl, refreshTTL)
	}
}

func TestTokenService_DifferentSecrets(t *testing.T) {
	secret1 := "secret-one"
	secret2 := "secret-two"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	service1 := auth.NewTokenService(secret1, accessTTL, refreshTTL)
	service2 := auth.NewTokenService(secret2, accessTTL, refreshTTL)

	adminID := uint(1)
	username := "testadmin"
	isSuperAdmin := false

	// Generate token with service1
	token, err := service1.GenerateAccessToken(adminID, username, isSuperAdmin)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Try to validate with service2 (different secret)
	_, err = service2.ValidateAccessToken(token)
	if err == nil {
		t.Error("ValidateAccessToken() should fail with different secret")
	}
}

func TestTokenService_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-signing"
	accessTTL := 1 * time.Millisecond // Very short TTL
	refreshTTL := 7 * 24 * time.Hour

	service := auth.NewTokenService(secret, accessTTL, refreshTTL)

	adminID := uint(1)
	username := "testadmin"
	isSuperAdmin := false

	// Generate token
	token, err := service.GenerateAccessToken(adminID, username, isSuperAdmin)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = service.ValidateAccessToken(token)
	if err == nil {
		t.Error("ValidateAccessToken() should fail for expired token")
	}
}
