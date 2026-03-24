-- Drop foreign key constraint for inbounds.tls_cert_id
ALTER TABLE inbounds
DROP CONSTRAINT IF EXISTS fk_inbounds_tls_cert;
