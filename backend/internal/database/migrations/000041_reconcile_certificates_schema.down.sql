-- Rollback certificate schema reconciliation
-- WARNING: This restores column structure but DATA IS LOST

DROP INDEX IF EXISTS idx_certificates_created_by;

-- Restore dropped columns (data will be lost)
ALTER TABLE certificates ADD COLUMN type TEXT NOT NULL DEFAULT '';
ALTER TABLE certificates ADD COLUMN acme_email TEXT NOT NULL DEFAULT '';
ALTER TABLE certificates ADD COLUMN acme_challenge_type TEXT NOT NULL DEFAULT '';
ALTER TABLE certificates ADD COLUMN is_valid BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE certificates ADD COLUMN issued_at DATETIME;
ALTER TABLE certificates ADD COLUMN expires_at DATETIME;
