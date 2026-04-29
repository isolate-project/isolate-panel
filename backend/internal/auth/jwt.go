package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const minJWTKeyLength = 64

// AdminValidator checks if an admin's claims are still valid against the database
type AdminValidator func(adminID uint) (isActive bool, isSuperAdmin bool, mustChangePassword bool, err error)

type TokenService struct {
	secret          []byte
	kid             string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string

	blacklist   map[string]time.Time
	blacklistMu sync.RWMutex
	done        chan struct{}

	adminValidator AdminValidator
	db             *gorm.DB
}

type Claims struct {
	AdminID            uint   `json:"admin_id"`
	Username           string `json:"username"`
	IsSuperAdmin       bool   `json:"is_super_admin"`
	MustChangePassword bool   `json:"must_change_password"`
	Permissions        uint64 `json:"permissions"`
	jwt.RegisteredClaims
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration, validator AdminValidator, db *gorm.DB) (*TokenService, error) {
	if len(secret) < minJWTKeyLength {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes (512 bits) for HS256 security", minJWTKeyLength)
	}

	kidHash := sha256.Sum256([]byte(secret))
	kid := hex.EncodeToString(kidHash[:8])

	ts := &TokenService{
		secret:          []byte(secret),
		kid:             kid,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		issuer:          "isolate-panel",
		blacklist:       make(map[string]time.Time),
		done:            make(chan struct{}),
		adminValidator:  validator,
		db:              db,
	}
	ts.loadBlacklistFromDB()
	go ts.cleanupBlacklist()
	return ts, nil
}

func (ts *TokenService) cleanupBlacklist() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			ts.blacklistMu.Lock()
			for hash, expiry := range ts.blacklist {
				if now.After(expiry) {
					delete(ts.blacklist, hash)
				}
			}
			ts.blacklistMu.Unlock()
			if ts.db != nil {
				ts.db.Exec("DELETE FROM revoked_tokens WHERE expires_at < ?", now)
			}
		case <-ts.done:
			return
		}
	}
}

// Stop stops the background cleanup goroutine
func (ts *TokenService) Stop() {
	close(ts.done)
}

func (ts *TokenService) loadBlacklistFromDB() {
	if ts.db == nil {
		return
	}
	var entries []struct {
		TokenHash string
		ExpiresAt time.Time
	}
	ts.db.Raw("SELECT token_hash, expires_at FROM revoked_tokens WHERE expires_at > ?", time.Now()).Scan(&entries)
	ts.blacklistMu.Lock()
	for _, e := range entries {
		ts.blacklist[e.TokenHash] = e.ExpiresAt
	}
	ts.blacklistMu.Unlock()
}

func (ts *TokenService) GenerateAccessToken(adminID uint, username string, isSuperAdmin bool, mustChangePassword bool) (string, error) {
	var perms uint64
	if isSuperAdmin {
		perms = uint64(NewPermissions(
			PermViewDashboard,
			PermManageUsers,
			PermManageInbounds,
			PermManageOutbounds,
			PermManageCores,
			PermManageSettings,
			PermViewLogs,
			PermManageCertificates,
			PermManageBackups,
			PermSuperAdmin,
		))
	}
	return ts.GenerateAccessTokenWithPermissions(adminID, username, isSuperAdmin, mustChangePassword, perms)
}

func (ts *TokenService) GenerateAccessTokenWithPermissions(adminID uint, username string, isSuperAdmin bool, mustChangePassword bool, permissions uint64) (string, error) {
	now := time.Now()

	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("failed to generate jti: %w", err)
	}

	claims := &Claims{
		AdminID:            adminID,
		Username:           username,
		IsSuperAdmin:       isSuperAdmin,
		MustChangePassword: mustChangePassword,
		Permissions:        permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    ts.issuer,
			Subject:   fmt.Sprintf("%d", adminID),
			ID:        hex.EncodeToString(jtiBytes),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = ts.kid
	return token.SignedString(ts.secret)
}

// GenerateRefreshToken generates a random refresh token
func (ts *TokenService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// BlacklistAccessToken adds a token to the in-memory blacklist.
// The token is automatically removed after its original expiry time.
func (ts *TokenService) BlacklistAccessToken(tokenString string) {
	hash := tokenHash(tokenString)
	expiry := time.Now().Add(ts.accessTokenTTL)

	ts.blacklistMu.Lock()
	ts.blacklist[hash] = expiry
	ts.blacklistMu.Unlock()

	if ts.db != nil {
		ts.db.Exec("INSERT OR IGNORE INTO revoked_tokens (token_hash, expires_at) VALUES (?, ?)", hash, expiry)
	}
}

func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ValidateAccessToken validates and parses a JWT access token
func (ts *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	// Check blacklist first
	hash := tokenHash(tokenString)
	ts.blacklistMu.RLock()
	_, revoked := ts.blacklist[hash]
	ts.blacklistMu.RUnlock()
	if revoked {
		return nil, fmt.Errorf("token has been revoked")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ts.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Validate admin state from database if validator is provided
		if ts.adminValidator != nil {
			isActive, isSuperAdmin, mustChangePassword, err := ts.adminValidator(claims.AdminID)
			if err != nil {
				return nil, fmt.Errorf("failed to validate admin state: %w", err)
			}
			if !isActive {
				return nil, fmt.Errorf("admin account is deactivated")
			}
			claims.IsSuperAdmin = isSuperAdmin
			claims.MustChangePassword = mustChangePassword
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (ts *TokenService) GetAccessTokenTTL() time.Duration {
	return ts.accessTokenTTL
}

func (ts *TokenService) GetRefreshTokenTTL() time.Duration {
	return ts.refreshTokenTTL
}

// JWKSKey represents a JWK Set key entry.
type JWKSKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	K   string `json:"k"`
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

func (ts *TokenService) GetJWKS() *JWKS {
	return &JWKS{
		Keys: []JWKSKey{
			{
				Kty: "oct",
				Kid: ts.kid,
				Use: "sig",
				Alg: "HS256",
			},
		},
	}
}
