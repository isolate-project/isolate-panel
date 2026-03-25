-- Migration: 000027_add_performance_indexes
-- Purpose: Add composite indexes for better query performance
-- Date: 2026-03-25

-- Composite index for inbounds listing by core and status
CREATE INDEX IF NOT EXISTS idx_inbounds_core_enabled ON inbounds(core_id, is_enabled);

-- Composite index for traffic stats by user and time
CREATE INDEX IF NOT EXISTS idx_traffic_stats_user_time ON traffic_stats(user_id, recorded_at);

-- Composite index for active connections by user and inbound
CREATE INDEX IF NOT EXISTS idx_active_connections_user_inbound ON active_connections(user_id, inbound_id);

-- Composite index for notifications by status and time
CREATE INDEX IF NOT EXISTS idx_notifications_status_time ON notifications(status, created_at);

-- Composite index for subscription accesses by user and time
CREATE INDEX IF NOT EXISTS idx_subscription_accesses_user_time ON subscription_accesses(user_id, accessed_at);

-- Composite index for login attempts by IP and time (for rate limiting)
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_time ON login_attempts(ip_address, attempted_at);

-- Composite index for user inbound mapping
CREATE INDEX IF NOT EXISTS idx_user_inbound_mapping_user ON user_inbound_mapping(user_id);
CREATE INDEX IF NOT EXISTS idx_user_inbound_mapping_inbound ON user_inbound_mapping(inbound_id);

-- Index for settings key lookup (already unique, but explicit index helps)
CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_key_unique ON settings(key);

-- Index for cores by name and status
CREATE INDEX IF NOT EXISTS idx_cores_name_enabled ON cores(name, is_enabled);

-- Analyze tables to update query planner statistics
ANALYZE inbounds;
ANALYZE traffic_stats;
ANALYZE active_connections;
ANALYZE notifications;
ANALYZE subscription_accesses;
ANALYZE login_attempts;
ANALYZE user_inbound_mapping;
ANALYZE settings;
ANALYZE cores;
