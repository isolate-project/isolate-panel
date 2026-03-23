-- Drop certificates table
DROP INDEX IF EXISTS idx_certificates_expires_at;
DROP INDEX IF EXISTS idx_certificates_domain;
DROP TABLE IF EXISTS certificates;
