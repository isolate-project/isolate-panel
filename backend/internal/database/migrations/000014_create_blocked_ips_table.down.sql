-- Drop blocked_ips table
DROP INDEX IF EXISTS idx_blocked_ips_expires_at;
DROP INDEX IF EXISTS idx_blocked_ips_ip_address;
DROP TABLE IF EXISTS blocked_ips;
