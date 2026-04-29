package models

import (
	"time"
)

// WebAuthnCredential stores FIDO2/WebAuthn credentials for an admin
type WebAuthnCredential struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AdminID         uint      `gorm:"not null;index" json:"admin_id"`
	CredentialID    string    `gorm:"not null;uniqueIndex" json:"credential_id"` // Base64-encoded credential ID
	PublicKey       []byte    `gorm:"not null" json:"-"`                         // COSE public key bytes
	AttestationType string    `gorm:"not null" json:"attestation_type"`
	Transport       string    `gorm:"not null" json:"transport"` // JSON array of transports
	SignCount       uint32    `gorm:"not null;default:0" json:"sign_count"`
	AAGUID          string    `gorm:"not null;default:''" json:"aaguid"` // Authenticator Attestation GUID
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastUsedAt      *time.Time `json:"last_used_at"`
	IsBackupEligible bool     `gorm:"default:false" json:"is_backup_eligible"`
	IsBackup         bool      `gorm:"default:false" json:"is_backup"`
	Attachment       string    `gorm:"default:''" json:"attachment"` // platform, cross-platform, or empty

	// Relationship
	Admin Admin `gorm:"foreignKey:AdminID" json:"-"`
}

func (WebAuthnCredential) TableName() string {
	return "webauthn_credentials"
}

// WebAuthnSessionData stores temporary challenge data for registration/authentication
// This is stored in-memory via the WebAuthnService, not in the database
type WebAuthnSessionData struct {
	Challenge        string    `json:"challenge"`
	UserID           uint      `json:"user_id"`
	Username         string    `json:"username"`
	CredentialID     string    `json:"credential_id,omitempty"` // For authentication
	ExpiresAt        time.Time `json:"expires_at"`
	IsRegistration   bool      `json:"is_registration"`
}

// IsExpired checks if the session data has expired
func (s *WebAuthnSessionData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
