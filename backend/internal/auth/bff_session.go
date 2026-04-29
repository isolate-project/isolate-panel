package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// BFFSession represents a server-side session for the Backend-for-Frontend pattern.
// The session stores the actual access and refresh tokens on the server,
// while the client only holds an opaque session ID in a httpOnly cookie.
// This prevents XSS from stealing JWT tokens stored in browser memory/localStorage.
type BFFSession struct {
	SessionID        string
	AdminID          uint
	Username         string
	IsSuperAdmin     bool
	MustChangePassword bool
	AccessToken      string
	RefreshToken     string // plaintext refresh token (server-side only)
	AccessExpiry     time.Time
	RefreshExpiry    time.Time
	CreatedAt        time.Time
	LastAccessedAt   time.Time
	UserAgent        string
	IPAddress        string
}

// BFFSessionManager manages server-side sessions for the BFF auth pattern.
// For single-node deployments, an in-memory map is sufficient.
// For multi-node, this should be backed by Redis or a shared store.
type BFFSessionManager struct {
	sessions   map[string]*BFFSession
	mu         sync.RWMutex
	ttl        time.Duration
	cleanupInterval time.Duration
	done       chan struct{}
}

// NewBFFSessionManager creates a new BFF session manager.
// ttl: session lifetime (e.g., 7 days for refresh token lifetime)
func NewBFFSessionManager(ttl time.Duration) *BFFSessionManager {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	
	sm := &BFFSessionManager{
		sessions:        make(map[string]*BFFSession),
		ttl:             ttl,
		cleanupInterval: 5 * time.Minute,
		done:            make(chan struct{}),
	}
	
	go sm.cleanupLoop()
	return sm
}

// Stop stops the background cleanup goroutine.
func (sm *BFFSessionManager) Stop() {
	close(sm.done)
}

// CreateSession creates a new BFF session and returns the session ID.
// The session ID is a cryptographically random 32-byte hex string.
func (sm *BFFSessionManager) CreateSession(adminID uint, username string, isSuperAdmin bool, mustChangePassword bool, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration, userAgent, ipAddress string) (string, error) {
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	sessionID := hex.EncodeToString(sessionBytes)
	
	now := time.Now()
	session := &BFFSession{
		SessionID:          sessionID,
		AdminID:            adminID,
		Username:           username,
		IsSuperAdmin:       isSuperAdmin,
		MustChangePassword: mustChangePassword,
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		AccessExpiry:       now.Add(accessTTL),
		RefreshExpiry:      now.Add(refreshTTL),
		CreatedAt:          now,
		LastAccessedAt:     now,
		UserAgent:          userAgent,
		IPAddress:          ipAddress,
	}
	
	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()
	
	return sessionID, nil
}

// GetSession retrieves a session by ID and updates the last accessed time.
// Returns nil if session not found or expired.
func (sm *BFFSessionManager) GetSession(sessionID string) *BFFSession {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()
	
	if !exists {
		return nil
	}
	
	now := time.Now()
	if now.After(session.RefreshExpiry) {
		sm.mu.Lock()
		delete(sm.sessions, sessionID)
		sm.mu.Unlock()
		return nil
	}
	
	sm.mu.Lock()
	session.LastAccessedAt = now
	sm.mu.Unlock()
	
	return session
}

// DeleteSession removes a session by ID.
func (sm *BFFSessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()
}

// UpdateAccessToken updates the access token and expiry for a session.
// Used after token refresh.
func (sm *BFFSessionManager) UpdateAccessToken(sessionID string, newAccessToken string, accessTTL time.Duration) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}
	
	if time.Now().After(session.RefreshExpiry) {
		delete(sm.sessions, sessionID)
		return false
	}
	
	session.AccessToken = newAccessToken
	session.AccessExpiry = time.Now().Add(accessTTL)
	session.LastAccessedAt = time.Now()
	return true
}

// UpdateRefreshToken updates both access and refresh tokens after refresh rotation.
func (sm *BFFSessionManager) UpdateRefreshToken(sessionID string, newAccessToken, newRefreshToken string, accessTTL, refreshTTL time.Duration) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}
	
	now := time.Now()
	if now.After(session.RefreshExpiry) {
		delete(sm.sessions, sessionID)
		return false
	}
	
	session.AccessToken = newAccessToken
	session.RefreshToken = newRefreshToken
	session.AccessExpiry = now.Add(accessTTL)
	session.RefreshExpiry = now.Add(refreshTTL)
	session.LastAccessedAt = now
	return true
}

// Count returns the number of active sessions.
func (sm *BFFSessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// cleanupLoop periodically removes expired sessions.
func (sm *BFFSessionManager) cleanupLoop() {
	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			sm.cleanup()
		case <-sm.done:
			return
		}
	}
}

// cleanup removes expired sessions.
func (sm *BFFSessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	for id, session := range sm.sessions {
		if now.After(session.RefreshExpiry) {
			delete(sm.sessions, id)
		}
	}
}
