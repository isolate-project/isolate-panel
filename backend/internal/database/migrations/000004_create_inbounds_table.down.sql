-- Drop inbounds table
DROP INDEX IF EXISTS idx_inbounds_core_id;
DROP INDEX IF EXISTS idx_inbounds_port;
DROP INDEX IF EXISTS idx_inbounds_protocol;
DROP TABLE IF EXISTS inbounds;
