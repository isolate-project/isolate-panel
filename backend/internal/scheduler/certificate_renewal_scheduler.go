package scheduler

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/services"
)

// CertificateRenewalScheduler runs automatic certificate renewal on a cron schedule.
type CertificateRenewalScheduler struct {
	certService *services.CertificateService
	cron        *cron.Cron
	mu          sync.Mutex
	jobEntry    cron.EntryID
}

// NewCertificateRenewalScheduler creates a new CertificateRenewalScheduler.
func NewCertificateRenewalScheduler(certService *services.CertificateService) *CertificateRenewalScheduler {
	return &CertificateRenewalScheduler{
		certService: certService,
		cron:        cron.New(),
	}
}

// Initialize starts the cron runner with a default schedule (daily at 3am).
func (s *CertificateRenewalScheduler) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Default schedule: daily at 3am
	cronExpr := "0 0 3 * * *" // 3:00 AM every day

	entryID, err := s.cron.AddFunc(cronExpr, s.runRenewal)
	if err != nil {
		return err
	}

	s.jobEntry = entryID
	s.cron.Start()
	return nil
}

// runRenewal is called by the cron runner and performs the certificate renewal check.
func (s *CertificateRenewalScheduler) runRenewal() {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error().Interface("panic", r).Msg("Scheduled certificate renewal panicked, recovered")
		}
	}()

	logger.Log.Info().Str("scheduler", "certificate_renewal").Msg("Starting certificate renewal check")

	certs, err := s.certService.ListCertificates()
	if err != nil {
		logger.Log.Error().Err(err).Str("scheduler", "certificate_renewal").Msg("Failed to list certificates")
		return
	}

	renewedCount := 0
	for _, cert := range certs {
		if cert.AutoRenew && (cert.Status == "active" || cert.Status == "expiring") {
			daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
			if daysUntilExpiry <= 30 {
				logger.Log.Info().
					Str("domain", cert.Domain).
					Str("status", string(cert.Status)).
					Time("not_after", cert.NotAfter).
					Msg("Certificate needs renewal")

				_, err := s.certService.RenewCertificate(cert.ID)
				if err != nil {
					logger.Log.Error().
						Err(err).
						Uint("cert_id", cert.ID).
						Str("domain", cert.Domain).
						Msg("Failed to renew certificate")
				} else {
					logger.Log.Info().
						Uint("cert_id", cert.ID).
						Str("domain", cert.Domain).
						Msg("Certificate renewed successfully")
					renewedCount++
				}
			}
		}
	}

	logger.Log.Info().
		Str("scheduler", "certificate_renewal").
		Int("total_checked", len(certs)).
		Int("renewed", renewedCount).
		Msg("Certificate renewal check completed")
}

// Stop gracefully stops the cron runner.
func (s *CertificateRenewalScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cron.Stop()
}