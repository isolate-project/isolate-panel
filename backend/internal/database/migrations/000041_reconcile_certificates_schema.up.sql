-- Reconcile certificates table schema with Certificate model
-- This migration adds columns the model expects and removes obsolete columns

-- Add columns that the model expects but don't exist in the DB
-- SQLite doesn't support IF NOT EXISTS with ALTER TABLE ADD COLUMN, so we use a try-catch approach
ALTER TABLE certificates ADD COLUMN is_wildcard BOOLEAN DEFAULT false;
ALTER TABLE certificates ADD COLUMN issuer_path VARCHAR(255);
ALTER TABLE certificates ADD COLUMN common_name VARCHAR(255);
ALTER TABLE certificates ADD COLUMN subject_alt_names TEXT;
ALTER TABLE certificates ADD COLUMN issuer VARCHAR(255);
ALTER TABLE certificates ADD COLUMN not_before DATETIME;
ALTER TABLE certificates ADD COLUMN not_after DATETIME;
ALTER TABLE certificates ADD COLUMN last_renewed_at DATETIME;
ALTER TABLE certificates ADD COLUMN status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE certificates ADD COLUMN status_reason TEXT;
ALTER TABLE certificates ADD COLUMN dns_provider VARCHAR(50);
ALTER TABLE certificates ADD COLUMN created_by INTEGER;

-- Drop obsolete columns that exist in DB but not in the model
-- SQLite doesn't support IF EXISTS with ALTER TABLE DROP COLUMN, so we use a try-catch approach
-- Drop indexes that reference columns we're about to drop
DROP INDEX IF EXISTS idx_certificates_expires_at;

ALTER TABLE certificates DROP COLUMN type;
ALTER TABLE certificates DROP COLUMN acme_email;
ALTER TABLE certificates DROP COLUMN acme_challenge_type;
ALTER TABLE certificates DROP COLUMN is_valid;
ALTER TABLE certificates DROP COLUMN issued_at;
ALTER TABLE certificates DROP COLUMN expires_at;

-- Create index on created_by for foreign key relationship
CREATE INDEX IF NOT EXISTS idx_certificates_created_by ON certificates(created_by);
