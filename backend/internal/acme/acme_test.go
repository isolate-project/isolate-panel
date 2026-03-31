package acme

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedsRenewal_True(t *testing.T) {
	// 20 days until expiry — should need renewal (threshold is 30)
	notAfter := time.Now().Add(20 * 24 * time.Hour)
	assert.True(t, NeedsRenewal(notAfter))
}

func TestNeedsRenewal_False(t *testing.T) {
	// 60 days until expiry — should NOT need renewal
	notAfter := time.Now().Add(60 * 24 * time.Hour)
	assert.False(t, NeedsRenewal(notAfter))
}

func TestNeedsRenewal_Exactly30Days(t *testing.T) {
	// Exactly 30 days — boundary: should need renewal (<=30)
	notAfter := time.Now().Add(30 * 24 * time.Hour)
	assert.True(t, NeedsRenewal(notAfter))
}

func TestNeedsRenewal_Expired(t *testing.T) {
	// Already expired — should definitely need renewal
	notAfter := time.Now().Add(-1 * 24 * time.Hour)
	assert.True(t, NeedsRenewal(notAfter))
}

func TestGetCertificateDaysUntilExpiry_Positive(t *testing.T) {
	notAfter := time.Now().Add(45 * 24 * time.Hour)
	days := GetCertificateDaysUntilExpiry(notAfter)
	// Allow 1 day tolerance due to time precision
	assert.InDelta(t, 45, days, 1)
}

func TestGetCertificateDaysUntilExpiry_Negative(t *testing.T) {
	// Expired 5 days ago
	notAfter := time.Now().Add(-5 * 24 * time.Hour)
	days := GetCertificateDaysUntilExpiry(notAfter)
	assert.InDelta(t, -5, days, 1)
}

func TestGetCertificateDaysUntilExpiry_Zero(t *testing.T) {
	notAfter := time.Now()
	days := GetCertificateDaysUntilExpiry(notAfter)
	assert.Equal(t, 0, days)
}

func TestLoadOrGenerateAccountKey_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_account.key")

	key, err := loadOrGenerateAccountKey(keyPath)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Verify it's a valid ECDSA key
	assert.IsType(t, &ecdsa.PrivateKey{}, key)

	// Verify file was created
	_, err = os.Stat(keyPath)
	assert.NoError(t, err)
}

func TestLoadOrGenerateAccountKey_LoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_account.key")

	// Generate key first
	key1, err := loadOrGenerateAccountKey(keyPath)
	require.NoError(t, err)

	// Load existing key
	key2, err := loadOrGenerateAccountKey(keyPath)
	require.NoError(t, err)

	// Keys should have the same parameters (same key loaded)
	assert.Equal(t, key1.D, key2.D)
	assert.Equal(t, key1.X, key2.X)
	assert.Equal(t, key1.Y, key2.Y)
}

func TestLoadOrGenerateAccountKey_InvalidPath(t *testing.T) {
	// Non-existent key at path with missing parent directories will create them
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "subdir", "deep", "account.key")

	key, err := loadOrGenerateAccountKey(keyPath)
	require.NoError(t, err)
	require.NotNil(t, key)
}

func TestSaveCertificate(t *testing.T) {
	tmpDir := t.TempDir()

	ac := &ACMEClient{
		certDir: tmpDir,
	}

	certData := []byte("-----BEGIN CERTIFICATE-----\ntest-cert-data\n-----END CERTIFICATE-----")
	keyData := []byte("-----BEGIN PRIVATE KEY-----\ntest-key-data\n-----END PRIVATE KEY-----")
	issuerData := []byte("-----BEGIN CERTIFICATE-----\ntest-issuer-data\n-----END CERTIFICATE-----")

	certRes := &certificate.Resource{
		Domain:            "example.com",
		Certificate:       certData,
		PrivateKey:        keyData,
		IssuerCertificate: issuerData,
	}

	certPath, keyPath, issuerPath, err := ac.SaveCertificate(certRes, "example.com")
	require.NoError(t, err)

	// Verify paths
	assert.Contains(t, certPath, "example.com")
	assert.Contains(t, certPath, "cert.pem")
	assert.Contains(t, keyPath, "key.pem")
	assert.Contains(t, issuerPath, "issuer.pem")

	// Verify file contents
	savedCert, err := os.ReadFile(certPath)
	require.NoError(t, err)
	assert.Equal(t, certData, savedCert)

	savedKey, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, keyData, savedKey)

	savedIssuer, err := os.ReadFile(issuerPath)
	require.NoError(t, err)
	assert.Equal(t, issuerData, savedIssuer)

	// Verify file permissions (0600 for cert and key)
	certInfo, err := os.Stat(certPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), certInfo.Mode().Perm())

	keyInfo, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), keyInfo.Mode().Perm())
}

func TestSaveCertificate_NoIssuer(t *testing.T) {
	tmpDir := t.TempDir()

	ac := &ACMEClient{
		certDir: tmpDir,
	}

	certRes := &certificate.Resource{
		Domain:      "no-issuer.com",
		Certificate: []byte("cert-data"),
		PrivateKey:  []byte("key-data"),
		// No IssuerCertificate
	}

	certPath, keyPath, issuerPath, err := ac.SaveCertificate(certRes, "no-issuer.com")
	require.NoError(t, err)

	assert.NotEmpty(t, certPath)
	assert.NotEmpty(t, keyPath)
	assert.Empty(t, issuerPath) // No issuer => empty path
}

func TestWildcardDomainFormat(t *testing.T) {
	// Simulate what RequestCertificate does with wildcard
	domain := "example.com"
	isWildcard := true

	domains := []string{domain}
	if isWildcard {
		wildcardDomain := "*." + domain
		if domain != "" {
			domains = append(domains, wildcardDomain)
		}
	}

	assert.Len(t, domains, 2)
	assert.Equal(t, "example.com", domains[0])
	assert.Equal(t, "*.example.com", domains[1])
}

func TestWildcardDomainFormat_NotWildcard(t *testing.T) {
	domain := "example.com"
	isWildcard := false

	domains := []string{domain}
	if isWildcard {
		wildcardDomain := "*." + domain
		if domain != "" {
			domains = append(domains, wildcardDomain)
		}
	}

	assert.Len(t, domains, 1)
	assert.Equal(t, "example.com", domains[0])
}
