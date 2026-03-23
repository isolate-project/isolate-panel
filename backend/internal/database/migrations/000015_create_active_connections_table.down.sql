-- Drop active_connections table
DROP INDEX IF EXISTS idx_active_connections_inbound_id;
DROP INDEX IF EXISTS idx_active_connections_user_id;
DROP TABLE IF EXISTS active_connections;
