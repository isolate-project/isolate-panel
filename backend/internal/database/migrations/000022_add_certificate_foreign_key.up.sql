-- Add foreign key constraint for inbounds.tls_cert_id referencing certificates.id
ALTER TABLE inbounds
ADD CONSTRAINT fk_inbounds_tls_cert
FOREIGN KEY (tls_cert_id) REFERENCES certificates(id)
ON DELETE SET NULL;
