package api

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupCertsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=memory"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Certificate{},
	))
	return db
}

func setupCertsApp(t *testing.T) (*fiber.App, *services.CertificateService, *gorm.DB) {
	t.Helper()
	db := setupCertsTestDB(t)

	tmpDir, err := os.MkdirTemp("", "certs-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	cfg := services.CertificateServiceConfig{
		CertDir:     tmpDir,
		Email:       "test@example.com",
		DNSProvider: "", // disables ACME
	}
	certSvc, err := services.NewCertificateService(db, cfg)
	require.NoError(t, err)

	handler := NewCertificatesHandler(certSvc, db)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("admin_id", uint(1))
		return c.Next()
	})

	certs := app.Group("/certificates")
	certs.Get("/", handler.ListCertificates)
	certs.Get("/dropdown", handler.ListCertificatesDropdown)
	certs.Get("/:id", handler.GetCertificate)
	certs.Post("/request", handler.RequestCertificate)
	certs.Post("/upload", handler.UploadCertificate)
	certs.Post("/:id/renew", handler.RenewCertificate)
	certs.Post("/:id/revoke", handler.RevokeCertificate)
	certs.Delete("/:id", handler.DeleteCertificate)

	return app, certSvc, db
}

func TestCertificatesHandler_ListCertificates(t *testing.T) {
	app, _, db := setupCertsApp(t)

	db.Create(&models.Certificate{
		Domain:   "example_" + t.Name() + ".com",
		Status:   models.CertificateStatusActive,
		NotAfter: time.Now().AddDate(0, 1, 0),
	})

	req, _ := http.NewRequest(http.MethodGet, "/certificates", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotNil(t, result["certificates"])
}

func TestCertificatesHandler_ListCertificatesDropdown(t *testing.T) {
	app, _, db := setupCertsApp(t)

	db.Create(&models.Certificate{
		Domain:   "example_" + t.Name() + ".com",
		Status:   models.CertificateStatusActive,
		NotAfter: time.Now().AddDate(0, 1, 0),
	})

	req, _ := http.NewRequest(http.MethodGet, "/certificates/dropdown", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotNil(t, result["options"])
}

func TestCertificatesHandler_GetCertificate(t *testing.T) {
	app, _, db := setupCertsApp(t)

	cert := &models.Certificate{
		Domain:   "example_" + t.Name() + ".com",
		Status:   models.CertificateStatusActive,
		NotAfter: time.Now().AddDate(0, 1, 0),
	}
	db.Create(cert)

	req, _ := http.NewRequest(http.MethodGet, "/certificates/"+uint2str(cert.ID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCertificatesHandler_DeleteCertificate(t *testing.T) {
	app, _, db := setupCertsApp(t)

	cert := &models.Certificate{
		Domain:   "example_" + t.Name() + ".com",
		Status:   models.CertificateStatusActive,
		NotAfter: time.Now().AddDate(0, 1, 0),
		CertPath: "/tmp/dummy_" + t.Name() + "/cert.pem",
	}
	db.Create(cert)

	req, _ := http.NewRequest(http.MethodDelete, "/certificates/"+uint2str(cert.ID), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
