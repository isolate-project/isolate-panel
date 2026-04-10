package services

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/isolate-project/isolate-panel/internal/acme"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// CertificateService manages TLS certificates
type CertificateService struct {
	db                  *gorm.DB
	acmeClient          *acme.ACMEClient
	certDir             string
	email               string
	dnsProvider         string
	credentials         map[string]string
	mu                  sync.Mutex
	stopRenewal         chan struct{}
	renewalTicker       *time.Ticker
	notificationService *NotificationService
}

// CertificateServiceConfig holds configuration for CertificateService
type CertificateServiceConfig struct {
	CertDir     string
	Email       string
	DNSProvider string // "cloudflare"
	Credentials map[string]string
	Staging     bool // Use ACME staging server
}

// SetNotificationService sets the notification service
func (cs *CertificateService) SetNotificationService(ns *NotificationService) {
	cs.notificationService = ns
}

// NewCertificateService creates a new certificate service
func NewCertificateService(db *gorm.DB, config CertificateServiceConfig) (*CertificateService, error) {
	// Create certificate directory
	if err := os.MkdirAll(config.CertDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %w", err)
	}

	cs := &CertificateService{
		db:          db,
		certDir:     config.CertDir,
		email:       config.Email,
		dnsProvider: config.DNSProvider,
		credentials: config.Credentials,
		stopRenewal: make(chan struct{}),
	}

	// Initialize ACME client if credentials are provided
	if config.DNSProvider != "" && len(config.Credentials) > 0 {
		acmeClient, err := acme.NewACMEClient(acme.ACMEConfig{
			Email:       config.Email,
			DNSProvider: config.DNSProvider,
			Credentials: config.Credentials,
			CertDir:     config.CertDir,
			Staging:     config.Staging,
		})
		if err != nil {
			// Non-fatal: continue without ACME, manual upload still works
			logger.Log.Warn().Err(err).Msg("ACME client initialization failed (manual certificate upload still available)")
		} else {
			cs.acmeClient = acmeClient
		}
	}

	// Start auto-renewal checker
	cs.startRenewalChecker()

	return cs, nil
}

// RequestCertificate requests a new certificate via ACME
func (cs *CertificateService) RequestCertificate(domain string, isWildcard bool) (*models.Certificate, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if certificate already exists (before ACME check for better error messages)
	var existing models.Certificate
	if err := cs.db.Where("domain = ?", domain).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("certificate for domain %s already exists", domain)
	}

	if cs.acmeClient == nil {
		return nil, fmt.Errorf("ACME client not initialized - configure DNS provider credentials first")
	}

	// Request certificate from ACME
	certRes, err := cs.acmeClient.RequestCertificate(domain, isWildcard)
	if err != nil {
		return nil, fmt.Errorf("failed to request certificate: %w", err)
	}

	// Parse certificate to extract metadata
	certBlock, _ := pem.Decode(certRes.Certificate)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to parse certificate")
	}

	x509Cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse x509 certificate: %w", err)
	}

	// Save certificate files
	certPath, keyPath, issuerPath, err := cs.acmeClient.SaveCertificate(certRes, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to save certificate files: %w", err)
	}

	// Create certificate record
	cert := &models.Certificate{
		Domain:          domain,
		IsWildcard:      isWildcard,
		CertPath:        certPath,
		KeyPath:         keyPath,
		IssuerPath:      issuerPath,
		CommonName:      x509Cert.Subject.CommonName,
		SubjectAltNames: x509Cert.DNSNames,
		Issuer:          x509Cert.Issuer.CommonName,
		NotBefore:       x509Cert.NotBefore,
		NotAfter:        x509Cert.NotAfter,
		AutoRenew:       true,
		Status:          models.CertificateStatusActive,
		ACMEProvider:    "letsencrypt",
		DNSProvider:     cs.dnsProvider,
	}

	if err := cs.db.Create(cert).Error; err != nil {
		// Rollback: delete certificate files
		os.RemoveAll(filepath.Dir(certPath))
		return nil, fmt.Errorf("failed to save certificate record: %w", err)
	}

	return cert, nil
}

// RenewCertificate renews an existing certificate
func (cs *CertificateService) RenewCertificate(certID uint) (*models.Certificate, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var cert models.Certificate
	if err := cs.db.First(&cert, certID).Error; err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}

	if cs.acmeClient == nil {
		return nil, fmt.Errorf("ACME client not initialized")
	}

	// Load existing certificate
	certRes := &certificate.Resource{
		Domain:      cert.Domain,
		Certificate: readFile(cert.CertPath),
		PrivateKey:  readFile(cert.KeyPath),
	}

	if cert.IssuerPath != "" {
		certRes.IssuerCertificate = readFile(cert.IssuerPath)
	}

	// Renew certificate
	renewed, err := cs.acmeClient.RenewCertificate(certRes)
	if err != nil {
		return nil, fmt.Errorf("failed to renew certificate: %w", err)
	}

	// Parse renewed certificate
	certBlock, _ := pem.Decode(renewed.Certificate)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to parse renewed certificate")
	}

	x509Cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse x509 certificate: %w", err)
	}

	// Save renewed certificate files
	certPath, keyPath, issuerPath, err := cs.acmeClient.SaveCertificate(renewed, cert.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to save renewed certificate files: %w", err)
	}

	// Update certificate record
	cert.CertPath = certPath
	cert.KeyPath = keyPath
	cert.IssuerPath = issuerPath
	cert.NotBefore = x509Cert.NotBefore
	cert.NotAfter = x509Cert.NotAfter
	cert.LastRenewedAt = ptr(time.Now())
	cert.Status = models.CertificateStatusActive

	if err := cs.db.Save(&cert).Error; err != nil {
		return nil, fmt.Errorf("failed to update certificate record: %w", err)
	}

	// Send notification
	if cs.notificationService != nil {
		cs.notificationService.NotifyCertRenewed(&cert)
	}

	return &cert, nil
}

// RevokeCertificate revokes a certificate
func (cs *CertificateService) RevokeCertificate(certID uint) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var cert models.Certificate
	if err := cs.db.First(&cert, certID).Error; err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}

	if cs.acmeClient == nil {
		return fmt.Errorf("ACME client not initialized")
	}

	// Read certificate
	certPEM := readFile(cert.CertPath)
	if len(certPEM) == 0 {
		return fmt.Errorf("certificate file not found")
	}

	// Revoke certificate
	if err := cs.acmeClient.RevokeCertificate(certPEM); err != nil {
		return fmt.Errorf("failed to revoke certificate: %w", err)
	}

	// Update status
	cert.Status = models.CertificateStatusRevoked
	if err := cs.db.Save(&cert).Error; err != nil {
		return fmt.Errorf("failed to update certificate status: %w", err)
	}

	return nil
}

// DeleteCertificate deletes a certificate (files + DB record)
func (cs *CertificateService) DeleteCertificate(certID uint) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var cert models.Certificate
	if err := cs.db.First(&cert, certID).Error; err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}

	// Delete certificate files
	certDir := filepath.Dir(cert.CertPath)
	if err := os.RemoveAll(certDir); err != nil {
		return fmt.Errorf("failed to delete certificate files: %w", err)
	}

	// Delete DB record
	if err := cs.db.Delete(&cert).Error; err != nil {
		return fmt.Errorf("failed to delete certificate record: %w", err)
	}

	return nil
}

// GetCertificate retrieves a certificate by ID
func (cs *CertificateService) GetCertificate(certID uint) (*models.Certificate, error) {
	var cert models.Certificate
	if err := cs.db.First(&cert, certID).Error; err != nil {
		return nil, err
	}
	return &cert, nil
}

// ListCertificates lists all certificates
func (cs *CertificateService) ListCertificates() ([]models.Certificate, error) {
	var certs []models.Certificate
	if err := cs.db.Order("created_at DESC").Find(&certs).Error; err != nil {
		return nil, err
	}
	return certs, nil
}

// UpdateCertificateStatus updates certificate status based on expiry
func (cs *CertificateService) UpdateCertificateStatus(cert *models.Certificate) {
	now := time.Now()

	if cert.NotAfter.Before(now) {
		cert.Status = models.CertificateStatusExpired
	} else if acme.NeedsRenewal(cert.NotAfter) {
		cert.Status = models.CertificateStatusExpiring
	} else {
		cert.Status = models.CertificateStatusActive
	}

	cs.db.Save(cert)
}

// startRenewalChecker starts a background goroutine to check for expiring certificates
func (cs *CertificateService) startRenewalChecker() {
	cs.renewalTicker = time.NewTicker(24 * time.Hour) // Check daily

	go func() {
		for {
			select {
			case <-cs.renewalTicker.C:
				cs.checkAndRenewCertificates()
			case <-cs.stopRenewal:
				cs.renewalTicker.Stop()
				return
			}
		}
	}()
}

// checkAndRenewCertificates checks for certificates needing renewal and renews them
func (cs *CertificateService) checkAndRenewCertificates() {
	var certs []models.Certificate
	if err := cs.db.Where("auto_renew = ? AND status IN (?, ?)", true, models.CertificateStatusActive, models.CertificateStatusExpiring).Find(&certs).Error; err != nil {
		return
	}

	for _, cert := range certs {
		if acme.NeedsRenewal(cert.NotAfter) {
			_, err := cs.RenewCertificate(cert.ID)
			if err != nil {
				// Log error but continue with other certificates
				continue
			}
		}
	}
}

// Stop stops the certificate service
func (cs *CertificateService) Stop() {
	close(cs.stopRenewal)
}

// Helper functions

func readFile(path string) []byte {
	if path == "" {
		return nil
	}
	data, _ := os.ReadFile(path)
	return data
}

func ptr[T any](v T) *T {
	return &v
}
