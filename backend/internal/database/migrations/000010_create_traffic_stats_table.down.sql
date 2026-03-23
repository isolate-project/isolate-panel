-- Drop traffic_stats table
DROP INDEX IF EXISTS idx_traffic_stats_recorded_at;
DROP INDEX IF EXISTS idx_traffic_stats_user_id;
DROP TABLE IF EXISTS traffic_stats;
