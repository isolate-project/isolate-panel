package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/registration"
)

// ACMEUser implements lego.User interface
type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string                        { return u.Email }
func (u *ACMEUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// ACMEClient wraps lego client for certificate operations
type ACMEClient struct {
	client      *lego.Client
	dnsProvider string
	credentials map[string]string
	certDir     string
}

// ACMEConfig holds ACME client configuration
type ACMEConfig struct {
	Email       string
	DNSProvider string // "cloudflare"
	Credentials map[string]string
	CertDir     string // Directory to store certificates
	Staging     bool   // Use staging server for testing
}

// NewACMEClient creates a new ACME client
func NewACMEClient(config ACMEConfig) (*ACMEClient, error) {
	// Generate or load private key for ACME account
	key, err := loadOrGenerateAccountKey(filepath.Join(config.CertDir, "account.key"))
	if err != nil {
		return nil, fmt.Errorf("failed to load/generate account key: %w", err)
	}

	user := &ACMEUser{
		Email: config.Email,
		key:   key,
	}

	// Configure lego client
	lc := lego.NewConfig(user)
	lc.Certificate.KeyType = certcrypto.RSA2048

	// Set CA directory (production or staging)
	if config.Staging {
		lc.CADirURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	} else {
		lc.CADirURL = "https://acme-v02.api.letsencrypt.org/directory"
	}

	client, err := lego.NewClient(lc)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %w", err)
	}

	// Register with ACME server
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, fmt.Errorf("failed to register with ACME: %w", err)
	}
	user.Registration = reg

	ac := &ACMEClient{
		client:      client,
		dnsProvider: config.DNSProvider,
		credentials: config.Credentials,
		certDir:     config.CertDir,
	}

	// Setup DNS challenge provider
	if err := ac.setupDNSChallenge(); err != nil {
		return nil, fmt.Errorf("failed to setup DNS challenge: %w", err)
	}

	return ac, nil
}

// setupDNSChallenge configures DNS-01 challenge provider
func (ac *ACMEClient) setupDNSChallenge() error {
	var provider challenge.Provider
	var err error

	switch ac.dnsProvider {
	case "cloudflare":
		provider, err = ac.createCloudflareProvider()
	default:
		return fmt.Errorf("unsupported DNS provider: %s", ac.dnsProvider)
	}

	if err != nil {
		return err
	}

	return ac.client.Challenge.SetDNS01Provider(provider)
}

// createCloudflareProvider creates Cloudflare DNS provider
func (ac *ACMEClient) createCloudflareProvider() (challenge.Provider, error) {
	config := cloudflare.NewDefaultConfig()

	if apiKey, ok := ac.credentials["api_key"]; ok {
		config.AuthKey = apiKey
	}
	if apiEmail, ok := ac.credentials["email"]; ok {
		config.AuthEmail = apiEmail
	}
	if apiToken, ok := ac.credentials["api_token"]; ok {
		config.AuthToken = apiToken
	}

	// Validate configuration
	if config.AuthKey == "" && config.AuthToken == "" {
		return nil, fmt.Errorf("cloudflare: API key or token is required")
	}

	return cloudflare.NewDNSProviderConfig(config)
}

// RequestCertificate requests a new certificate
func (ac *ACMEClient) RequestCertificate(domain string, isWildcard bool) (*certificate.Resource, error) {
	domains := []string{domain}

	// Add wildcard domain if requested
	if isWildcard {
		wildcardDomain := "*." + domain
		// Add base domain if not already present
		if domain != "" {
			domains = append(domains, wildcardDomain)
		}
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	cert, err := ac.client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain certificate: %w", err)
	}

	return cert, nil
}

// RenewCertificate renews an existing certificate
func (ac *ACMEClient) RenewCertificate(certRes *certificate.Resource) (*certificate.Resource, error) {
	renewed, err := ac.client.Certificate.Renew(*certRes, true, false, "")
	if err != nil {
		return nil, fmt.Errorf("failed to renew certificate: %w", err)
	}

	return renewed, nil
}

// RevokeCertificate revokes a certificate
func (ac *ACMEClient) RevokeCertificate(certPEM []byte) error {
	return ac.client.Certificate.Revoke(certPEM)
}

// SaveCertificate saves certificate files to disk
func (ac *ACMEClient) SaveCertificate(cert *certificate.Resource, domain string) (certPath, keyPath, issuerPath string, err error) {
	// Create domain directory
	domainDir := filepath.Join(ac.certDir, domain)
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return "", "", "", fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Save certificate
	certPath = filepath.Join(domainDir, "cert.pem")
	if err := os.WriteFile(certPath, cert.Certificate, 0600); err != nil {
		return "", "", "", fmt.Errorf("failed to save certificate: %w", err)
	}

	// Save private key
	keyPath = filepath.Join(domainDir, "key.pem")
	if err := os.WriteFile(keyPath, cert.PrivateKey, 0600); err != nil {
		return "", "", "", fmt.Errorf("failed to save private key: %w", err)
	}

	// Save issuer certificate (if present)
	issuerPath = ""
	if len(cert.IssuerCertificate) > 0 {
		issuerPath = filepath.Join(domainDir, "issuer.pem")
		if err := os.WriteFile(issuerPath, cert.IssuerCertificate, 0600); err != nil {
			return "", "", "", fmt.Errorf("failed to save issuer certificate: %w", err)
		}
	}

	return certPath, keyPath, issuerPath, nil
}

// GetCertificateDaysUntilExpiry returns days until certificate expires
func GetCertificateDaysUntilExpiry(notAfter time.Time) int {
	return int(time.Until(notAfter).Hours() / 24)
}

// NeedsRenewal checks if certificate needs renewal (within 30 days of expiry)
func NeedsRenewal(notAfter time.Time) bool {
	return GetCertificateDaysUntilExpiry(notAfter) <= 30
}

// Helper functions

func loadOrGenerateAccountKey(path string) (*ecdsa.PrivateKey, error) {
	// Try to load existing key
	if data, err := os.ReadFile(path); err == nil {
		key, err := certcrypto.ParsePEMPrivateKey(data)
		if err == nil {
			if ecdsaKey, ok := key.(*ecdsa.PrivateKey); ok {
				return ecdsaKey, nil
			}
		}
	}

	// Generate new key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Save key
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, keyPEM, 0600); err != nil {
		return nil, err
	}

	return key, nil
}
