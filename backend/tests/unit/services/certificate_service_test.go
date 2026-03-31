package services_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"github.com/vovk4morkovk4/isolate-panel/internal/services"
	"github.com/vovk4morkovk4/isolate-panel/tests/testutil"
)

// newTestCertService creates a CertificateService with no ACME (acmeClient=nil)
// using the public constructor with an empty DNSProvider config.
func newTestCertService(t testing.TB) (*services.CertificateService, *testing.T) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{
		CertDir: certDir,
		// No DNSProvider → acmeClient will be nil
	})
	require.NoError(t, err)
	return svc, nil
}

func TestCertificateService_ListCertificates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Create test certificates directly in DB
	cert1 := &models.Certificate{
		Domain:     "example1.com",
		CertPath:   filepath.Join(certDir, "cert1.pem"),
		KeyPath:    filepath.Join(certDir, "key1.pem"),
		CommonName: "example1.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	cert2 := &models.Certificate{
		Domain:     "example2.com",
		CertPath:   filepath.Join(certDir, "cert2.pem"),
		KeyPath:    filepath.Join(certDir, "key2.pem"),
		CommonName: "example2.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(10 * 24 * time.Hour),
		Status:     models.CertificateStatusExpiring,
	}
	require.NoError(t, db.Create(cert1).Error)
	require.NoError(t, db.Create(cert2).Error)

	certs, err := svc.ListCertificates()
	require.NoError(t, err)
	assert.Len(t, certs, 2)
}

func TestCertificateService_GetCertificate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	cert := &models.Certificate{
		Domain:     "get-test.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "get-test.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	found, err := svc.GetCertificate(cert.ID)
	require.NoError(t, err)
	assert.Equal(t, "get-test.com", found.Domain)
	assert.Equal(t, models.CertificateStatusActive, found.Status)
}

func TestCertificateService_GetCertificate_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	_, err = svc.GetCertificate(99999)
	assert.Error(t, err)
}

func TestCertificateService_DeleteCertificate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Create cert files
	domainDir := filepath.Join(certDir, "delete-test.com")
	require.NoError(t, os.MkdirAll(domainDir, 0700))
	certPath := filepath.Join(domainDir, "cert.pem")
	keyPath := filepath.Join(domainDir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, []byte("cert-data"), 0600))
	require.NoError(t, os.WriteFile(keyPath, []byte("key-data"), 0600))

	cert := &models.Certificate{
		Domain:     "delete-test.com",
		CertPath:   certPath,
		KeyPath:    keyPath,
		CommonName: "delete-test.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	// Delete
	err = svc.DeleteCertificate(cert.ID)
	require.NoError(t, err)

	// Verify DB record is deleted
	_, err = svc.GetCertificate(cert.ID)
	assert.Error(t, err)

	// Verify files are deleted
	_, err = os.Stat(certPath)
	assert.True(t, os.IsNotExist(err))
}

func TestCertificateService_DeleteCertificate_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	err = svc.DeleteCertificate(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCertificateService_UpdateCertificateStatus_ActiveToExpiring(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Certificate expiring in 20 days (within 30-day renewal window)
	cert := &models.Certificate{
		Domain:     "expiring-test.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "expiring-test.com",
		NotBefore:  time.Now().Add(-60 * 24 * time.Hour),
		NotAfter:   time.Now().Add(20 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	svc.UpdateCertificateStatus(cert)

	updated, err := svc.GetCertificate(cert.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CertificateStatusExpiring, updated.Status)
}

func TestCertificateService_UpdateCertificateStatus_ExpiringToExpired(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Certificate already expired
	cert := &models.Certificate{
		Domain:     "expired-test.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "expired-test.com",
		NotBefore:  time.Now().Add(-120 * 24 * time.Hour),
		NotAfter:   time.Now().Add(-1 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	svc.UpdateCertificateStatus(cert)

	updated, err := svc.GetCertificate(cert.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CertificateStatusExpired, updated.Status)
}

func TestCertificateService_UpdateCertificateStatus_FarFuture(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Certificate valid for 60 more days — should stay active
	cert := &models.Certificate{
		Domain:     "far-future.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "far-future.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	svc.UpdateCertificateStatus(cert)

	updated, err := svc.GetCertificate(cert.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CertificateStatusActive, updated.Status)
}

func TestCertificateService_RequestCertificate_NoACMEClient(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	_, err = svc.RequestCertificate("example.com", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME client not initialized")
}

func TestCertificateService_RequestCertificate_DuplicateDomain(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Create existing certificate in DB
	existing := &models.Certificate{
		Domain:     "duplicate.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "duplicate.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(existing).Error)

	// Try to request certificate for same domain — should fail with duplicate error
	_, err = svc.RequestCertificate("duplicate.com", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCertificateService_RenewCertificate_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	_, err = svc.RenewCertificate(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCertificateService_RevokeCertificate_NoACME(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	cert := &models.Certificate{
		Domain:     "revoke-test.com",
		CertPath:   filepath.Join(certDir, "cert.pem"),
		KeyPath:    filepath.Join(certDir, "key.pem"),
		CommonName: "revoke-test.com",
		NotBefore:  time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:   time.Now().Add(60 * 24 * time.Hour),
		Status:     models.CertificateStatusActive,
	}
	require.NoError(t, db.Create(cert).Error)

	err = svc.RevokeCertificate(cert.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME client not initialized")
}

func TestCertificateService_Stop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)

	// Should not panic
	assert.NotPanics(t, func() {
		svc.Stop()
	})
}

func TestCertificateService_ListCertificates_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	certs, err := svc.ListCertificates()
	require.NoError(t, err)
	assert.Empty(t, certs)
}

func TestCertificateService_RequestWildcardCertificate_NoACME(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.TeardownTestDB(t, db)
	certDir := t.TempDir()

	svc, err := services.NewCertificateService(db, services.CertificateServiceConfig{CertDir: certDir})
	require.NoError(t, err)
	defer svc.Stop()

	// Request wildcard — will fail at ACME client check (nil), but verifies
	// that the code path for wildcard is reachable
	_, err = svc.RequestCertificate("wildcard-test.com", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ACME client not initialized")
}
