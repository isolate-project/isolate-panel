package api

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/acme"
	"github.com/isolate-project/isolate-panel/internal/models"
	"github.com/isolate-project/isolate-panel/internal/services"
	"gorm.io/gorm"
)

type CertificatesHandler struct {
	certService *services.CertificateService
	db          *gorm.DB
}

func NewCertificatesHandler(certService *services.CertificateService, db *gorm.DB) *CertificatesHandler {
	return &CertificatesHandler{
		certService: certService,
		db:          db,
	}
}

// CertificateRequest represents a certificate request
type CertificateRequest struct {
	Domain     string `json:"domain" validate:"required"`
	IsWildcard bool   `json:"is_wildcard"`
	Email      string `json:"email"` // Optional, uses service default if not provided
}

// ListCertificates returns all certificates
//
// @Summary      List certificates
// @Description  Returns all SSL/TLS certificates with current status
// @Tags         certificates
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /certificates [get]
// @Security     BearerAuth
func (h *CertificatesHandler) ListCertificates(c fiber.Ctx) error {
	certs, err := h.certService.ListCertificates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Update status for each certificate
	for i := range certs {
		h.certService.UpdateCertificateStatus(&certs[i])
	}

	return c.JSON(fiber.Map{
		"certificates": certs,
		"total":        len(certs),
	})
}

// ListCertificatesDropdown returns a simplified list for dropdown selection
//
// @Summary      Certificates dropdown
// @Description  Returns active/expiring certificates formatted for UI dropdown selectors
// @Tags         certificates
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /certificates/dropdown [get]
// @Security     BearerAuth
func (h *CertificatesHandler) ListCertificatesDropdown(c fiber.Ctx) error {
	certs, err := h.certService.ListCertificates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Filter only active/expiring certificates
	type CertOption struct {
		ID     uint   `json:"id"`
		Domain string `json:"domain"`
		Label  string `json:"label"` // e.g., "example.com (expires in 30 days)"
	}

	var options []CertOption
	for _, cert := range certs {
		if cert.Status == models.CertificateStatusActive || cert.Status == models.CertificateStatusExpiring {
			days := acme.GetCertificateDaysUntilExpiry(cert.NotAfter)
			var label string
			if days <= 0 {
				label = fmt.Sprintf("%s (expired)", cert.Domain)
			} else if days <= 30 {
				label = fmt.Sprintf("%s (expires in %d days) ⚠️", cert.Domain, days)
			} else {
				label = fmt.Sprintf("%s (valid for %d days)", cert.Domain, days)
			}
			if cert.IsWildcard {
				label = "*." + cert.Domain + " (wildcard)"
			}
			options = append(options, CertOption{
				ID:     cert.ID,
				Domain: cert.Domain,
				Label:  label,
			})
		}
	}

	return c.JSON(fiber.Map{
		"options": options,
	})
}

// GetCertificate returns a single certificate
//
// @Summary      Get certificate
// @Description  Returns a single certificate by ID with current status
// @Tags         certificates
// @Produce      json
// @Param        id   path  int  true  "Certificate ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /certificates/{id} [get]
// @Security     BearerAuth
func (h *CertificatesHandler) GetCertificate(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid certificate ID",
		})
	}

	cert, err := h.certService.GetCertificate(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Certificate not found",
		})
	}

	h.certService.UpdateCertificateStatus(cert)

	return c.JSON(fiber.Map{
		"certificate": cert,
	})
}

// RequestCertificate requests a new certificate via ACME
//
// @Summary      Request certificate
// @Description  Request a new SSL/TLS certificate via Let's Encrypt ACME
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        body  body  CertificateRequest  true  "Domain and options"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /certificates [post]
// @Security     BearerAuth
func (h *CertificatesHandler) RequestCertificate(c fiber.Ctx) error {
	var req CertificateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Domain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Domain is required",
		})
	}

	cert, err := h.certService.RequestCertificate(req.Domain, req.IsWildcard)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "Certificate requested successfully",
		"certificate": cert,
	})
}

// RenewCertificate renews an existing certificate
//
// @Summary      Renew certificate
// @Description  Force-renew a certificate via ACME (even if not yet expired)
// @Tags         certificates
// @Produce      json
// @Param        id   path  int  true  "Certificate ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /certificates/{id}/renew [post]
// @Security     BearerAuth
func (h *CertificatesHandler) RenewCertificate(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid certificate ID",
		})
	}

	cert, err := h.certService.RenewCertificate(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Certificate renewed successfully",
		"certificate": cert,
	})
}

// RevokeCertificate revokes a certificate
//
// @Summary      Revoke certificate
// @Description  Revoke a certificate via ACME and mark it as revoked
// @Tags         certificates
// @Produce      json
// @Param        id   path  int  true  "Certificate ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /certificates/{id}/revoke [post]
// @Security     BearerAuth
func (h *CertificatesHandler) RevokeCertificate(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid certificate ID",
		})
	}

	if err := h.certService.RevokeCertificate(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Certificate revoked successfully",
	})
}

// DeleteCertificate deletes a certificate
//
// @Summary      Delete certificate
// @Description  Delete a certificate record and its files
// @Tags         certificates
// @Produce      json
// @Param        id   path  int  true  "Certificate ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /certificates/{id} [delete]
// @Security     BearerAuth
func (h *CertificatesHandler) DeleteCertificate(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid certificate ID",
		})
	}

	if err := h.certService.DeleteCertificate(uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Certificate deleted successfully",
	})
}

// UploadCertificateRequest represents a manual certificate upload
type UploadCertificateRequest struct {
	Certificate string `json:"certificate" validate:"required"`
	PrivateKey  string `json:"private_key" validate:"required"`
	Issuer      string `json:"issuer"`
	Domain      string `json:"domain" validate:"required"`
	IsWildcard  bool   `json:"is_wildcard"`
}

// UploadCertificate uploads a certificate manually (PEM format)
//
// @Summary      Upload certificate
// @Description  Upload a manually obtained certificate in PEM format
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        body  body  UploadCertificateRequest  true  "Certificate and private key in PEM format"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /certificates/upload [post]
// @Security     BearerAuth
func (h *CertificatesHandler) UploadCertificate(c fiber.Ctx) error {
	var req UploadCertificateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate PEM format
	certBlock, err := parsePEM(req.Certificate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid certificate format: " + err.Error(),
		})
	}

	if _, err := parsePEM(req.PrivateKey); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid private key format: " + err.Error(),
		})
	}

	// Parse certificate to extract metadata
	x509Cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse certificate: " + err.Error(),
		})
	}

	// Create certificate directory
	certDir := "/etc/isolate-panel/certs/" + req.Domain
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create certificate directory",
		})
	}

	// Save certificate files
	certPath := certDir + "/cert.pem"
	if err := os.WriteFile(certPath, []byte(req.Certificate), 0600); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save certificate",
		})
	}

	keyPath := certDir + "/key.pem"
	if err := os.WriteFile(keyPath, []byte(req.PrivateKey), 0600); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save private key",
		})
	}

	issuerPath := ""
	if req.Issuer != "" {
		issuerPath = certDir + "/issuer.pem"
		if err := os.WriteFile(issuerPath, []byte(req.Issuer), 0600); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save issuer certificate",
			})
		}
	}

	// Create certificate record
	cert := &models.Certificate{
		Domain:          req.Domain,
		IsWildcard:      req.IsWildcard,
		CertPath:        certPath,
		KeyPath:         keyPath,
		IssuerPath:      issuerPath,
		CommonName:      x509Cert.Subject.CommonName,
		SubjectAltNames: x509Cert.DNSNames,
		Issuer:          x509Cert.Issuer.CommonName,
		NotBefore:       x509Cert.NotBefore,
		NotAfter:        x509Cert.NotAfter,
		AutoRenew:       false, // Manual certificates don't auto-renew
		Status:          models.CertificateStatusActive,
		ACMEProvider:    "manual",
	}

	if err := h.db.Create(cert).Error; err != nil {
		os.RemoveAll(certDir)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save certificate record: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "Certificate uploaded successfully",
		"certificate": cert,
	})
}

// Helper functions

func parsePEM(data string) (*pem.Block, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	return block, nil
}
