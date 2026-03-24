package models

import (
	"time"
)

// CertificateStatus represents the status of a certificate
type CertificateStatus string

const (
	CertificateStatusPending  CertificateStatus = "pending"
	CertificateStatusActive   CertificateStatus = "active"
	CertificateStatusExpiring CertificateStatus = "expiring"
	CertificateStatusExpired  CertificateStatus = "expired"
	CertificateStatusRevoked  CertificateStatus = "revoked"
)

// Certificate represents a TLS certificate
type Certificate struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Domain     string `gorm:"uniqueIndex;not null" json:"domain"`
	IsWildcard bool   `gorm:"default:false" json:"is_wildcard"` // true for *.domain.com

	// Certificate data (stored in files, path in DB)
	CertPath   string `json:"cert_path"`   // Path to certificate file
	KeyPath    string `json:"key_path"`    // Path to private key file
	IssuerPath string `json:"issuer_path"` // Path to issuer certificate (if any)

	// Metadata
	CommonName      string   `json:"common_name"`
	SubjectAltNames []string `gorm:"serializer:json" json:"subject_alt_names"`
	Issuer          string   `json:"issuer"`

	// Validity
	NotBefore     time.Time  `json:"not_before"`
	NotAfter      time.Time  `json:"not_after"`
	AutoRenew     bool       `gorm:"default:true" json:"auto_renew"`
	LastRenewedAt *time.Time `json:"last_renewed_at"`

	// Status
	Status       CertificateStatus `gorm:"default:pending" json:"status"`
	StatusReason string            `json:"status_reason"` // Error message if failed

	// ACME configuration
	ACMEProvider string `gorm:"default:letsencrypt" json:"acme_provider"` // letsencrypt, zerossl
	DNSProvider  string `json:"dns_provider"`                             // cloudflare, route53, etc.

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *uint     `json:"created_by"` // Admin ID who created this
}

func (Certificate) TableName() string {
	return "certificates"
}
