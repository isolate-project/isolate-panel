-- Migration: 000027_add_performance_indexes (DOWN)
-- Purpose: Remove performance indexes
-- Date: 2026-03-25

-- Drop composite indexes
DROP INDEX IF EXISTS idx_inbounds_core_enabled;
DROP INDEX IF EXISTS idx_traffic_stats_user_time;
DROP INDEX IF EXISTS idx_active_connections_user_inbound;
DROP INDEX IF EXISTS idx_notifications_status_time;
DROP INDEX IF EXISTS idx_subscription_accesses_user_time;
DROP INDEX IF EXISTS idx_login_attempts_ip_time;
DROP INDEX IF EXISTS idx_user_inbound_mapping_user;
DROP INDEX IF EXISTS idx_user_inbound_mapping_inbound;
DROP INDEX IF EXISTS idx_settings_key_unique;
DROP INDEX IF EXISTS idx_cores_name_enabled;
