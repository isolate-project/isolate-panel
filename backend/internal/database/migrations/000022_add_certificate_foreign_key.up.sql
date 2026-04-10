-- No-op: foreign key inbounds.tls_cert_id -> certificates.id
-- already defined in 000004_create_inbounds_table.up.sql.
-- SQLite does not support ALTER TABLE ... ADD CONSTRAINT.
SELECT 1;
