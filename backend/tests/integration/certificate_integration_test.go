package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/isolate-project/isolate-panel/tests/testutil"
)

// TestCertificateLifecycle_Integration tests the full certificate lifecycle:
// create → list → get → update status → delete
func TestCertificateLifecycle_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	// Create service without ACME (empty DNSProvider → acmeClient=nil)
	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Step 1: Verify empty state
	certs, err := svc.ListCertificates()
	require.NoError(t, err)
	assert.Empty(t, certs)

	// Step 2: Create certificate records directly (simulating ACME result)
	domainDir := filepath.Join(certDir, "integration-test.com")
	require.NoError(t, os.MkdirAll(domainDir, 0700))
	certPath := filepath.Join(domainDir, "cert.pem")
	keyPath := filepath.Join(domainDir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, []byte("cert-data"), 0600))
	require.NoError(t, os.WriteFile(keyPath, []byte("key-data"), 0600))

	cert1 := &models.Certificate{
		Domain:       "integration-test.com",
		IsWildcard:   false,
		CertPath:     certPath,
		KeyPath:      keyPath,
		CommonName:   "integration-test.com",
		Issuer:       "Test CA",
		NotBefore:    time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:     time.Now().Add(60 * 24 * time.Hour),
		AutoRenew:    true,
		Status:       models.CertificateStatusActive,
		ACMEProvider: "letsencrypt",
		DNSProvider:  "cloudflare",
	}
	require.NoError(t, db.Create(cert1).Error)

	// Create a second wildcard certificate
	wildcardDir := filepath.Join(certDir, "wildcard-test.com")
	require.NoError(t, os.MkdirAll(wildcardDir, 0700))
	wildcardCertPath := filepath.Join(wildcardDir, "cert.pem")
	wildcardKeyPath := filepath.Join(wildcardDir, "key.pem")
	require.NoError(t, os.WriteFile(wildcardCertPath, []byte("wildcard-cert"), 0600))
	require.NoError(t, os.WriteFile(wildcardKeyPath, []byte("wildcard-key"), 0600))

	cert2 := &models.Certificate{
		Domain:          "wildcard-test.com",
		IsWildcard:      true,
		CertPath:        wildcardCertPath,
		KeyPath:         wildcardKeyPath,
		CommonName:      "*.wildcard-test.com",
		SubjectAltNames: []string{"wildcard-test.com", "*.wildcard-test.com"},
		Issuer:          "Test CA",
		NotBefore:       time.Now().Add(-10 * 24 * time.Hour),
		NotAfter:        time.Now().Add(15 * 24 * time.Hour), // Expiring soon
		AutoRenew:       true,
		Status:          models.CertificateStatusActive,
		ACMEProvider:    "letsencrypt",
		DNSProvider:     "cloudflare",
	}
	require.NoError(t, db.Create(cert2).Error)

	// Step 3: List certificates
	certs, err = svc.ListCertificates()
	require.NoError(t, err)
	assert.Len(t, certs, 2)

	// Step 4: Get specific certificate
	found, err := svc.GetCertificate(cert1.ID)
	require.NoError(t, err)
	assert.Equal(t, "integration-test.com", found.Domain)
	assert.Equal(t, models.CertificateStatusActive, found.Status)
	assert.False(t, found.IsWildcard)

	// Step 5: Verify wildcard certificate
	wildcardFound, err := svc.GetCertificate(cert2.ID)
	require.NoError(t, err)
	assert.True(t, wildcardFound.IsWildcard)
	assert.Equal(t, "*.wildcard-test.com", wildcardFound.CommonName)

	// Step 6: Update status — cert2 should be "expiring" (15 days left < 30 threshold)
	svc.UpdateCertificateStatus(cert2)
	updated, err := svc.GetCertificate(cert2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CertificateStatusExpiring, updated.Status)

	// Step 7: Update status — cert1 should stay "active" (60 days left)
	svc.UpdateCertificateStatus(cert1)
	active, err := svc.GetCertificate(cert1.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CertificateStatusActive, active.Status)

	// Step 8: Delete certificate
	err = svc.DeleteCertificate(cert1.ID)
	require.NoError(t, err)

	// Verify deleted from DB
	_, err = svc.GetCertificate(cert1.ID)
	assert.Error(t, err)

	// Verify files deleted
	_, err = os.Stat(certPath)
	assert.True(t, os.IsNotExist(err))

	// Step 9: Remaining list should have 1 certificate
	certs, err = svc.ListCertificates()
	require.NoError(t, err)
	assert.Len(t, certs, 1)
	assert.Equal(t, "wildcard-test.com", certs[0].Domain)
}

// TestCertificateInboundBinding_Integration tests linking certificates to inbounds
func TestCertificateInboundBinding_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	testutil.SeedTestData(t, db)

	certDir := t.TempDir()

	// Create a certificate
	cert := &models.Certificate{
		Domain:     "binding-test.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "binding-test.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	// Create inbound with TLS cert binding
	certID := cert.ID
	core := testutil.GetTestCore(t, db, "xray")
	inbound := &models.Inbound{
		Name:          "tls-bound-inbound",
		Protocol:      "vless",
		CoreID:        core.ID,
		ListenAddress: "0.0.0.0",
		Port:          10443,
		TLSEnabled:    true,
		TLSCertID:     &certID,
		IsEnabled:     true,
	}
	require.NoError(t, db.Create(inbound).Error)

	// Verify the binding
	var loaded models.Inbound
	require.NoError(t, db.First(&loaded, inbound.ID).Error)
	require.NotNil(t, loaded.TLSCertID)
	assert.Equal(t, cert.ID, *loaded.TLSCertID)
	assert.True(t, loaded.TLSEnabled)

	// Clear TLS → cert should be clearable
	loaded.TLSEnabled = false
	loaded.TLSCertID = nil
	require.NoError(t, db.Save(&loaded).Error)

	var reloaded models.Inbound
	require.NoError(t, db.First(&reloaded, inbound.ID).Error)
	assert.Nil(t, reloaded.TLSCertID)
}
